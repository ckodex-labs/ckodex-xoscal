package transparency

import (
	"context"
	"encoding/json"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExportClaimReceipt emits a deterministic metadata receipt only after the
// latest persisted verification event proves the beta profile. Evidence bytes
// are deliberately excluded; the receipt carries content digests so the
// operator can retrieve and verify them independently.
func (s *ExchangeServer) ExportClaimReceipt(ctx context.Context, req *servicesv1.ExportClaimReceiptRequest) (*servicesv1.ExportClaimReceiptResponse, error) {
	claimID := req.GetClaimId()
	if claimID == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id is required")
	}
	claim, err := s.store.GetClaim(ctx, claimID)
	if err != nil {
		return nil, mapStoreError("get claim", claimID, err)
	}
	if err := s.store.VerifyVerificationChain(ctx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: verification audit chain is invalid: %v", err)
	}
	if claim.TrustState != "verified" {
		return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: claim trust state is %q; run verification until it is verified", claim.TrustState)
	}
	var ps proofState
	if err := json.Unmarshal([]byte(claim.ProofStateJSON), &ps); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: proof state is invalid JSON")
	}
	if ps.Source != "source_bound" || ps.Signature != "signature_verified" ||
		ps.Policy != "policy_admissible" || ps.State != "current_state_verified" {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: the complete beta proof envelope is not present")
	}
	events, err := s.store.ListVerificationEvents(ctx, claimID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list verification events: %v", err)
	}
	if len(events) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: no persisted verification event exists")
	}
	latest := events[len(events)-1]
	if latest.TrustState != claim.TrustState || !sameJSON(latest.ProofStateJSON, claim.ProofStateJSON) {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: claim projection does not match the latest verification event")
	}
	if ok, diagnostic := s.verifyDigest(ctx, claim); !ok {
		return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: %s", diagnostic)
	}

	claimJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(claimToProto(claim))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode receipt claim: %v", err)
	}
	eventProto, err := verificationEventToProto(latest)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "encode receipt event: %v", err)
	}
	eventJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(eventProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode receipt event: %v", err)
	}
	proofJSON := []byte(claim.ProofStateJSON)
	if !json.Valid(proofJSON) {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: proof state is not valid JSON")
	}
	var evidenceJSON []json.RawMessage
	var sourceRefs []EvidenceRef
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &sourceRefs); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: source references are invalid JSON")
	}
	type receiptEvidenceRef struct {
		Ref    string
		Digest string
	}
	refs := make([]receiptEvidenceRef, 0, len(sourceRefs))
	for _, ref := range sourceRefs {
		refs = append(refs, receiptEvidenceRef{Ref: ref.Ref, Digest: ref.Digest})
	}
	var proofRefs []struct {
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.ProofRefsJSON), &proofRefs); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: proof references are invalid JSON")
	}
	for _, ref := range proofRefs {
		refs = append(refs, receiptEvidenceRef{Ref: ref.Ref, Digest: ref.Digest})
	}
	var policyRefs []struct {
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.PolicyRefsJSON), &policyRefs); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: policy references are invalid JSON")
	}
	for _, ref := range policyRefs {
		refs = append(refs, receiptEvidenceRef{Ref: ref.Ref, Digest: ref.Digest})
	}
	seenEvidence := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		key := ref.Digest
		if key == "" {
			key = "ref:" + ref.Ref
		}
		if key == "" {
			return nil, status.Error(codes.FailedPrecondition, "receipt export blocked: an evidence reference is empty")
		}
		if _, seen := seenEvidence[key]; seen {
			continue
		}
		seenEvidence[key] = struct{}{}
		var evidence *Evidence
		if ref.Digest != "" {
			evidence, err = s.store.GetEvidenceByDigest(ctx, ref.Digest)
		} else {
			evidence, err = s.store.GetEvidence(ctx, ref.Ref)
		}
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: evidence %s is unavailable", key)
		}
		blob, blobErr := s.store.GetEvidenceBlob(ctx, evidence.ID)
		if blobErr != nil || digestFor(blob) != evidence.Digest || int64(len(blob)) != evidence.SizeBytes {
			return nil, status.Errorf(codes.FailedPrecondition, "receipt export blocked: evidence %s failed content verification", key)
		}
		encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(evidenceToProto(evidence))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "encode receipt evidence: %v", err)
		}
		evidenceJSON = append(evidenceJSON, json.RawMessage(encoded))
	}
	receipt := struct {
		SchemaVersion     string            `json:"schema_version"`
		Claim             json.RawMessage   `json:"claim"`
		Evidence          []json.RawMessage `json:"evidence"`
		ProofState        json.RawMessage   `json:"proof_state"`
		VerificationEvent json.RawMessage   `json:"verification_event"`
		TrustState        string            `json:"trust_state"`
	}{
		SchemaVersion:     "xoscal-beta-receipt-v1",
		Claim:             json.RawMessage(claimJSON),
		Evidence:          evidenceJSON,
		ProofState:        json.RawMessage(proofJSON),
		VerificationEvent: json.RawMessage(eventJSON),
		TrustState:        claim.TrustState,
	}
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode receipt: %v", err)
	}
	return &servicesv1.ExportClaimReceiptResponse{
		ClaimId:               claim.ID,
		ReceiptJson:           string(receiptJSON),
		ReceiptDigest:         digestFor(receiptJSON),
		TrustState:            claim.TrustState,
		VerificationEventHash: latest.EventHash,
		ExportedAt:            timestamppb.New(time.Now().UTC()),
	}, nil
}

// PreflightImport validates a batch without mutating the workspace. The beta
// requires source evidence to be present and content-address verified before a
// record can be committed.
