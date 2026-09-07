package transparency

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---- Key registry ----

func newRegistryFixture(t *testing.T) (*ExchangeServer, ed25519.PublicKey, ed25519.PrivateKey, ed25519.PrivateKey) {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	retiredPub, retiredPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate retired key: %v", err)
	}
	registry := &KeyRegistry{Keys: map[string]KeyEntry{
		"issuer-2026": {KeyID: "issuer-2026", Algorithm: "ed25519", PublicKey: "base64:" + base64.StdEncoding.EncodeToString(pub), Status: "active"},
		"issuer-2025": {KeyID: "issuer-2025", Algorithm: "ed25519", PublicKey: "base64:" + base64.StdEncoding.EncodeToString(retiredPub), Status: "retired"},
	}}
	server := NewExchangeServer(store).WithKeyRegistry(registry)
	return server, pub, priv, retiredPriv
}

func registryClaim(t *testing.T, server *ExchangeServer, issuerJSON string) *Claim {
	t.Helper()
	ctx := context.Background()
	blob := []byte("registry-fixture-evidence")
	if err := server.store.CreateEvidence(ctx, &Evidence{
		ID: "ev-registry", MediaType: "application/json", BomKind: "sbom",
		Digest: digestFor(blob), SizeBytes: int64(len(blob)), ValidFrom: time.Now().UTC(), Blob: blob,
	}); err != nil {
		t.Fatalf("seed evidence: %v", err)
	}
	claim := &Claim{
		ID: "claim_registry", Type: "artifact.produced_by",
		SubjectJSON: `{"kind":"artifact","digest":"sha256:subject"}`,
		IssuerJSON:  issuerJSON,
		ValidFrom:   time.Now().UTC(), ObservedTime: time.Now().UTC(),
		SourceRefsJSON: `[{"digest":"` + digestFor(blob) + `"}]`,
	}
	if err := server.store.CreateClaim(ctx, claim); err != nil {
		t.Fatalf("create claim: %v", err)
	}
	return claim
}

// registryClaimWithSignature seeds a claim whose issuer JSON is issuerJSON,
// signed with the given Ed25519 key over the canonical claim payload.
func registryClaimWithSignature(t *testing.T, server *ExchangeServer, pub ed25519.PublicKey, signingKey ed25519.PrivateKey, issuerJSON string) *Claim {
	t.Helper()
	claim := registryClaim(t, server, issuerJSON)
	if pub == nil {
		// The signing key is the registry entry for the issuer's key_id.
		var issuer map[string]string
		if err := json.Unmarshal([]byte(issuerJSON), &issuer); err != nil {
			t.Fatalf("issuer json: %v", err)
		}
		entry, ok := server.keyRegistry.Keys[issuer["key_id"]]
		if !ok {
			t.Fatalf("key %q not in fixture registry", issuer["key_id"])
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(entry.PublicKey, "base64:"))
		if err != nil {
			t.Fatalf("decode registry key: %v", err)
		}
		pub = ed25519.PublicKey(raw)
	}
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	sig := ed25519.Sign(signingKey, payload)
	claim.ProofRefsJSON = fmt.Sprintf("[{\"type\":\"signature\",\"ref\":\"ev-sig-%s\"}]", claim.ID)
	sigBlob := []byte(sig)
	if err := server.store.CreateEvidence(context.Background(), &Evidence{
		ID: "ev-sig-" + claim.ID, MediaType: "application/octet-stream", BomKind: "signature",
		Digest: digestFor(sigBlob), SizeBytes: int64(len(sigBlob)), ValidFrom: time.Now().UTC(), Blob: sigBlob,
	}); err != nil {
		t.Fatalf("store signature evidence: %v", err)
	}
	return claim
}

