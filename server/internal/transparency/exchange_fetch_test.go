package transparency

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newFetchFixture wires an exchange server with an enabled fetch policy and
// an HTTPS test server whose certificate the fetcher will trust.
func newFetchFixture(t *testing.T, handler http.HandlerFunc, policy ...*FetchPolicy) (*ExchangeServer, *httptest.Server) {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	var pol *FetchPolicy
	if len(policy) > 0 {
		pol = policy[0]
	}
	if pol == nil {
		pol = &FetchPolicy{Enabled: true, MaxBytes: 1 << 20, Timeout: 5 * time.Second}
	}
	policy = nil
	ts := httptest.NewTLSServer(handler)
	t.Cleanup(ts.Close)

	server := NewExchangeServer(store).WithFetchPolicy(pol)
	server.fetchClientFn = func(*FetchPolicy) *http.Client {
		return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	}
	return server, ts
}

func fetchEventCount(t *testing.T, s *ExchangeServer, rawURL, outcome string) int {
	t.Helper()
	events, err := s.store.ListFetchEvents(context.Background(), 100)
	if err != nil {
		t.Fatalf("list fetch events: %v", err)
	}
	count := 0
	for _, ev := range events {
		if ev.URL == rawURL && ev.Outcome == outcome {
			count++
		}
	}
	return count
}

func TestFetchExternalEvidenceDisabledByDefault(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	server := NewExchangeServer(store) // no policy: fail-closed
	const rawURL = "https://example.com/evidence.json"
	_, err = server.FetchExternalEvidence(ctx, &servicesv1.FetchExternalEvidenceRequest{Url: rawURL})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition for disabled fetching, got %v", err)
	}
	if got := fetchEventCount(t, server, rawURL, "policy_denied"); got != 1 {
		t.Fatalf("expected the denial to be audited, got %d events", got)
	}
}

func TestFetchExternalEvidenceRejectsNonHTTPS(t *testing.T) {
	server, _ := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {})
	const rawURL = "http://example.com/evidence.json"
	_, err := server.FetchExternalEvidence(context.Background(), &servicesv1.FetchExternalEvidenceRequest{Url: rawURL})
	if err == nil {
		t.Fatal("expected rejection of a plain-http URL")
	}
	if got := fetchEventCount(t, server, rawURL, "policy_denied"); got != 1 {
		t.Fatalf("expected denial audit event, got %d", got)
	}
}

func TestFetchExternalEvidenceHostAllowlist(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store).WithFetchPolicy(&FetchPolicy{
		Enabled:      true,
		AllowedHosts: []string{"evidence.example.com"},
	})
	const rawURL = "https://untrusted.example.com/evidence.json"
	_, err = server.FetchExternalEvidence(context.Background(), &servicesv1.FetchExternalEvidenceRequest{Url: rawURL})
	if err == nil {
		t.Fatal("expected host allowlist denial")
	}
	if got := fetchEventCount(t, server, rawURL, "policy_denied"); got != 1 {
		t.Fatalf("expected denial audit event, got %d", got)
	}
}

