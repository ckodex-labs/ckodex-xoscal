package transparency

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PreflightImport validates a batch without mutating the workspace. The beta
// requires source evidence to be present and content-address verified before a
// record can be committed.
func (s *ExchangeServer) PreflightImport(ctx context.Context, req *servicesv1.PreflightImportRequest) (*servicesv1.PreflightImportResponse, error) {
	records, results, valid := s.prepareImportRecords(ctx, req.GetRecords())
	_ = records // preparation is intentionally side-effect free
	return &servicesv1.PreflightImportResponse{Valid: valid, Results: results}, nil
}

// ImportBatch commits a preflighted batch in one transaction. Partial commits
// are deliberately not available in the beta because they make operator
// recovery and claim/evidence correspondence ambiguous.

// ImportBatch commits a preflighted batch in one transaction. Partial commits
// are deliberately not available in the beta because they make operator
// recovery and claim/evidence correspondence ambiguous.
func (s *ExchangeServer) ImportBatch(ctx context.Context, req *servicesv1.ImportBatchRequest) (*servicesv1.ImportBatchResponse, error) {
	if !req.GetAllOrNothing() {
		return nil, status.Error(codes.InvalidArgument, "all_or_nothing must be true for beta batch import")
	}
	records, results, valid := s.prepareImportRecords(ctx, req.GetRecords())
	if !valid {
		return &servicesv1.ImportBatchResponse{Committed: false, Results: results}, nil
	}
	if err := s.store.CreateImportBatch(ctx, records); err != nil {
		return nil, status.Errorf(codes.Aborted, "batch import was rolled back: %v", err)
	}
	for _, result := range results {
		if result.GetStatus() == "ready" {
			result.Status = "imported"
		}
	}
	return &servicesv1.ImportBatchResponse{Committed: true, Results: results}, nil
}

func (s *ExchangeServer) prepareImportRecords(ctx context.Context, inputs []*servicesv1.ImportRecord) ([]*ImportRecord, []*servicesv1.ImportRecordResult, bool) {
	if len(inputs) == 0 {
		return nil, []*servicesv1.ImportRecordResult{{Index: -1, Status: "invalid", Diagnostics: []string{"at least one import record is required"}}}, false
	}
	if len(inputs) > maxImportRecords {
		return nil, []*servicesv1.ImportRecordResult{{Index: -1, Status: "invalid", Diagnostics: []string{fmt.Sprintf("at most %d import records are allowed per beta batch", maxImportRecords)}}}, false
	}
	records := make([]*ImportRecord, 0, len(inputs))
	results := make([]*servicesv1.ImportRecordResult, 0, len(inputs))
	claimIDs := make(map[string]int, len(inputs))
	valid := true
	var responseIndex int32
	for index, input := range inputs {
		result := &servicesv1.ImportRecordResult{Index: responseIndex, Status: "ready"}
		responseIndex++
		if input == nil || input.GetClaim() == nil {
			result.Status = "invalid"
			result.Diagnostics = append(result.Diagnostics, "claim is required")
			results = append(results, result)
			valid = false
			continue
		}
		claimProto := input.GetClaim()
		result.ClaimId = claimProto.GetId()
		if previous, exists := claimIDs[result.ClaimId]; exists {
			result.Status = "invalid"
			result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("claim id duplicates import record %d", previous))
			valid = false
		}
		claimIDs[result.ClaimId] = index
		claim, diagnostics := prepareImportClaim(claimProto)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		record := &ImportRecord{Claim: claim}
		batchEvidenceByID := make(map[string]*Evidence)
		batchEvidenceByDigest := make(map[string]*Evidence)
		for evidenceIndex, inputEvidence := range input.GetEvidence() {
			evidence, evidenceDiagnostics := prepareImportEvidence(inputEvidence)
			result.Diagnostics = append(result.Diagnostics, prefixImportDiagnostics(evidenceIndex, evidenceDiagnostics)...)
			if evidence == nil {
				continue
			}
			if _, exists := batchEvidenceByID[evidence.ID]; exists {
				result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("evidence id %q is duplicated in record", evidence.ID))
			}
			if _, exists := batchEvidenceByDigest[evidence.Digest]; exists {
				result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("evidence digest %q is duplicated in record", evidence.Digest))
			}
			batchEvidenceByID[evidence.ID] = evidence
			batchEvidenceByDigest[evidence.Digest] = evidence
			record.Evidence = append(record.Evidence, evidence)
			if existing, err := s.store.GetEvidence(ctx, evidence.ID); err == nil {
				if !equivalentEvidenceContent(existing, evidence) {
					result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("evidence id %q conflicts with stored content", evidence.ID))
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("check existing evidence %q: %v", evidence.ID, err))
			}
			if existing, err := s.store.GetEvidenceByDigest(ctx, evidence.Digest); err == nil {
				if !equivalentEvidenceContent(existing, evidence) {
					result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("evidence digest %q conflicts with stored content", evidence.Digest))
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("check existing evidence digest %q: %v", evidence.Digest, err))
			}
		}
		if claim != nil {
			result.Diagnostics = append(result.Diagnostics, s.validateImportReferences(ctx, claim, batchEvidenceByID, batchEvidenceByDigest)...)
			if existing, err := s.store.GetClaim(ctx, claim.ID); err == nil {
				if claimsEquivalent(existing, claim) {
					result.Status = "already_present"
				} else {
					result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("claim id %q conflicts with stored content", claim.ID))
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				result.Diagnostics = append(result.Diagnostics, fmt.Sprintf("check existing claim %q: %v", claim.ID, err))
			}
		}
		if len(result.Diagnostics) > 0 {
			result.Status = "invalid"
			valid = false
		}
		records = append(records, record)
		results = append(results, result)
	}
	return records, results, valid
}