func TestRegistryResolvesKeyID(t *testing.T) {
	server, pub, priv, _ := newRegistryFixture(t)
	claim := registryClaimWithSignature(t, server, pub, priv, `{"kind":"ci","key_id":"issuer-2026"}`)

	result := server.verifyClaimSignatureWithRegistry(context.Background(), claim)
	if result.State != "signature_verified" || result.Provider != "available" {
		t.Fatalf("registry key_id verification failed: %+v", result)
	}
}

func TestRegistryRetiredKeyStillVerifiesButSurfacesRotation(t *testing.T) {
	server, _, _, retiredPriv := newRegistryFixture(t)
	// Sign with the retired key; the issuer references the retired key ID.
	claim := registryClaimWithSignature(t, server, nil, retiredPriv, `{"kind":"ci","key_id":"issuer-2025"}`)

	result := server.verifyClaimSignatureWithRegistry(context.Background(), claim)
	if result.State != "signature_verified" {
		t.Fatalf("a retired key must still verify existing claims: %+v", result)
	}
	if !strings.Contains(result.Message, "retired") {
		t.Fatalf("expected the rotation to surface in diagnostics, got %q", result.Message)
	}
}

func TestRegistryUnknownKeyIDIsUnavailable(t *testing.T) {
	server, pub, priv, _ := newRegistryFixture(t)
	claim := registryClaimWithSignature(t, server, pub, priv, `{"kind":"ci","key_id":"issuer-2099"}`)

	result := server.verifyClaimSignatureWithRegistry(context.Background(), claim)
	if result.State != "unverified" || result.Provider != "unavailable" {
		t.Fatalf("an unknown key_id must be unavailable, not invalid: %+v", result)
	}
}

func TestRegistryInlineKeyWins(t *testing.T) {
	server, _, _, _ := newRegistryFixture(t)
	// Inline key that is NOT in the registry must still verify.
	inlinePub, inlinePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate inline key: %v", err)
	}
	issuerJSON := fmt.Sprintf(`{"kind":"ci","key":"{\"algorithm\":\"ed25519\",\"public_key\":\"base64:%s\"}"}`,
		base64.StdEncoding.EncodeToString(inlinePub))
	claim := registryClaimWithSignature(t, server, inlinePub, inlinePriv, issuerJSON)

	result := server.verifyClaimSignatureWithRegistry(context.Background(), claim)
	if result.State != "signature_verified" {
		t.Fatalf("inline key must win over the registry: %+v", result)
	}
}

// ---- Remote policy ----

func TestRemotePolicyRequiresEnabledFetchPolicy(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store) // no fetch policy
	claim := &Claim{PolicyRefsJSON: `[{"type":"xoscal-remote","url":"https://policy.example.com/p.json"}]`}

	result := server.evaluateRemoteClaimPolicy(context.Background(), claim)
	if result.State != "unevaluated" || result.Provider != "unavailable" {
		t.Fatalf("remote policy without a fetch policy must be unevaluated/unavailable: %+v", result)
	}
}

func TestRemotePolicyFetchesAndEvaluates(t *testing.T) {
	policyDoc := `{"version":"xoscal-policy-v1","allowed_relations":["produced_by"]}`
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(policyDoc))
	}))
	defer ts.Close()

	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store).WithFetchPolicy(&FetchPolicy{Enabled: true, Timeout: 5 * time.Second})
	server.fetchClientFn = func(*FetchPolicy) *http.Client {
		return &http.Client{Transport: ts.Client().Transport}
	}

	claim := &Claim{
		IssuerJSON:     `{"kind":"ci"}`,
		BomKind:        "sbom",
		PredicateJSON:  `{"relation":"produced_by"}`,
		PolicyRefsJSON: fmt.Sprintf(`[{"type":"xoscal-remote","url":"%s/policy.json"}]`, ts.URL),
	}
	result := server.evaluateRemoteClaimPolicy(context.Background(), claim)
	if result.State != "policy_admissible" {
		t.Fatalf("expected the remote policy to admit the claim: %+v", result)
	}
}

