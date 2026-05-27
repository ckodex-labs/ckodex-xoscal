package transparency

import (
	"context"
	"testing"
	"time"
)

func TestStore_ClaimLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	claim := &Claim{
		ID:             "claim_01HXTEST",
		Type:           "artifact.produced_by",
		SubjectJSON:    `{"kind":"artifact","digest":"sha256:abc123"}`,
		PredicateJSON:  `{"relation":"produced_by"}`,
		IssuerJSON:     `{"kind":"workload","id":"spiffe://test/workload"}`,
		BomKind:        "sbom",
		ValidFrom:      time.Now().UTC(),
		ObservedTime:   time.Now().UTC(),
		SourceRefsJSON: `[{"ref":"urn:test:evidence:1","digest":"sha256:abc123","media_type":"application/json"}]`,
		TrustState:     "candidate",
	}

	if err := store.CreateClaim(ctx, claim); err != nil {
		t.Fatalf("create claim: %v", err)
	}

	got, err := store.GetClaim(ctx, claim.ID)
	if err != nil {
		t.Fatalf("get claim: %v", err)
	}
	if got.ID != claim.ID {
		t.Fatalf("expected id %s, got %s", claim.ID, got.ID)
	}
	if got.Type != claim.Type {
		t.Fatalf("expected type %s, got %s", claim.Type, got.Type)
	}

	claims, err := store.ListClaims(ctx, "", "sbom", "", "", time.Time{}, 10)
	if err != nil {
		t.Fatalf("list claims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}

	if err := store.UpdateClaimTrustState(ctx, claim.ID, "verified", `{"source":"source_bound"}`); err != nil {
		t.Fatalf("update trust state: %v", err)
	}
	got, err = store.GetClaim(ctx, claim.ID)
	if err != nil {
		t.Fatalf("get claim after update: %v", err)
	}
	if got.TrustState != "verified" {
		t.Fatalf("expected trust_state verified, got %s", got.TrustState)
	}
}

func TestStore_EvidenceLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ev := &Evidence{
		ID:          "evidence_01HXTEST",
		MediaType:   "application/vnd.cyclonedx+json",
		BomKind:     "sbom",
		Digest:      "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SizeBytes:   1024,
		StorageJSON: `{"uris":["https://example.com/evidence.json"],"fetch_policy":"any"}`,
		ValidFrom:   time.Now().UTC(),
	}

	if err := store.CreateEvidence(ctx, ev); err != nil {
		t.Fatalf("create evidence: %v", err)
	}

	got, err := store.GetEvidence(ctx, ev.ID)
	if err != nil {
		t.Fatalf("get evidence: %v", err)
	}
	if got.ID != ev.ID {
		t.Fatalf("expected id %s, got %s", ev.ID, got.ID)
	}

	gotByDigest, err := store.GetEvidenceByDigest(ctx, ev.Digest)
	if err != nil {
		t.Fatalf("get evidence by digest: %v", err)
	}
	if gotByDigest.ID != ev.ID {
		t.Fatalf("expected id %s from digest lookup, got %s", ev.ID, gotByDigest.ID)
	}
}
