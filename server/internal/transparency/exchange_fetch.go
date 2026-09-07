package transparency

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultFetchMaxBytes     = int64(10 << 20)
	defaultFetchTimeout      = 30 * time.Second
	defaultFetchMaxRedirects = 3
)

// FetchPolicy bounds external evidence fetches. A nil policy means fetching
// is disabled: the RPC fails closed instead of fetching unbounded content.
type FetchPolicy struct {
	Enabled      bool
	MaxBytes     int64
	Timeout      time.Duration
	AllowedHosts []string
	MaxRedirects int
}

func (p *FetchPolicy) normalized() *FetchPolicy {
	if p == nil {
		// A nil policy is the fail-closed default: fetching stays disabled
		// while the remaining bounds keep sane values for diagnostics.
		return &FetchPolicy{Enabled: false, MaxBytes: defaultFetchMaxBytes, Timeout: defaultFetchTimeout, MaxRedirects: defaultFetchMaxRedirects}
	}
	out := &FetchPolicy{Enabled: p.Enabled, AllowedHosts: p.AllowedHosts}
	if p.MaxBytes > 0 {
		out.MaxBytes = p.MaxBytes
	} else {
		out.MaxBytes = defaultFetchMaxBytes
	}
	if p.Timeout > 0 {
		out.Timeout = p.Timeout
	} else {
		out.Timeout = defaultFetchTimeout
	}
	if p.MaxRedirects >= 0 {
		out.MaxRedirects = p.MaxRedirects
	} else {
		out.MaxRedirects = defaultFetchMaxRedirects
	}
	return out
}

// hostAllowed reports whether the policy admits host. An empty allowlist
// admits every host; the https and size bounds still apply.
func (p *FetchPolicy) hostAllowed(host string) bool {
	if len(p.AllowedHosts) == 0 {
		return true
	}
	for _, allowed := range p.AllowedHosts {
		if allowed == host {
			return true
		}
	}
	return false
}

func (p *FetchPolicy) validateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url is not parseable")
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("only https URLs are fetchable")
	}
	if parsed.Host == "" {
		return fmt.Errorf("url host is required")
	}
	if !p.hostAllowed(parsed.Hostname()) {
		return fmt.Errorf("host %q is not in the fetch allowlist", parsed.Hostname())
	}
	return nil
}

// FetchExternalEvidence retrieves an external artifact over HTTPS under the
// deployment's bounded fetch policy, stores it as content-addressed evidence,
// and appends a fetch audit event. Policy denials and fetch failures are
// recorded as audit events too: denials are auditable, not invisible.
func (s *ExchangeServer) FetchExternalEvidence(ctx context.Context, req *servicesv1.FetchExternalEvidenceRequest) (*servicesv1.FetchExternalEvidenceResponse, error) {
	policy := s.fetchPolicy.normalized()
	start := time.Now().UTC()

	if !policy.Enabled {
		s.recordFetch(ctx, req.GetUrl(), "", 0, "policy_denied", "external fetching is disabled by policy", start)
		return nil, status.Error(codes.FailedPrecondition, "external evidence fetching is disabled by policy")
	}
	if err := policy.validateURL(req.GetUrl()); err != nil {
		s.recordFetch(ctx, req.GetUrl(), "", 0, "policy_denied", err.Error(), start)
		return nil, status.Errorf(codes.InvalidArgument, "fetch policy violation: %v", err)
	}

	blob, err := s.fetchBounded(ctx, req.GetUrl(), policy, s.fetchClient(policy))
	if err != nil {
		s.recordFetch(ctx, req.GetUrl(), "", 0, "fetch_failed", err.Error(), start)
		return nil, status.Errorf(codes.Unavailable, "external fetch failed: %v", err)
	}

	digest := digestFor(blob)
	existing, err := s.store.GetEvidenceByDigest(ctx, digest)
	if err == nil && existing != nil {
		// Content-addressed idempotency: the same bytes are already evidence.
		s.recordFetch(ctx, req.GetUrl(), digest, int64(len(blob)), "fetched", "already present", start)
		return &servicesv1.FetchExternalEvidenceResponse{
			Evidence: evidenceToProto(existing),
			Stored:   true,
			Audit: &servicesv1.EvidenceFetchAudit{
				Url:        req.GetUrl(),
				Digest:     digest,
				SizeBytes:  int64(len(blob)),
				Outcome:    "fetched",
				Detail:     "already present",
				FetchedAt:  timestamppb.New(start),
				DurationMs: time.Since(start).Milliseconds(),
			},
		}, nil
	}

	ev, err := s.storeFetchedEvidence(ctx, req, blob, digest, start)
	if err != nil {
		s.recordFetch(ctx, req.GetUrl(), digest, int64(len(blob)), "fetch_failed", err.Error(), start)
		return nil, status.Errorf(codes.Internal, "store fetched evidence: %v", err)
	}

	s.recordFetch(ctx, req.GetUrl(), digest, int64(len(blob)), "fetched", "", start)
	return &servicesv1.FetchExternalEvidenceResponse{
		Evidence: evidenceToProto(ev),
		Stored:   true,
		Audit: &servicesv1.EvidenceFetchAudit{
			Url:        req.GetUrl(),
			Digest:     digest,
			SizeBytes:  int64(len(blob)),
			Outcome:    "fetched",
			FetchedAt:  timestamppb.New(start),
			DurationMs: time.Since(start).Milliseconds(),
		},
	}, nil
}

