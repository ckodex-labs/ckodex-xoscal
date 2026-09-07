package transparency

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VerifyClaim runs verification checks and updates trust state.
func (s *ExchangeServer) VerifyClaim(ctx context.Context, req *servicesv1.VerifyClaimRequest) (*servicesv1.VerifyClaimResponse, error) {
	claim, err := s.store.GetClaim(ctx, req.GetClaimId())
	if err != nil {
		return nil, mapStoreError("get claim", req.GetClaimId(), err)
	}

	checks, err := normalizeVerificationChecks(req.GetChecks())
	if err != nil {
		return nil, err
	}

	proofState := defaultProofState()
	var diagnostics []string
	providerStates := make(map[string]string, len(checks))

	for _, check := range checks {
		switch check {
		case "digest":
			if ok, diag := s.verifyDigest(ctx, claim); ok {
				proofState.Source = "source_bound"
				providerStates[check] = "available"
			} else {
				proofState.Source = "rejected"
				providerStates[check] = "invalid"
				diagnostics = append(diagnostics, diag)
			}
		case "signature":
			result := s.verifyClaimSignatureWithRegistry(ctx, claim)
			proofState.Signature = result.State
			providerStates[check] = result.Provider
			if result.Message != "" {
				diagnostics = append(diagnostics, "signature: "+result.Message)
			}
		case "policy":
			result := s.evaluateClaimPolicyDispatched(ctx, claim)
			proofState.Policy = result.State
			providerStates[check] = result.Provider
			if result.Message != "" {
				diagnostics = append(diagnostics, "policy: "+result.Message)
			}
		case "transparency":
			result := s.verifyTransparencyInclusion(ctx, s.store, claim, s.transparencyCfg)
			proofState.Transparency = result.State
			providerStates[check] = result.Provider
			if result.Message != "" {
				diagnostics = append(diagnostics, "transparency: "+result.Message)
			}
		case "witness":
			proofState.WitnessJson = `{"status":"missing"}`
			providerStates[check] = "unavailable"
			diagnostics = append(diagnostics, "witness: no checkpoint provided")
		}
	}

	proofState.Claim = "claim_bound"
	proofState.Graph = "unresolved"
	proofState.State = "current_state_verified"
	proofState.Discovery = "discovered"

	trustState := computeTrustState(proofState)
	proofJSON, err := json.Marshal(proofState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode proof state: %v", err)
	}

	checksJSON, err := json.Marshal(checks)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode verification checks: %v", err)
	}
	providerJSON, err := json.Marshal(providerStates)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode provider states: %v", err)
	}
	diagnosticsJSON, err := json.Marshal(diagnostics)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode verification diagnostics: %v", err)
	}
	fingerprint := verificationFingerprint(claim, checks)
	event, err := s.store.RecordVerification(ctx, claim.ID, trustState, string(proofJSON), &VerificationEvent{
		EventID:           "verification-" + strings.TrimPrefix(fingerprint, "sha256:"),
		RequestID:         verificationRequestID(ctx, fingerprint),
		IdempotencyKey:    fingerprint,
		ChecksJSON:        string(checksJSON),
		ProviderStateJSON: string(providerJSON),
		DiagnosticsJSON:   string(diagnosticsJSON),
		CreatedAt:         time.Now().UTC(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "record verification: %v", err)
	}

	storedProofState := proofState
	if err := json.Unmarshal([]byte(event.ProofStateJSON), &storedProofState); err != nil {
		return nil, status.Errorf(codes.Internal, "decode recorded proof state: %v", err)
	}
	var storedDiagnostics []string
	if err := json.Unmarshal([]byte(event.DiagnosticsJSON), &storedDiagnostics); err != nil {
		return nil, status.Errorf(codes.Internal, "decode recorded diagnostics: %v", err)
	}
	protoPS := proofStateToProto(storedProofState)
	return &servicesv1.VerifyClaimResponse{
		ClaimId:     claim.ID,
		ProofState:  protoPS,
		TrustState:  event.TrustState,
		Diagnostics: storedDiagnostics,
	}, nil
}

func normalizeVerificationChecks(checks []string) ([]string, error) {
	if len(checks) == 0 {
		return []string{"digest", "signature", "policy"}, nil
	}
	seen := make(map[string]struct{}, len(checks))
	normalized := make([]string, 0, len(checks))
	for _, check := range checks {
		check = strings.TrimSpace(check)
		if _, ok := map[string]struct{}{
			"digest": {}, "signature": {}, "policy": {}, "transparency": {}, "witness": {},
		}[check]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported verification check %q", check)
		}
		if _, ok := seen[check]; ok {
			continue
		}
		seen[check] = struct{}{}
		normalized = append(normalized, check)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func verificationFingerprint(claim *Claim, checks []string) string {
	payload := struct {
		Version    string
		ClaimID    string
		Checks     []string
		SourceRefs string
		ProofRefs  string
		PolicyRefs string
	}{
		Version: "verification-profile-v1", ClaimID: claim.ID, Checks: checks,
		SourceRefs: claim.SourceRefsJSON, ProofRefs: claim.ProofRefsJSON, PolicyRefs: claim.PolicyRefsJSON,
	}
	b, _ := json.Marshal(payload)
	return digestFor(b)
}

func verificationRequestID(ctx context.Context, fingerprint string) string {
	if requestID := interceptors.RequestIDFromContext(ctx); requestID != "" {
		return requestID
	}
	// Direct callers without the request-ID interceptor still get a stable
	// correlation value tied to the idempotent verification profile.
	return "generated-" + strings.TrimPrefix(fingerprint, "sha256:")[:16]
}

// UploadEvidence stores evidence by content-address.