func TestFetchExternalEvidenceStoresContentAddressedEvidence(t *testing.T) {
	payload := []byte(`{"kind":"external-evidence"}`)
	server, ts := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})
	ctx := context.Background()
	resp, err := server.FetchExternalEvidence(ctx, &servicesv1.FetchExternalEvidenceRequest{
		Url:       ts.URL + "/artifact.json",
		MediaType: "application/json",
		BomKind:   "sbom",
	})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !resp.GetStored() {
		t.Fatal("expected evidence to be stored")
	}
	if resp.GetAudit().GetOutcome() != "fetched" {
		t.Fatalf("expected fetched outcome, got %q", resp.GetAudit().GetOutcome())
	}
	if resp.GetEvidence().GetDigest() != digestFor(payload) {
		t.Fatalf("stored digest does not match fetched bytes: %s", resp.GetEvidence().GetDigest())
	}
	if resp.GetEvidence().GetMediaType() != "application/json" || resp.GetEvidence().GetBomKind() != "sbom" {
		t.Fatalf("media type or BOM kind not persisted: %s %s", resp.GetEvidence().GetMediaType(), resp.GetEvidence().GetBomKind())
	}

	// The stored evidence must re-verify against its content digest.
	blob, err := server.store.GetEvidenceBlob(ctx, resp.GetEvidence().GetId())
	if err != nil {
		t.Fatalf("get stored blob: %v", err)
	}
	if digestFor(blob) != resp.GetEvidence().GetDigest() || int64(len(blob)) != resp.GetEvidence().GetSizeBytes() {
		t.Fatal("stored content does not match the declared digest or size")
	}

	// Success is audited too.
	events, err := server.store.ListFetchEvents(ctx, 100)
	if err != nil {
		t.Fatalf("list fetch events: %v", err)
	}
	if len(events) != 1 || events[0].Outcome != "fetched" || events[0].Digest != resp.GetEvidence().GetDigest() {
		t.Fatalf("expected one fetched audit event, got %+v", events)
	}
}

func TestFetchExternalEvidenceIsIdempotentByDigest(t *testing.T) {
	server, ts := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("stable-bytes"))
	}, nil)
	ctx := context.Background()
	req := &servicesv1.FetchExternalEvidenceRequest{Url: ts.URL + "/stable"}
	first, err := server.FetchExternalEvidence(ctx, req)
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	second, err := server.FetchExternalEvidence(ctx, req)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if second.GetEvidence().GetId() != first.GetEvidence().GetId() {
		t.Fatalf("expected the same evidence id, got %s then %s", first.GetEvidence().GetId(), second.GetEvidence().GetId())
	}
	if second.GetAudit().GetDetail() != "already present" {
		t.Fatalf("expected already-present detail, got %q", second.GetAudit().GetDetail())
	}
}

func TestFetchExternalEvidenceRejectsOversizedContent(t *testing.T) {
	server, ts := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	}, &FetchPolicy{Enabled: true, MaxBytes: 1024, Timeout: 5 * time.Second})

	_, err := server.FetchExternalEvidence(context.Background(), &servicesv1.FetchExternalEvidenceRequest{Url: ts.URL + "/big"})
	if err == nil {
		t.Fatal("expected the size cap to reject the fetch")
	}
	if !strings.Contains(err.Error(), "policy maximum") {
		t.Fatalf("expected a policy maximum diagnostic, got %v", err)
	}
	// Rejected content must not be stored.
	events, err := server.store.ListFetchEvents(context.Background(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Outcome != "fetch_failed" {
		t.Fatalf("expected one fetch_failed audit event, got %+v", events)
	}
}

func TestFetchExternalEvidenceBlocksHTTPRedirect(t *testing.T) {
	server, ts := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example.com/payload", http.StatusFound)
	}, nil)
	_, err := server.FetchExternalEvidence(context.Background(), &servicesv1.FetchExternalEvidenceRequest{Url: ts.URL + "/redirect"})
	if err == nil {
		t.Fatal("expected the http redirect to be blocked")
	}
}

func TestFetchExternalEvidenceRecordsFetchFailures(t *testing.T) {
	server, ts := newFetchFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	_, err := server.FetchExternalEvidence(context.Background(), &servicesv1.FetchExternalEvidenceRequest{Url: ts.URL + "/missing"})
	if err == nil {
		t.Fatal("expected a 404 to fail the fetch")
	}
	events, err := server.store.ListFetchEvents(context.Background(), 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Outcome != "fetch_failed" {
		t.Fatalf("expected one fetch_failed audit event, got %+v", events)
	}
	if !strings.Contains(events[0].Detail, "404") {
		t.Fatalf("expected the status code in the audit detail, got %q", events[0].Detail)
	}
}