func TestRemotePolicyPolicyDenialIsInadmissible(t *testing.T) {
	policyDoc := `{"version":"xoscal-policy-v1","allowed_relations":["built_from"]}`
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(policyDoc))
	}))
	defer ts.Close()

	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store).WithFetchPolicy(&FetchPolicy{Enabled: true, Timeout: 5 * time.Second})
	server.fetchClientFn = func(*FetchPolicy) *http.Client {
		return &http.Client{Transport: ts.Client().Transport}
	}
	claim := &Claim{
		PredicateJSON:  `{"relation":"produced_by"}`,
		PolicyRefsJSON: fmt.Sprintf(`[{"type":"xoscal-remote","url":"%s/p.json"}]`, ts.URL),
	}
	result := server.evaluateRemoteClaimPolicy(context.Background(), claim)
	if result.State != "policy_inadmissible" {
		t.Fatalf("a disallowing remote policy must be inadmissible: %+v", result)
	}
}

// ---- Transparency inclusion ----

func transparencyFixture(t *testing.T) (*ExchangeServer, ed25519.PublicKey, ed25519.PrivateKey, *Claim) {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	logPub, logPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate log key: %v", err)
	}
	cfg := &TransparencyConfig{Enabled: true, LogID: "beta-log", PublicKey: "base64:" + base64.StdEncoding.EncodeToString(logPub)}
	server := NewExchangeServer(store).WithTransparencyConfig(cfg)

	claim := &Claim{
		ID: "claim_transparency", Type: "artifact.produced_by",
		SubjectJSON:   `{"kind":"artifact","digest":"sha256:subject"}`,
		PredicateJSON: `{"relation":"produced_by"}`,
		IssuerJSON:    `{"kind":"ci"}`,
		ValidFrom:     time.Now().UTC(), ObservedTime: time.Now().UTC(),
		SourceRefsJSON: `[]`, PolicyRefsJSON: `[]`,
	}
	if err := server.store.CreateClaim(context.Background(), claim); err != nil {
		t.Fatalf("create claim: %v", err)
	}
	return server, logPub, logPriv, claim
}

func inclusionEvidenceFor(t *testing.T, server *ExchangeServer, claim *Claim, logPriv ed25519.PrivateKey, logID string, mutate func(*transparencyInclusion)) string {
	t.Helper()
	leaf, err := canonicalClaimPayload(claim)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	leafHash := merkleLeafHash(leaf)
	proof := transparencyInclusion{
		Version:  "xoscal-transparency-v1",
		LogID:    logID,
		LogIndex: 0,
		RootHash: fmt.Sprintf("%x", leafHash), // single-leaf tree: root == leaf hash
	}
	proof.Checkpoint.RootHash = proof.RootHash
	proof.Checkpoint.LogSize = 1
	sig := ed25519.Sign(logPriv, canonicalCheckpointBytes(logID, proof.Checkpoint.RootHash, proof.Checkpoint.LogSize))
	proof.Checkpoint.Signature = base64.StdEncoding.EncodeToString(sig)
	if mutate != nil {
		mutate(&proof)
	}
	blob, err := json.Marshal(proof)
	if err != nil {
		t.Fatalf("marshal inclusion: %v", err)
	}
	ev := &Evidence{
		ID: "ev-inclusion", MediaType: "application/json", BomKind: "transparency",
		Digest: digestFor(blob), SizeBytes: int64(len(blob)), ValidFrom: time.Now().UTC(), Blob: blob,
	}
	if err := server.store.CreateEvidence(context.Background(), ev); err != nil {
		t.Fatalf("store inclusion evidence: %v", err)
	}
	return ev.ID
}

