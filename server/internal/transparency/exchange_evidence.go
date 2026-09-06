package transparency

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadEvidence stores evidence by content-address.
func (s *ExchangeServer) UploadEvidence(ctx context.Context, req *servicesv1.UploadEvidenceRequest) (*servicesv1.UploadEvidenceResponse, error) {
	ev := req.GetEvidence()
	if ev == nil || ev.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "evidence id is required")
	}
	if ev.GetDigest() == "" || !strings.HasPrefix(ev.GetDigest(), "sha256:") {
		return nil, status.Error(codes.InvalidArgument, "evidence digest must use sha256:<hex> format")
	}
	if len(req.GetBlob()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "evidence blob is required for beta content verification")
	}
	if int64(len(req.GetBlob())) != ev.GetSizeBytes() {
		return nil, status.Errorf(codes.InvalidArgument, "evidence size_bytes %d does not match blob size %d", ev.GetSizeBytes(), len(req.GetBlob()))
	}
	if got := digestFor(req.GetBlob()); got != ev.GetDigest() {
		return nil, status.Errorf(codes.InvalidArgument, "evidence digest does not match blob: got %s", got)
	}

	evDB := evidenceFromProto(ev)
	evDB.Blob = append([]byte(nil), req.GetBlob()...)
	if evDB.CreatedAt.IsZero() {
		evDB.CreatedAt = time.Now().UTC()
	}
	if evDB.ValidFrom.IsZero() {
		evDB.ValidFrom = evDB.CreatedAt
	}

	if err := s.store.CreateEvidence(ctx, evDB); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			if existing, lookupErr := s.store.GetEvidence(ctx, evDB.ID); lookupErr == nil && evidenceEquivalent(existing, evDB) {
				return &servicesv1.UploadEvidenceResponse{Evidence: evidenceToProto(existing), Stored: true}, nil
			}
			if existing, lookupErr := s.store.GetEvidenceByDigest(ctx, evDB.Digest); lookupErr == nil && evidenceEquivalent(existing, evDB) {
				return &servicesv1.UploadEvidenceResponse{Evidence: evidenceToProto(existing), Stored: true}, nil
			}
			return nil, status.Errorf(codes.AlreadyExists, "evidence %q or digest %q already exists", evDB.ID, evDB.Digest)
		}
		return nil, status.Errorf(codes.Internal, "create evidence: %v", err)
	}

	return &servicesv1.UploadEvidenceResponse{Evidence: ev, Stored: true}, nil
}

// GetEvidence retrieves evidence by ID.

// GetEvidence retrieves evidence by ID.
func (s *ExchangeServer) GetEvidence(ctx context.Context, req *servicesv1.GetEvidenceRequest) (*servicesv1.GetEvidenceResponse, error) {
	ev, err := s.store.GetEvidence(ctx, req.GetEvidenceId())
	if err != nil {
		return nil, mapStoreError("get evidence", req.GetEvidenceId(), err)
	}
	return &servicesv1.GetEvidenceResponse{Evidence: evidenceToProto(ev)}, nil
}

// VerifyEvidence checks digest and optionally fetches and re-hashes.

// VerifyEvidence checks digest and optionally fetches and re-hashes.
func (s *ExchangeServer) VerifyEvidence(ctx context.Context, req *servicesv1.VerifyEvidenceRequest) (*servicesv1.VerifyEvidenceResponse, error) {
	ev, err := s.store.GetEvidence(ctx, req.GetEvidenceId())
	if err != nil {
		return nil, mapStoreError("get evidence", req.GetEvidenceId(), err)
	}

	resp := &servicesv1.VerifyEvidenceResponse{EvidenceId: ev.ID}
	if !req.GetFetchAndHash() {
		resp.Error = "fetch_and_hash must be true to verify stored content"
		return resp, nil
	}
	blob, err := s.store.GetEvidenceBlob(ctx, ev.ID)
	if err != nil {
		return nil, mapStoreError("get evidence content", ev.ID, err)
	}
	resp.DigestOk = digestFor(blob) == ev.Digest
	resp.SizeOk = int64(len(blob)) == ev.SizeBytes
	if !resp.DigestOk || !resp.SizeOk {
		resp.Error = "stored content does not match the declared digest or size"
	}

	return resp, nil
}

// SyncClaims is intentionally outside the beta trust boundary. A peer-sync
// implementation must verify the peer identity, replay window, and imported
// proof bundle before it can mutate this store.

// SyncClaims is intentionally outside the beta trust boundary. A peer-sync
// implementation must verify the peer identity, replay window, and imported
// proof bundle before it can mutate this store.
func (s *ExchangeServer) SyncClaims(ctx context.Context, req *servicesv1.SyncClaimsRequest) (*servicesv1.SyncClaimsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "peer claim sync is not part of the beta trust boundary")
}

// ---- helpers ----

// ---- helpers ----

func (s *ExchangeServer) verifyDigest(ctx context.Context, claim *Claim) (bool, string) {
	var refs []EvidenceRef
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &refs); err != nil {
		return false, "invalid source_refs JSON"
	}
	if len(refs) == 0 {
		return false, "claim has no source evidence references"
	}
	for _, ref := range refs {
		if ref.Digest == "" {
			return false, "source evidence reference has no digest"
		}
		ev, err := s.store.GetEvidenceByDigest(ctx, ref.Digest)
		if err != nil {
			return false, fmt.Sprintf("evidence digest %s not found", ref.Digest)
		}
		blob, err := s.store.GetEvidenceBlob(ctx, ev.ID)
		if err != nil {
			return false, fmt.Sprintf("evidence content %s is unavailable", ref.Digest)
		}
		if digestFor(blob) != ev.Digest || int64(len(blob)) != ev.SizeBytes {
			return false, fmt.Sprintf("evidence content %s failed digest or size verification", ref.Digest)
		}
	}
	return true, ""
}
