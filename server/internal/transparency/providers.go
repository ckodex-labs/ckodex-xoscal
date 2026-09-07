package transparency

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type providerResult struct {
	State    string
	Provider string
	Message  string
}

// signatureVerifier abstracts the supported detached-signature algorithms.
// Every algorithm verifies the same canonical claim payload; the algorithms
// differ in key encoding and signature byte layout, both documented in
// docs/BETA-CONTRACT.md.
type signatureVerifier struct {
	algorithm string
	size      int
	verify    func(payload, signature []byte) bool
}

// ed25519Verifier adapts an Ed25519 public key: 64-byte raw signatures over
// the canonical payload directly.
func ed25519Verifier(publicKey ed25519.PublicKey) signatureVerifier {
	return signatureVerifier{
		algorithm: "ed25519",
		size:      ed25519.SignatureSize,
		verify: func(payload, sig []byte) bool {
			return ed25519.Verify(publicKey, payload, sig)
		},
	}
}

// ecdsaVerifier builds a verifier for a raw r||s signature over the SHA-256
// digest of the canonical payload. size is 2*byteLen(curve).
func ecdsaVerifier(curve elliptic.Curve, publicX, publicY *big.Int) (signatureVerifier, error) {
	size := (curve.Params().BitSize + 7) / 8 * 2
	return signatureVerifier{
		algorithm: curve.Params().Name,
		size:      size,
		verify: func(payload, sig []byte) bool {
			if len(sig) != size {
				return false
			}
			h := sha256.Sum256(payload)
			r := new(big.Int).SetBytes(sig[:len(sig)/2])
			s := new(big.Int).SetBytes(sig[len(sig)/2:])
			return ecdsa.Verify(&ecdsa.PublicKey{Curve: curve, X: publicX, Y: publicY}, h[:][:], r, s)
		},
	}, nil
}

// verifyClaimSignature verifies the beta detached-signature contract:
// issuer.key_json contains a public key and the signature proof ref resolves
// to an evidence blob containing the raw signature bytes.
func verifyClaimSignature(ctx context.Context, store Store, claim *Claim) providerResult {
	signature, result := resolveSignatureEvidence(ctx, store, claim)
	if result != nil {
		return *result
	}
	verifier, retired, err := resolveIssuerVerifier(nil, claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: err.Error()}
	}
	return verifySignatureAgainstKey(claim, signature, verifier, retired)
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
	verifier, retired, err := resolveIssuerVerifier(s.keyRegistry, claim)
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
	return verifySignatureAgainstKey(claim, signature, verifier, retired)
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
	return signature, nil
}

// resolveIssuerVerifier resolves the claim issuer's key to a verifier. The
// inline `key` object wins when present; otherwise the registry resolves the
// issuer's key_id. The second return value flags a retired key so the
// diagnostic can surface the rotation.
func resolveIssuerVerifier(registry *KeyRegistry, claim *Claim) (signatureVerifier, bool, error) {
	var issuer map[string]string
	if err := json.Unmarshal([]byte(claim.IssuerJSON), &issuer); err != nil {
		return signatureVerifier{}, false, fmt.Errorf("claim issuer is not valid JSON")
	}
	if inline, ok := issuer["key"]; ok && strings.TrimSpace(inline) != "" {
		return verifierFromKeyJSON(inline)
	}

	keyID := strings.TrimSpace(issuer["key_id"])
	if keyID == "" {
		return signatureVerifier{}, false, fmt.Errorf("issuer has neither an inline key nor a key_id")
	}
	entry, ok := registry.lookup(keyID)
	if !ok {
		// Unknown key IDs are unavailable, not invalid: the registry may not
		// know keys that a later rotation introduces.
		return signatureVerifier{}, false, fmt.Errorf("issuer key_id %q is not registered", keyID)
	}
	verifier, err := verifierFromKeyMaterial(entry.Algorithm, entry.PublicKey)
	if err != nil {
		return signatureVerifier{}, false, fmt.Errorf("registered key %q: %v", keyID, err)
	}
	return verifier, entry.Status == "retired", nil
}

// verifierFromKeyJSON parses an inline issuer key object.
func verifierFromKeyJSON(raw string) (signatureVerifier, bool, error) {
	var key struct {
		Algorithm string `json:"algorithm"`
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal([]byte(raw), &key); err != nil {
		return signatureVerifier{}, false, fmt.Errorf("issuer key metadata is not valid JSON")
	}
	verifier, err := verifierFromKeyMaterial(key.Algorithm, key.PublicKey)
	if err != nil {
		return signatureVerifier{}, false, err
	}
	return verifier, false, nil
}

// verifierFromKeyMaterial decodes a base64 public key for a supported
// algorithm: ed25519 (32-byte key, 64-byte signature) or ECDSA P-256/P-384
// (raw x||y point coordinates, raw r||s signature over the SHA-256 digest).
func verifierFromKeyMaterial(algorithm, encodedKey string) (signatureVerifier, error) {
	publicKey, err := decodeBase64Key(encodedKey)
	if err != nil {
		return signatureVerifier{}, fmt.Errorf("issuer public key is not valid base64")
	}
	switch algorithm {
	case "ed25519":
		if len(publicKey) != ed25519.PublicKeySize {
			return signatureVerifier{}, fmt.Errorf("issuer public key is not a valid Ed25519 key")
		}
		return ed25519Verifier(ed25519.PublicKey(publicKey)), nil
	case "ecdsa-p256", "ecdsa-p384":
		size := pointSize(algorithm)
		if len(publicKey) != 2*size {
			return signatureVerifier{}, fmt.Errorf("issuer public key is not a valid %s point", algorithm)
		}
		curve := elliptic.P256()
		if algorithm == "ecdsa-p384" {
			curve = elliptic.P384()
		}
		verifier, err := ecdsaVerifier(curve, new(big.Int).SetBytes(publicKey[:size]), new(big.Int).SetBytes(publicKey[size:]))
		if err != nil {
			return signatureVerifier{}, err
		}
		return verifier, nil
	default:
		return signatureVerifier{}, fmt.Errorf("signature algorithm %q is not supported", algorithm)
	}
}

func pointSize(algorithm string) int {
	if algorithm == "ecdsa-p384" {
		return 48
	}
	return 32
}

// verifySignatureAgainstKey verifies the detached signature against a
// resolved issuer verifier. A retired key still verifies; the caller
// surfaces the rotation in the result message.
func verifySignatureAgainstKey(claim *Claim, signature []byte, verifier signatureVerifier, retired bool) providerResult {
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("canonical claim payload: %v", err)}
	}
	if len(signature) != verifier.size {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("signature evidence is not a %d-byte %s signature", verifier.size, verifier.algorithm)}
	}
	if !verifier.verify(payload, signature) {
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