func TestTransparencyInclusionVerifies(t *testing.T) {
	server, _, logPriv, claim := transparencyFixture(t)
	evID := inclusionEvidenceFor(t, server, claim, logPriv, "beta-log", nil)
	claim.ProofRefsJSON = fmt.Sprintf(`[{"type":"transparency","ref":%q}]`, evID)

	result := server.verifyTransparencyInclusion(context.Background(), server.store, claim, server.transparencyCfg)
	if result.State != "inclusion_verified" || result.Provider != "available" {
		t.Fatalf("expected inclusion_verified, got %+v", result)
	}
}

func TestTransparencyInclusionFailsClosedWithoutConfig(t *testing.T) {
	server, _, _, claim := transparencyFixture(t)
	server.transparencyCfg = nil

	result := server.verifyTransparencyInclusion(context.Background(), server.store, claim, nil)
	if result.State != "missing" || result.Provider != "unavailable" {
		t.Fatalf("without config the check must stay unavailable: %+v", result)
	}
}

func TestTransparencyInclusionRejectsTamperedLeaf(t *testing.T) {
	server, _, logPriv, claim := transparencyFixture(t)
	evID := inclusionEvidenceFor(t, server, claim, logPriv, "beta-log", func(p *transparencyInclusion) {
		// Corrupt the root so the leaf no longer verifies against it.
		p.RootHash = strings.Repeat("ab", 32)
	})
	claim.ProofRefsJSON = fmt.Sprintf(`[{"type":"transparency","ref":%q}]`, evID)

	result := server.verifyTransparencyInclusion(context.Background(), server.store, claim, server.transparencyCfg)
	if result.State != "invalid" {
		t.Fatalf("a tampered root must be invalid, got %+v", result)
	}
}

func TestTransparencyInclusionRejectsForeignLogKey(t *testing.T) {
	server, _, _, claim := transparencyFixture(t)
	otherPriv := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	evID := inclusionEvidenceFor(t, server, claim, otherPriv, "beta-log", nil)
	claim.ProofRefsJSON = fmt.Sprintf(`[{"type":"transparency","ref":%q}]`, evID)

	result := server.verifyTransparencyInclusion(context.Background(), server.store, claim, server.transparencyCfg)
	if result.State != "invalid" {
		t.Fatalf("a checkpoint from a foreign log key must be invalid, got %+v", result)
	}
}

func TestTransparencyInclusionRejectsWrongLogID(t *testing.T) {
	server, _, logPriv, claim := transparencyFixture(t)
	evID := inclusionEvidenceFor(t, server, claim, logPriv, "other-log", nil)
	claim.ProofRefsJSON = fmt.Sprintf(`[{"type":"transparency","ref":%q}]`, evID)

	result := server.verifyTransparencyInclusion(context.Background(), server.store, claim, server.transparencyCfg)
	if result.State != "invalid" {
		t.Fatalf("an inclusion proof for a different log must be invalid: %+v", result)
	}
}

// ---- ECDSA P-256 / P-384 signature algorithms ----

