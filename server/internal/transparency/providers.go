package transparency

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type providerResult struct {
	State    string
	Provider string
	Message  string
}

// verifyClaimSignature verifies the beta detached-signature contract:
// issuer.key_json contains an Ed25519 public key and the signature proof ref
// resolves to an evidence blob containing the raw 64-byte signature.
func verifyClaimSignature(ctx context.Context, store Store, claim *Claim) providerResult {
	signature, result := resolveSignatureEvidence(ctx, store, claim)
	if result != nil {
		return *result
	}
	publicKey, retired, err := resolveIssuerKey(nil, claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: err.Error()}
	}
	return verifySignatureAgainstKey(claim, signature, publicKey, retired)
}

// verifyClaimSignatureWithRegistry verifies the claim signature with the
// deployment key registry: the inline issuer key wins when present, otherwise
// the registry resolves the issuer's key_id. A retired key still verifies but
// the result message surfaces the rotation; an unknown key_id is explicitly
// unavailable and can never produce a verified state.
func (s *ExchangeServer) verifyClaimSignatureWithRegistry(ctx context.Context, claim *Claim) providerResult {
	signature, result := resolveSignatureEvidence(ctx, s.store, claim)
	if result != nil {
		return *result
	}
	publicKey, retired, err := resolveIssuerKey(s.keyRegistry, claim)
	if err != nil {
		msg := err.Error()
		state := "invalid"
		provider := "available"
		if strings.Contains(msg, "is not registered") {
			state = "unverified"
			provider = "unavailable"
		}
		return providerResult{State: state, Provider: provider, Message: msg}
	}
	return verifySignatureAgainstKey(claim, signature, publicKey, retired)
}

// resolveSignatureEvidence locates and validates the signature proof
// reference. A nil providerResult means the signature bytes are ready for
// cryptographic verification.
func resolveSignatureEvidence(ctx context.Context, store Store, claim *Claim) ([]byte, *providerResult) {
	var refs []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.ProofRefsJSON), &refs); err != nil {
		return nil, &providerResult{State: "invalid", Provider: "available", Message: "signature proof references are not valid JSON"}
	}
	var signatureRef *struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	for i := range refs {
		if refs[i].Type == "signature" {
			ref := refs[i]
			signatureRef = &ref
			break
		}
	}
	if signatureRef == nil {
		return nil, &providerResult{State: "unverified", Provider: "available", Message: "signature proof reference is required"}
	}
	signature, err := resolveEvidenceBlob(ctx, store, signatureRef.Ref, signatureRef.Digest)
	if err != nil {
		return nil, &providerResult{State: "unverified", Provider: "available", Message: fmt.Sprintf("signature evidence is unavailable: %v", err)}
	}
	if len(signature) != ed25519.SignatureSize {
		return nil, &providerResult{State: "invalid", Provider: "available", Message: "signature evidence is not a 64-byte Ed25519 signature"}
	}
	return signature, nil
}