func prepareImportClaim(input *servicesv1.Claim) (*Claim, []string) {
	var diagnostics []string
	if input.GetId() == "" {
		diagnostics = append(diagnostics, "claim id is required")
	}
	if input.GetPredicate() == nil || input.GetPredicate().GetRelation() == "" {
		diagnostics = append(diagnostics, "claim predicate relation is required")
	}
	if input.GetSubject() == nil || (input.GetSubject().GetId() == "" && input.GetSubject().GetDigest() == "") {
		diagnostics = append(diagnostics, "claim subject id or digest is required")
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	claim := claimFromProto(input)
	claim.TrustState = "candidate"
	claim.ProofStateJSON = defaultProofStateJSON()
	claim.CreatedAt = time.Now().UTC()
	if claim.ObservedTime.IsZero() {
		claim.ObservedTime = claim.CreatedAt
	}
	return claim, nil
}

func prepareImportEvidence(input *servicesv1.ImportEvidence) (*Evidence, []string) {
	if input == nil || input.GetEvidence() == nil {
		return nil, []string{"evidence metadata and blob are required"}
	}
	ev := input.GetEvidence()
	var diagnostics []string
	if ev.GetId() == "" {
		diagnostics = append(diagnostics, "evidence id is required")
	}
	if ev.GetDigest() == "" || !strings.HasPrefix(ev.GetDigest(), "sha256:") {
		diagnostics = append(diagnostics, "evidence digest must use sha256:<hex> format")
	}
	if len(input.GetBlob()) == 0 {
		diagnostics = append(diagnostics, "evidence blob is required")
	}
	if int64(len(input.GetBlob())) != ev.GetSizeBytes() {
		diagnostics = append(diagnostics, fmt.Sprintf("evidence size_bytes %d does not match blob size %d", ev.GetSizeBytes(), len(input.GetBlob())))
	}
	if ev.GetDigest() != "" && strings.HasPrefix(ev.GetDigest(), "sha256:") && len(input.GetBlob()) > 0 && digestFor(input.GetBlob()) != ev.GetDigest() {
		diagnostics = append(diagnostics, fmt.Sprintf("evidence digest does not match blob: got %s", digestFor(input.GetBlob())))
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}
	evidence := evidenceFromProto(ev)
	evidence.Blob = append([]byte(nil), input.GetBlob()...)
	evidence.CreatedAt = time.Now().UTC()
	evidence.ValidFrom = evidence.CreatedAt
	return evidence, nil
}

func prefixImportDiagnostics(index int, diagnostics []string) []string {
	for i := range diagnostics {
		diagnostics[i] = fmt.Sprintf("evidence[%d]: %s", index, diagnostics[i])
	}
	return diagnostics
}

func equivalentEvidenceContent(left, right *Evidence) bool {
	return evidenceEquivalent(left, right) && digestFor(left.Blob) == digestFor(right.Blob)
}

func (s *ExchangeServer) validateImportReferences(ctx context.Context, claim *Claim, byID map[string]*Evidence, byDigest map[string]*Evidence) []string {
	var diagnostics []string
	var sourceRefs []EvidenceRef
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &sourceRefs); err != nil || len(sourceRefs) == 0 {
		return []string{"at least one source evidence reference is required"}
	}
	for _, ref := range sourceRefs {
		if ref.Digest == "" {
			diagnostics = append(diagnostics, "source evidence references must include a digest")
			continue
		}
		if _, ok := byDigest[ref.Digest]; ok {
			continue
		}
		if _, err := s.store.GetEvidenceByDigest(ctx, ref.Digest); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("source evidence digest %s is not available", ref.Digest))
		}
	}
	for _, raw := range []struct {
		name string
		json string
	}{
		{name: "proof", json: claim.ProofRefsJSON}, {name: "policy", json: claim.PolicyRefsJSON},
	} {
		var refs []struct {
			Ref    string `json:"ref"`
			Digest string `json:"digest"`
		}
		if raw.json == "" || raw.json == "null" {
			continue
		}
		if err := json.Unmarshal([]byte(raw.json), &refs); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("%s references are invalid JSON", raw.name))
			continue
		}
		for _, ref := range refs {
			if ref.Digest == "" && ref.Ref == "" {
				diagnostics = append(diagnostics, fmt.Sprintf("%s evidence reference is empty", raw.name))
				continue
			}
			if ref.Digest != "" {
				if _, ok := byDigest[ref.Digest]; ok {
					continue
				}
				if _, err := s.store.GetEvidenceByDigest(ctx, ref.Digest); err == nil {
					continue
				} else if !errors.Is(err, sql.ErrNoRows) {
					diagnostics = append(diagnostics, fmt.Sprintf("check %s evidence digest %s: %v", raw.name, ref.Digest, err))
				}
			} else if _, ok := byID[ref.Ref]; ok {
				continue
			} else if _, err := s.store.GetEvidence(ctx, ref.Ref); err == nil {
				continue
			} else if !errors.Is(err, sql.ErrNoRows) {
				diagnostics = append(diagnostics, fmt.Sprintf("check %s evidence %s: %v", raw.name, ref.Ref, err))
			}
		}
	}
	return diagnostics
}

// VerifyClaim runs verification checks and updates trust state.