func ecdsaClaimFixture(t *testing.T, server *ExchangeServer, curve elliptic.Curve, algorithm string) (*Claim, ed25519.PublicKey) {
	t.Helper()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	size := (curve.Params().BitSize + 7) / 8
	point := make([]byte, 2*size)
	private.PublicKey.X.FillBytes(point[:size])
	private.PublicKey.Y.FillBytes(point[size:])

	claim := registryClaim(t, server, fmt.Sprintf(`{"kind":"ci","key":"{\"algorithm\":%q,\"public_key\":\"base64:%s\"}"}`,
		algorithm, base64.StdEncoding.EncodeToString(point)))
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	h := sha256.Sum256(payload)
	sigDER, err := ecdsa.SignASN1(rand.Reader, private, h[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Parse the DER signature into raw r||s.
	var der struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(sigDER, &der); err != nil {
		t.Fatalf("parse der: %v", err)
	}
	byteLen := size / 2
	raw := make([]byte, 2*byteLen)
	der.R.FillBytes(raw[:byteLen])
	der.S.FillBytes(raw[byteLen:])

	claim.ProofRefsJSON = fmt.Sprintf("[{\"type\":\"signature\",\"ref\":\"ev-sig-%s\"}]", claim.ID)
	if err := server.store.CreateEvidence(context.Background(), &Evidence{
		ID: "ev-sig-" + claim.ID, MediaType: "application/octet-stream", BomKind: "signature",
		Digest: digestFor(raw), SizeBytes: int64(len(raw)), ValidFrom: time.Now().UTC(), Blob: raw,
	}); err != nil {
		t.Fatalf("store signature evidence: %v", err)
	}
	return claim, nil
}

func TestECDSAP256SignatureVerifies(t *testing.T) {
	server, _, _, _ := newRegistryFixture(t)
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	point := make([]byte, 64)
	curve.Params().BitSize = curve.Params().BitSize // no-op clarity
	private.PublicKey.X.FillBytes(point[:32])
	private.PublicKey.Y.FillBytes(point[32:])
	issuerJSON := fmt.Sprintf(`{"kind":"ci","key":"{\"algorithm\":\"ecdsa-p256\",\"public_key\":\"base64:%s\"}"}`,
		base64.StdEncoding.EncodeToString(point))
	claim := registryClaim(t, server, issuerJSON)
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	h := sha256.Sum256(payload)
	r, s, err := ecdsa.Sign(rand.Reader, private, h[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	raw := make([]byte, 64)
	r.FillBytes(raw[:32])
	s.FillBytes(raw[32:])
	sigBlob := raw
	if err := server.store.CreateEvidence(context.Background(), &Evidence{
		ID: "ev-sig-ecdsa", MediaType: "application/octet-stream", BomKind: "signature",
		Digest: digestFor(sigBlob), SizeBytes: int64(len(sigBlob)), ValidFrom: time.Now().UTC(), Blob: sigBlob,
	}); err != nil {
		t.Fatalf("store signature: %v", err)
	}
	claim.ProofRefsJSON = `[{"type":"signature","ref":"ev-sig-ecdsa"}]`

	result := server.verifyClaimSignatureWithRegistry(context.Background(), claim)
	if result.State != "signature_verified" {
		t.Fatalf("ECDSA P-256 signature must verify: %+v", result)
	}
}

func TestECDSAP384SignatureVerifies(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	private, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	point := make([]byte, 96)
	private.PublicKey.X.FillBytes(point[:48])
	private.PublicKey.Y.FillBytes(point[48:])
	claim := &Claim{
		ID: "claim-p384", Type: "artifact.produced_by",
		SubjectJSON: `{"kind":"artifact","digest":"sha256:s"}`,
		IssuerJSON: fmt.Sprintf(`{"kind":"ci","key":"{\"algorithm\":\"ecdsa-p384\",\"public_key\":\"base64:%s\"}"}`,
			base64.StdEncoding.EncodeToString(point)),
		ValidFrom: time.Now().UTC(), ObservedTime: time.Now().UTC(),
	}
	payload, err := canonicalClaimPayload(claim)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	h := sha256.Sum256(payload)
	sigDER, err := ecdsa.SignASN1(rand.Reader, private, h[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	var der struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(sigDER, &der); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw := make([]byte, 96)
	der.R.FillBytes(raw[:48])
	der.S.FillBytes(raw[48:])

	verifier, err := verifierFromKeyMaterial("ecdsa-p384", "base64:"+base64.StdEncoding.EncodeToString(point))
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if got := verifySignatureAgainstKey(claim, raw, verifier, false); got.State != "signature_verified" {
		t.Fatalf("P-384 signature must verify: %+v", got)
	}
}

func TestUnsupportedAlgorithmIsRejected(t *testing.T) {
	_, err := verifierFromKeyMaterial("rsa-2048", "base64:AAAA")
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("an unsupported algorithm must be rejected explicitly, got %v", err)
	}
}