// resolveIssuerKey returns the Ed25519 public key for a claim issuer. The
// inline `key` object wins when present; otherwise the registry resolves the
// issuer's key_id. The second return value flags a retired key so the
// diagnostic can surface the rotation.
func resolveIssuerKey(registry *KeyRegistry, claim *Claim) (ed25519.PublicKey, bool, error) {
	var issuer map[string]string
	if err := json.Unmarshal([]byte(claim.IssuerJSON), &issuer); err != nil {
		return nil, false, fmt.Errorf("claim issuer is not valid JSON")
	}
	if inline, ok := issuer["key"]; ok && strings.TrimSpace(inline) != "" {
		var key struct {
			Algorithm string `json:"algorithm"`
			PublicKey string `json:"public_key"`
		}
		if err := json.Unmarshal([]byte(inline), &key); err != nil {
			return nil, false, fmt.Errorf("issuer key metadata is not valid JSON")
		}
		if key.Algorithm != "ed25519" {
			return nil, false, fmt.Errorf("issuer key algorithm must be ed25519")
		}
		publicKey, err := decodeBase64Key(key.PublicKey)
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			return nil, false, fmt.Errorf("issuer public key is not a valid Ed25519 key")
		}
		return ed25519.PublicKey(publicKey), false, nil
	}

	keyID := strings.TrimSpace(issuer["key_id"])
	if keyID == "" {
		return nil, false, fmt.Errorf("issuer has neither an inline key nor a key_id")
	}
	entry, ok := registry.lookup(keyID)
	if !ok {
		// Unknown key IDs are unavailable, not invalid: the registry may not
		// know keys that a later rotation introduces.
		return nil, false, fmt.Errorf("issuer key_id %q is not registered", keyID)
	}
	if entry.Algorithm != "ed25519" {
		return nil, false, fmt.Errorf("registered key %q algorithm must be ed25519", keyID)
	}
	publicKey, err := decodeBase64Key(entry.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, false, fmt.Errorf("registered key %q is not a valid Ed25519 key", keyID)
	}
	return ed25519.PublicKey(publicKey), entry.Status == "retired", nil
}

// verifySignatureAgainstKey verifies the detached signature against a
// resolved issuer public key. A retired key still verifies; the caller
// surfaces the rotation in the result message.
func verifySignatureAgainstKey(claim *Claim, signature []byte, publicKey ed25519.PublicKey, retired bool) providerResult {
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("canonical claim payload: %v", err)}
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return providerResult{State: "invalid", Provider: "available", Message: "signature does not match the canonical claim payload"}
	}
	if retired {
		return providerResult{State: "signature_verified", Provider: "available", Message: "issuer key is retired; rotation is visible in diagnostics"}
	}
	return providerResult{State: "signature_verified", Provider: "available"}
}

// evaluateClaimPolicy evaluates the beta xoscal-json policy contract. Policy
// documents are stored as content-addressed evidence and may constrain the
// claim relation, BOM kind, issuer kind, and validity window.
func evaluateClaimPolicy(ctx context.Context, store Store, claim *Claim) providerResult {
	var refs []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.PolicyRefsJSON), &refs); err != nil {
		return providerResult{State: "policy_inadmissible", Provider: "available", Message: "policy references are not valid JSON"}
	}
	if len(refs) == 0 {
		return providerResult{State: "unevaluated", Provider: "available", Message: "policy reference is required"}
	}
	for _, ref := range refs {
		if ref.Type != "xoscal-json" {
			return providerResult{State: "unevaluated", Provider: "unavailable", Message: fmt.Sprintf("policy type %q is not supported by the beta provider", ref.Type)}
		}
		blob, err := resolveEvidenceBlob(ctx, store, ref.Ref, ref.Digest)
		if err != nil {
			return providerResult{State: "unevaluated", Provider: "available", Message: fmt.Sprintf("policy evidence is unavailable: %v", err)}
		}
		var policy betaPolicy
		if err := json.Unmarshal(blob, &policy); err != nil {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: "policy document is not valid JSON"}
		}
		if policy.Version != "xoscal-policy-v1" {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: "policy version must be xoscal-policy-v1"}
		}
		if err := policy.evaluate(claim); err != nil {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: err.Error()}
		}
	}
	return providerResult{State: "policy_admissible", Provider: "available"}
}

type betaPolicy struct {
	Version          string   `json:"version"`
	AllowedRelations []string `json:"allowed_relations"`
	AllowedBOMKinds  []string `json:"allowed_bom_kinds"`
	AllowedIssuers   []string `json:"allowed_issuer_kinds"`
	MaxValidityHours int      `json:"max_validity_hours"`
}