// ListFetchEvents returns recent fetch audit events, newest first.
func (s *ExchangeServer) ListFetchEvents(ctx context.Context, req *servicesv1.ListFetchEventsRequest) (*servicesv1.ListFetchEventsResponse, error) {
	events, err := s.store.ListFetchEvents(ctx, int(req.GetPageSize()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list fetch events: %v", err)
	}
	out := make([]*servicesv1.EvidenceFetchAudit, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.toProto())
	}
	return &servicesv1.ListFetchEventsResponse{Events: out}, nil
}

// newFetchClient builds the HTTP client the fetcher uses in production:
// policy timeout and a bounded redirect chain that never leaves https.
func newFetchClient(policy *FetchPolicy) *http.Client {
	return &http.Client{
		Timeout: policy.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > policy.MaxRedirects {
				return fmt.Errorf("exceeded %d redirects", policy.MaxRedirects)
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("redirect left the https scheme")
			}
			return nil
		},
	}
}

// fetchClient returns the HTTP client for one fetch. The override seam exists
// so tests can supply a transport that trusts test certificates; production
// always uses newFetchClient.
func (s *ExchangeServer) fetchClient(policy *FetchPolicy) *http.Client {
	if s.fetchClientFn != nil {
		return s.fetchClientFn(policy)
	}
	return newFetchClient(policy)
}

// fetchBounded performs the HTTP GET with the policy's bounds: hard byte cap
// (over-limit is an error, never silent truncation), timeout, and a bounded
// redirect chain that never leaves the https scheme. The client is a
// parameter so tests can supply a transport that trusts test certificates.
func (s *ExchangeServer) fetchBounded(ctx context.Context, raw string, policy *FetchPolicy, client *http.Client) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, policy.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if policy.MaxBytes > 0 && resp.ContentLength > policy.MaxBytes {
		return nil, fmt.Errorf("content length %d exceeds policy maximum %d", resp.ContentLength, policy.MaxBytes)
	}

	blob, err := readBounded(resp.Body, policy.MaxBytes)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

// readBounded reads at most max bytes; one extra byte proves truncation and
// fails the fetch instead of storing silently truncated evidence.
func readBounded(r io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(r, max+1)
	blob, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(blob)) > max {
		return nil, fmt.Errorf("content exceeds policy maximum of %d bytes", max)
	}
	return blob, nil
}

func (s *ExchangeServer) storeFetchedEvidence(ctx context.Context, req *servicesv1.FetchExternalEvidenceRequest, blob []byte, digest string, start time.Time) (*Evidence, error) {
	id := req.GetEvidenceId()
	if strings.TrimSpace(id) == "" {
		id = fmt.Sprintf("ev-fetch-%s", digest[len("sha256:"):len("sha256:")+16])
	}
	now := time.Now().UTC()
	ev := &Evidence{
		ID:          id,
		MediaType:   req.GetMediaType(),
		BomKind:     req.GetBomKind(),
		Digest:      digest,
		SizeBytes:   int64(len(blob)),
		StorageJSON: toJSON(map[string]interface{}{"uris": []string{req.GetUrl()}, "fetch_policy": "bounded"}),
		CreatedAt:   now,
		ValidFrom:   now,
		Blob:        append([]byte(nil), blob...),
	}
	if ev.MediaType == "" {
		ev.MediaType = "application/octet-stream"
	}
	if ev.BomKind == "" {
		ev.BomKind = "external"
	}
	if err := s.store.CreateEvidence(ctx, ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (s *ExchangeServer) recordFetch(ctx context.Context, rawURL, digest string, size int64, outcome, detail string, start time.Time) {
	_ = s.store.RecordFetchEvent(ctx, FetchEvent{
		URL:        rawURL,
		Digest:     digest,
		SizeBytes:  size,
		Outcome:    outcome,
		Detail:     detail,
		FetchedAt:  time.Now().UTC(),
		DurationMS: time.Since(start).Milliseconds(),
	})
}
