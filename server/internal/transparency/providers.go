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
	var refs []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.ProofRefsJSON), &refs); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "signature proof references are not valid JSON"}
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
		return providerResult{State: "unverified", Provider: "available", Message: "signature proof reference is required"}
	}
	signature, err := resolveEvidenceBlob(ctx, store, signatureRef.Ref, signatureRef.Digest)
	if err != nil {
		return providerResult{State: "unverified", Provider: "available", Message: fmt.Sprintf("signature evidence is unavailable: %v", err)}
	}
	if len(signature) != ed25519.SignatureSize {
		return providerResult{State: "invalid", Provider: "available", Message: "signature evidence is not a 64-byte Ed25519 signature"}
	}

	var issuer map[string]string
	if err := json.Unmarshal([]byte(claim.IssuerJSON), &issuer); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "claim issuer is not valid JSON"}
	}
	var key struct {
		Algorithm string `json:"algorithm"`
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal([]byte(issuer["key"]), &key); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "issuer key metadata is not valid JSON"}
	}
	if key.Algorithm != "ed25519" {
		return providerResult{State: "invalid", Provider: "available", Message: "issuer key algorithm must be ed25519"}
	}
	publicKey, err := decodeBase64Key(key.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return providerResult{State: "invalid", Provider: "available", Message: "issuer public key is not a valid Ed25519 key"}
	}
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("canonical claim payload: %v", err)}
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		return providerResult{State: "invalid", Provider: "available", Message: "signature does not match the canonical claim payload"}
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
