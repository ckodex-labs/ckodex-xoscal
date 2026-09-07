package transparency

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ---- Key registry with rotation ----

// KeyEntry is one registered issuer public key. Retired keys still verify
// existing claims but are flagged: rotation must not silently invalidate
// history, and it must be visible in diagnostics.
type KeyEntry struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Status    string `json:"status"` // active | retired
}

// KeyRegistry resolves issuer key IDs to public keys. It is populated from
// deployment configuration; rotation is a config change that keeps prior
// keys registered as retired.
type KeyRegistry struct {
	Keys map[string]KeyEntry `json:"keys"`
}

func (r *KeyRegistry) lookup(keyID string) (KeyEntry, bool) {
	if r == nil || r.Keys == nil {
		return KeyEntry{}, false
	}
	entry, ok := r.Keys[keyID]
	return entry, ok
}

// resolveIssuerKey returns the Ed25519 public key for a claim issuer. The
// inline `key` object wins when present; otherwise the registry resolves the
// issuer's key_id. The second return value flags a retired key so the
// diagnostic can surface the rotation.
func (r *KeyRegistry) resolveIssuerKey(claim *Claim) (ed25519.PublicKey, bool, error) {
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
	entry, ok := r.lookup(keyID)
	if !ok {
		// Unknown key IDs are unavailable, not invalid: the registry may not
		// know keys a later rotation introduces.
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

// ---- Remote policy provider ----

// evaluateRemoteClaimPolicy fetches a policy document from an allowlisted
// HTTPS endpoint through the bounded fetch policy and evaluates it as an
// xoscal-policy-v1 document. The fetch reuses the external fetch bounds and
// its audit trail; a fetch failure leaves the check explicitly unevaluated.
func (s *ExchangeServer) evaluateRemoteClaimPolicy(ctx context.Context, claim *Claim) providerResult {
	var refs []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal([]byte(claim.PolicyRefsJSON), &refs); err != nil {
		return providerResult{State: "policy_inadmissible", Provider: "available", Message: "policy references are not valid JSON"}
	}
	policy := s.fetchPolicy.normalized()
	if !policy.Enabled {
		return providerResult{State: "unevaluated", Provider: "unavailable", Message: "remote policy evaluation requires an enabled fetch policy"}
	}
	for _, ref := range refs {
		if ref.Type != "xoscal-remote" {
			continue
		}
		if err := policy.validateURL(ref.URL); err != nil {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: fmt.Sprintf("remote policy URL rejected: %v", err)}
		}
		blob, err := s.fetchBounded(ctx, ref.URL, policy, s.fetchClient(policy))
		if err != nil {
			return providerResult{State: "unevaluated", Provider: "unavailable", Message: fmt.Sprintf("remote policy is unavailable: %v", err)}
		}
		var doc betaPolicy
		if err := json.Unmarshal(blob, &doc); err != nil {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: "remote policy document is not valid JSON"}
		}
		if doc.Version != "xoscal-policy-v1" {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: "remote policy version must be xoscal-policy-v1"}
		}
		if err := doc.evaluate(claim); err != nil {
			return providerResult{State: "policy_inadmissible", Provider: "available", Message: err.Error()}
		}
	}
	return providerResult{State: "policy_admissible", Provider: "available"}
}

// evaluateClaimPolicyDispatched routes each policy reference to its provider:
// xoscal-json documents resolve from stored evidence; xoscal-remote documents
// are fetched through the bounded fetch policy and evaluated the same way.
func (s *ExchangeServer) evaluateClaimPolicyDispatched(ctx context.Context, claim *Claim) providerResult {
	var refs []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(claim.PolicyRefsJSON), &refs); err != nil {
		return providerResult{State: "policy_inadmissible", Provider: "available", Message: "policy references are not valid JSON"}
	}
	hasRemote := false
	for _, ref := range refs {
		if ref.Type == "xoscal-remote" {
			hasRemote = true
			break
		}
	}
	if hasRemote {
		return s.evaluateRemoteClaimPolicy(ctx, claim)
	}
	return evaluateClaimPolicy(ctx, s.store, claim)
}

// ---- Transparency inclusion provider ----

// TransparencyConfig enables the transparency inclusion provider. Without it
// the transparency check stays explicitly unavailable.
type TransparencyConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	LogID     string `mapstructure:"log_id"`
	PublicKey string `mapstructure:"public_key"` // base64 Ed25519 public key of the log
}

// transparencyInclusion is the evidence payload a transparency proof
// reference resolves to: an RFC 6962 style inclusion proof for the claim's
// canonical payload plus a checkpoint signed by the log's key.
type transparencyInclusion struct {
	Version       string   `json:"version"`
	LogID         string   `json:"log_id"`
	LogIndex      uint64   `json:"log_index"`
	RootHash      string   `json:"root_hash"`
	InclusionPath []string `json:"inclusion_path"` // hex-encoded sibling hashes, leaf to root
	Checkpoint    struct {
		RootHash  string `json:"root_hash"`
		LogSize   uint64 `json:"log_size"`
		Signature string `json:"signature"` // base64 Ed25519 over the canonical checkpoint bytes
	} `json:"checkpoint"`
}