func (p betaPolicy) evaluate(claim *Claim) error {
	var predicate map[string]string
	if err := json.Unmarshal([]byte(claim.PredicateJSON), &predicate); err != nil {
		return fmt.Errorf("claim predicate is not valid JSON")
	}
	if !containsOrUnrestricted(p.AllowedRelations, predicate["relation"]) {
		return fmt.Errorf("policy disallows claim relation %q", predicate["relation"])
	}
	if !containsOrUnrestricted(p.AllowedBOMKinds, claim.BomKind) {
		return fmt.Errorf("policy disallows BOM kind %q", claim.BomKind)
	}
	var issuer map[string]string
	if err := json.Unmarshal([]byte(claim.IssuerJSON), &issuer); err != nil {
		return fmt.Errorf("claim issuer is not valid JSON")
	}
	if !containsOrUnrestricted(p.AllowedIssuers, issuer["kind"]) {
		return fmt.Errorf("policy disallows issuer kind %q", issuer["kind"])
	}
	if p.MaxValidityHours > 0 && claim.ValidTo != nil {
		if claim.ValidTo.Sub(claim.ValidFrom) > time.Duration(p.MaxValidityHours)*time.Hour {
			return fmt.Errorf("claim validity exceeds policy maximum of %d hours", p.MaxValidityHours)
		}
	}
	return nil
}

func containsOrUnrestricted(values []string, candidate string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func resolveEvidenceBlob(ctx context.Context, store Store, ref, digest string) ([]byte, error) {
	var evidence *Evidence
	var err error
	if digest != "" {
		evidence, err = store.GetEvidenceByDigest(ctx, digest)
	} else if ref != "" {
		evidence, err = store.GetEvidence(ctx, ref)
	} else {
		return nil, fmt.Errorf("evidence reference and digest are both empty")
	}
	if err != nil {
		return nil, err
	}
	if digest != "" && evidence.Digest != digest {
		return nil, fmt.Errorf("evidence digest does not match reference")
	}
	blob, err := store.GetEvidenceBlob(ctx, evidence.ID)
	if err != nil {
		return nil, err
	}
	if digestFor(blob) != evidence.Digest || int64(len(blob)) != evidence.SizeBytes {
		return nil, fmt.Errorf("stored evidence failed content-address verification")
	}
	return blob, nil
}

func decodeBase64Key(value string) ([]byte, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "base64:"))
	return base64.StdEncoding.DecodeString(value)
}

func canonicalClaimPayload(claim *Claim) ([]byte, error) {
	payload := struct {
		Version       string `json:"version"`
		ID            string `json:"id"`
		Type          string `json:"type"`
		SubjectJSON   string `json:"subject_json"`
		PredicateJSON string `json:"predicate_json"`
		ObjectJSON    string `json:"object_json"`
		IssuerJSON    string `json:"issuer_json"`
		BOMKind       string `json:"bom_kind"`
		ValidFrom     string `json:"valid_from"`
		ValidTo       string `json:"valid_to,omitempty"`
		ObservedTime  string `json:"observed_time"`
		SourceRefs    string `json:"source_refs_json"`
		PolicyRefs    string `json:"policy_refs_json"`
		Extensions    string `json:"extensions_json"`
	}{
		Version: "xoscal-claim-signature-v1", ID: claim.ID, Type: claim.Type,
		SubjectJSON: claim.SubjectJSON, PredicateJSON: claim.PredicateJSON,
		ObjectJSON: claim.ObjectJSON, IssuerJSON: claim.IssuerJSON, BOMKind: claim.BomKind,
		ValidFrom: claim.ValidFrom.UTC().Format(time.RFC3339Nano), ObservedTime: claim.ObservedTime.UTC().Format(time.RFC3339Nano),
		SourceRefs: claim.SourceRefsJSON, PolicyRefs: claim.PolicyRefsJSON, Extensions: claim.ExtensionsJSON,
	}
	if claim.ValidTo != nil {
		payload.ValidTo = claim.ValidTo.UTC().Format(time.RFC3339Nano)
	}
	return json.Marshal(payload)
}
