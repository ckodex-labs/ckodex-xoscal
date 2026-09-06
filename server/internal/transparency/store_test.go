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

	claims, _, err := store.ListClaims(ctx, "", "sbom", "", "", time.Time{}, 10, "")
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

func TestStore_ListClaimsFiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	base := time.Now().UTC().Add(-3 * time.Minute)
	claims := []*Claim{
		{ID: "claim_page_1", SubjectJSON: `{"digest":"sha256:one"}`, PredicateJSON: `{"relation":"depends_on"}`, BomKind: "sbom", ValidFrom: base, ObservedTime: base, SourceRefsJSON: `[]`, TrustState: "candidate", CreatedAt: base},
		{ID: "claim_page_2", SubjectJSON: `{"digest":"sha256:two"}`, PredicateJSON: `{"relation":"produced_by"}`, BomKind: "sbom", ValidFrom: base.Add(time.Minute), ObservedTime: base.Add(time.Minute), SourceRefsJSON: `[]`, TrustState: "incomplete", CreatedAt: base.Add(time.Minute)},
		{ID: "claim_page_3", SubjectJSON: `{"digest":"sha256:two"}`, PredicateJSON: `{"relation":"produced_by"}`, BomKind: "sbom", ValidFrom: base.Add(2 * time.Minute), ObservedTime: base.Add(2 * time.Minute), SourceRefsJSON: `[]`, TrustState: "verified", CreatedAt: base.Add(2 * time.Minute)},
	}
	for _, claim := range claims {
		if err := store.CreateClaim(ctx, claim); err != nil {
			t.Fatalf("create claim %s: %v", claim.ID, err)
		}
	}

	page, token, err := store.ListClaims(ctx, "sha256:two", "sbom", "produced_by", "", time.Time{}, 1, "")
	if err != nil {
		t.Fatalf("list filtered claims: %v", err)
	}
	if len(page) != 1 || page[0].ID != "claim_page_3" || token == "" {
		t.Fatalf("unexpected first page: len=%d id=%q token=%q", len(page), page[0].ID, token)
	}
	page, token, err = store.ListClaims(ctx, "sha256:two", "sbom", "produced_by", "", time.Time{}, 1, token)
	if err != nil {
		t.Fatalf("list second filtered page: %v", err)
	}
	if len(page) != 1 || page[0].ID != "claim_page_2" || token != "" {
		t.Fatalf("unexpected second page: len=%d id=%q token=%q", len(page), page[0].ID, token)
	}
}

func TestStore_RecordVerificationIsAtomicIdempotentAndChained(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	claim := &Claim{
		ID:             "claim_audit_1",
		SubjectJSON:    `{ "digest": "sha256:audit" }`,
		PredicateJSON:  `{ "relation": "audited" }`,
		IssuerJSON:     `{}`,
		ValidFrom:      time.Now().UTC(),
		ObservedTime:   time.Now().UTC(),
		SourceRefsJSON: `[]`,
		TrustState:     "candidate",
		ProofStateJSON: `{}`,
		CreatedAt:      time.Now().UTC(),
	}
	if err := store.CreateClaim(ctx, claim); err != nil {
		t.Fatalf("create claim: %v", err)
	}

	first, err := store.RecordVerification(ctx, claim.ID, "incomplete", `{"state":"first"}`, &VerificationEvent{
		EventID:           "verification_audit_1",
		RequestID:         "request_audit_1",
		IdempotencyKey:    "profile_audit_1",
		ChecksJSON:        `["digest"]`,
		ProviderStateJSON: `{"digest":"available"}`,
		DiagnosticsJSON:   `[]`,
	})
	if err != nil {
		t.Fatalf("record first verification: %v", err)
	}
	if first.EventHash == "" || first.PreviousHash != "" {
		t.Fatalf("unexpected first chain values: previous=%q hash=%q", first.PreviousHash, first.EventHash)
	}

	repeated, err := store.RecordVerification(ctx, claim.ID, "incomplete", `{"state":"different"}`, &VerificationEvent{
		EventID:           "verification_audit_repeat",
		RequestID:         "request_audit_repeat",
		IdempotencyKey:    "profile_audit_1",
		ChecksJSON:        `["digest"]`,
		ProviderStateJSON: `{"digest":"available"}`,
		DiagnosticsJSON:   `[]`,
	})
	if err != nil {
		t.Fatalf("record repeated verification: %v", err)
	}
	if repeated.EventID != first.EventID || repeated.EventHash != first.EventHash {
		t.Fatalf("idempotency returned a different event: first=%+v repeated=%+v", first, repeated)
	}

	second, err := store.RecordVerification(ctx, claim.ID, "rejected", `{"state":"second"}`, &VerificationEvent{
		EventID:           "verification_audit_2",
		RequestID:         "request_audit_2",
		IdempotencyKey:    "profile_audit_2",
		ChecksJSON:        `["digest"]`,
		ProviderStateJSON: `{"digest":"invalid"}`,
		DiagnosticsJSON:   `["invalid"]`,
	})
	if err != nil {
		t.Fatalf("record second verification: %v", err)
	}
	if second.PreviousHash != first.EventHash || second.EventHash == first.EventHash {
		t.Fatalf("verification chain did not advance: first=%+v second=%+v", first, second)
	}
	if err := store.VerifyVerificationChain(ctx); err != nil {
		t.Fatalf("verify event chain: %v", err)
	}
	if _, err := store.(*SQLiteStore).db.ExecContext(ctx,
		`UPDATE verification_events SET trust_state = 'tampered' WHERE event_id = ?`, first.EventID); err == nil {
		t.Fatal("append-only verification event was mutable")
	}

	events, err := store.ListVerificationEvents(ctx, claim.ID)
	if err != nil {
		t.Fatalf("list verification events: %v", err)
	}
	if len(events) != 2 || events[0].EventID != first.EventID || events[1].EventID != second.EventID {
		t.Fatalf("unexpected audit history: %+v", events)
	}
	got, err := store.GetClaim(ctx, claim.ID)
	if err != nil {
		t.Fatalf("get claim after verification: %v", err)
	}
	if got.TrustState != "rejected" || got.ProofStateJSON != `{"state":"second"}` {
		t.Fatalf("claim projection does not match latest event: %+v", got)
	}
}