// canonicalCheckpointBytes is the exact byte string the log signs. It binds
// the log identity, the root hash, and the tree size.
func canonicalCheckpointBytes(logID, rootHash string, logSize uint64) []byte {
	return []byte(fmt.Sprintf("xoscal-transparency-checkpoint-v1\nlog_id:%s\nroot:%s\nsize:%d\n", logID, rootHash, logSize))
}

// merkleLeafHash is RFC 6962 leaf hashing: SHA256(0x00 || leaf).
func merkleLeafHash(leaf []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(leaf)
	return h.Sum(nil)
}

// verifyInclusionProof replays the RFC 6962 inclusion path from the leaf to
// the root: SHA256(0x01 || left || right) per level, sibling order chosen by
// the index bit.
func verifyInclusionProof(leafHash []byte, index uint64, path []string, rootHash string) error {
	computed := leafHash
	idx := index
	for i, step := range path {
		sibling, err := hex.DecodeString(step)
		if err != nil || len(sibling) != sha256.Size {
			return fmt.Errorf("inclusion path element %d is not a hex sha256 hash", i)
		}
		h := sha256.New()
		h.Write([]byte{0x01})
		if idx%2 == 0 {
			h.Write(computed)
			h.Write(sibling)
		} else {
			h.Write(sibling)
			h.Write(computed)
		}
		computed = h.Sum(nil)
		idx /= 2
	}
	if fmt.Sprintf("%x", computed) != strings.ToLower(rootHash) {
		return fmt.Errorf("inclusion path does not verify against the checkpoint root")
	}
	return nil
}

// verifyTransparencyInclusion checks a claim's transparency proof reference
// against the configured log. The check fails closed: without configuration
// it is unavailable; with configuration, any malformed or non-verifying
// proof is invalid and can never produce a verified trust state.
func (s *ExchangeServer) verifyTransparencyInclusion(ctx context.Context, store Store, claim *Claim, cfg *TransparencyConfig) providerResult {
	if cfg == nil || !cfg.Enabled {
		return providerResult{State: "missing", Provider: "unavailable", Message: "transparency: no transparency log is configured"}
	}

	var refs []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.ProofRefsJSON), &refs); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "proof references are not valid JSON"}
	}
	var inclusionRef *struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	for i := range refs {
		if refs[i].Type == "transparency" {
			ref := refs[i]
			inclusionRef = &ref
			break
		}
	}
	if inclusionRef == nil {
		return providerResult{State: "missing", Provider: "available", Message: "transparency: inclusion proof not provided"}
	}
	blob, err := resolveEvidenceBlob(ctx, store, inclusionRef.Ref, inclusionRef.Digest)
	if err != nil {
		return providerResult{State: "missing", Provider: "available", Message: fmt.Sprintf("transparency evidence is unavailable: %v", err)}
	}

	var inclusion transparencyInclusion
	if err := json.Unmarshal(blob, &inclusion); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "transparency inclusion proof is not valid JSON"}
	}
	if inclusion.Version != "xoscal-transparency-v1" {
		return providerResult{State: "invalid", Provider: "available", Message: "transparency version must be xoscal-transparency-v1"}
	}
	if inclusion.LogID != cfg.LogID {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("inclusion proof is for log %q, not the configured log %q", inclusion.LogID, cfg.LogID)}
	}

	logKey, err := decodeBase64Key(cfg.PublicKey)
	if err != nil || len(logKey) != ed25519.PublicKeySize {
		return providerResult{State: "invalid", Provider: "available", Message: "configured transparency log key is not a valid Ed25519 key"}
	}
	checkpointSig, err := base64.StdEncoding.DecodeString(inclusion.Checkpoint.Signature)
	if err != nil || len(checkpointSig) != ed25519.SignatureSize {
		return providerResult{State: "invalid", Provider: "available", Message: "checkpoint signature is not a 64-byte Ed25519 signature"}
	}
	checkpointBytes := canonicalCheckpointBytes(cfg.LogID, inclusion.Checkpoint.RootHash, inclusion.Checkpoint.LogSize)
	if !ed25519.Verify(logKey, checkpointBytes, checkpointSig) {
		return providerResult{State: "invalid", Provider: "available", Message: "checkpoint signature does not verify against the configured log key"}
	}
	if inclusion.Checkpoint.RootHash != inclusion.RootHash {
		return providerResult{State: "invalid", Provider: "available", Message: "inclusion proof root does not match the checkpoint root"}
	}

	leaf, err := canonicalClaimPayload(claim)
	if err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: fmt.Sprintf("canonical claim payload: %v", err)}
	}
	if err := verifyInclusionProof(merkleLeafHash(leaf), inclusion.LogIndex, inclusion.InclusionPath, inclusion.RootHash); err != nil {
		return providerResult{State: "invalid", Provider: "available", Message: "transparency: " + err.Error()}
	}
	return providerResult{State: "inclusion_verified", Provider: "available"}
}
