package transparency

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExchangeEvidenceVerificationIsContentAddressed(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	server := NewExchangeServer(store)
	blob := []byte("beta evidence")
	digest := digestFor(blob)

	_, err = server.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{Evidence: &servicesv1.Evidence{
		Id:        "evidence_beta_1",
		MediaType: "application/json",
		BomKind:   "sbom",
		Digest:    digest,
		SizeBytes: int64(len(blob)),
	}, Blob: blob})
	if err != nil {
		t.Fatalf("upload evidence: %v", err)
	}

	verified, err := server.VerifyEvidence(ctx, &servicesv1.VerifyEvidenceRequest{
		EvidenceId:   "evidence_beta_1",
		FetchAndHash: true,
	})
	if err != nil {
		t.Fatalf("verify evidence: %v", err)
	}
	if !verified.DigestOk || !verified.SizeOk || verified.Error != "" {
		t.Fatalf("expected verified content, got digest=%v size=%v error=%q", verified.DigestOk, verified.SizeOk, verified.Error)
	}
}

func TestExchangeRejectsEvidenceDigestMismatch(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	server := NewExchangeServer(store)
	_, err = server.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{Evidence: &servicesv1.Evidence{
		Id:        "evidence_beta_bad",
		MediaType: "application/json",
		BomKind:   "sbom",
		Digest:    "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		SizeBytes: 5,
	}, Blob: []byte("hello")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestExchangeVerificationFailsClosedWhenProvidersAreUnavailable(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	server := NewExchangeServer(store)
	blob := []byte("claim evidence")
	_, err = server.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{Evidence: &servicesv1.Evidence{
		Id:        "evidence_beta_claim",
		MediaType: "application/json",
		BomKind:   "sbom",
		Digest:    digestFor(blob),
		SizeBytes: int64(len(blob)),
	}, Blob: blob})
	if err != nil {
		t.Fatalf("upload evidence: %v", err)
	}

	created, err := server.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: &servicesv1.Claim{
		Id:      "claim_beta_1",
		Type:    "artifact.produced_by",
		Subject: &servicesv1.Reference{Kind: "artifact", Digest: "sha256:subject"},
		Predicate: &servicesv1.Predicate{
			Relation: "produced_by",
		},
		SourceRefs: []*servicesv1.EvidenceRef{{Digest: digestFor(blob)}},
	}})
	if err != nil {
		t.Fatalf("create claim: %v", err)
	}
	if created.TrustState != "candidate" {
		t.Fatalf("initial trust state = %q, want candidate", created.TrustState)
	}
	if created.GetClaim().GetTrustState() != "candidate" || created.GetClaim().GetProofStateJson() == "" {
		t.Fatalf("create response omitted current trust projection: %+v", created.GetClaim())
	}

	verified, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: "claim_beta_1"})
	if err != nil {
		t.Fatalf("verify claim: %v", err)
	}
	if verified.TrustState != "incomplete" {
		t.Fatalf("trust state = %q, want incomplete", verified.TrustState)
	}
	if len(verified.Diagnostics) < 2 {
		t.Fatalf("diagnostics = %v, want unavailable signature and policy diagnostics", verified.Diagnostics)
	}
	if verified.ProofState.GetSignature() == "signature_verified" || verified.ProofState.GetPolicy() == "policy_admissible" {
		t.Fatal("unsupported providers must not produce verified proof fields")
	}

	repeated, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: "claim_beta_1"})
	if err != nil {
		t.Fatalf("repeat verify claim: %v", err)
	}
	if repeated.TrustState != verified.TrustState || len(repeated.Diagnostics) != len(verified.Diagnostics) {
		t.Fatalf("idempotent verification changed response: first=%+v repeated=%+v", verified, repeated)
	}
	events, err := store.ListVerificationEvents(ctx, "claim_beta_1")
	if err != nil {
		t.Fatalf("list verification audit events: %v", err)
	}
	if len(events) != 1 || events[0].EventHash == "" || events[0].RequestID == "" {
		t.Fatalf("expected one chained verification event with request identity, got %+v", events)
	}
	listed, _, err := store.ListClaims(ctx, "", "", "", "", time.Time{}, 10, "")
	if err != nil {
		t.Fatalf("list claims: %v", err)
	}
	if len(listed) != 1 || listed[0].TrustState != "incomplete" || listed[0].ProofStateJSON == "" {
		t.Fatalf("list response projection is stale: %+v", listed)
	}
}

func TestExchangeProvidersVerifySignedClaimAndPolicy(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	policyBlob := []byte(`{"version":"xoscal-policy-v1","allowed_relations":["produced_by"],"allowed_bom_kinds":["sbom"],"allowed_issuer_kinds":["workload"]}`)
	sourceBlob := []byte("signed source")
	policyDigest := digestFor(policyBlob)
	sourceDigest := digestFor(sourceBlob)
	issuerKey, err := json.Marshal(map[string]string{
		"algorithm":  "ed25519",
		"public_key": base64.StdEncoding.EncodeToString(publicKey),
	})
	if err != nil {
		t.Fatalf("encode issuer key: %v", err)
	}
	claimProto := &servicesv1.Claim{
		Id:      "claim_provider_1",
		Type:    "artifact.produced_by",
		Subject: &servicesv1.Reference{Kind: "artifact", Digest: "sha256:subject-provider"},
		Predicate: &servicesv1.Predicate{
			Relation: "produced_by",
		},
		Issuer:  &servicesv1.Identity{Kind: "workload", Id: "spiffe://beta/provider", KeyJson: string(issuerKey)},
		BomKind: "sbom",
		ValidTime: &servicesv1.TimeWindow{
			FromTime: timestamppb.New(time.Now().UTC()),
		},
		ObservedTime: timestamppb.New(time.Now().UTC()),
		SourceRefs:   []*servicesv1.EvidenceRef{{Ref: "source_provider_1", Digest: sourceDigest}},
		PolicyRefs:   []*servicesv1.PolicyRef{{Type: "xoscal-json", Ref: "policy_provider_1", Digest: policyDigest}},
	}
	claimDB := claimFromProto(claimProto)
	payload, err := canonicalClaimPayload(claimDB)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	signature := ed25519.Sign(privateKey, payload)
	signatureDigest := digestFor(signature)
	claimProto.ProofRefs = []*servicesv1.ProofRef{{Type: "signature", Ref: "signature_provider_1", Digest: signatureDigest}}

	for _, evidence := range []*servicesv1.UploadEvidenceRequest{
		{Evidence: &servicesv1.Evidence{Id: "source_provider_1", MediaType: "text/plain", BomKind: "sbom", Digest: sourceDigest, SizeBytes: int64(len(sourceBlob))}, Blob: sourceBlob},
		{Evidence: &servicesv1.Evidence{Id: "policy_provider_1", MediaType: "application/json", BomKind: "policy", Digest: policyDigest, SizeBytes: int64(len(policyBlob))}, Blob: policyBlob},
		{Evidence: &servicesv1.Evidence{Id: "signature_provider_1", MediaType: "application/octet-stream", BomKind: "signature", Digest: signatureDigest, SizeBytes: int64(len(signature))}, Blob: signature},
	} {
		if _, err := server.UploadEvidence(ctx, evidence); err != nil {
			t.Fatalf("upload provider evidence: %v", err)
		}
	}
	if _, err := server.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: claimProto}); err != nil {
		t.Fatalf("create signed claim: %v", err)
	}
	verified, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: claimProto.Id})
	if err != nil {
		t.Fatalf("verify signed claim: %v", err)
	}
	if verified.TrustState != "verified" || verified.ProofState.GetSignature() != "signature_verified" || verified.ProofState.GetPolicy() != "policy_admissible" {
		t.Fatalf("provider verification did not pass: %+v", verified)
	}
	if len(verified.Diagnostics) != 0 {
		t.Fatalf("verified claim returned diagnostics: %v", verified.Diagnostics)
	}
}

func TestExchangeRejectsInvalidClaimSignature(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	badSignature := make([]byte, ed25519.SignatureSize)
	claim := &servicesv1.Claim{
		Id:         "claim_provider_bad_signature",
		Type:       "artifact.produced_by",
		Subject:    &servicesv1.Reference{Kind: "artifact", Digest: "sha256:subject-bad"},
		Predicate:  &servicesv1.Predicate{Relation: "produced_by"},
		Issuer:     &servicesv1.Identity{Kind: "workload", KeyJson: `{"algorithm":"ed25519","public_key":"` + base64.StdEncoding.EncodeToString(publicKey) + `"}`},
		SourceRefs: []*servicesv1.EvidenceRef{{Digest: digestFor([]byte("bad source"))}},
	}
	claim.ProofRefs = []*servicesv1.ProofRef{{Type: "signature", Ref: "bad_signature", Digest: digestFor(badSignature)}}
	if _, err := server.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{Evidence: &servicesv1.Evidence{Id: "bad_signature", MediaType: "application/octet-stream", BomKind: "signature", Digest: digestFor(badSignature), SizeBytes: int64(len(badSignature))}, Blob: badSignature}); err != nil {
		t.Fatalf("upload bad signature: %v", err)
	}
	if _, err := server.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: claim}); err != nil {
		t.Fatalf("create invalid signed claim: %v", err)
	}
	verified, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: claim.Id, Checks: []string{"signature"}})
	if err != nil {
		t.Fatalf("verify invalid signed claim: %v", err)
	}
	if verified.ProofState.GetSignature() != "invalid" || verified.TrustState != "rejected" {
		t.Fatalf("invalid signature was not rejected: %+v", verified)
	}
}

func TestExchangeReceiptExportRequiresVerifiedClaim(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)
	blob := []byte("receipt gate evidence")
	digest := digestFor(blob)
	if _, err := server.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{
		Evidence: &servicesv1.Evidence{Id: "receipt_gate_evidence", MediaType: "text/plain", BomKind: "sbom", Digest: digest, SizeBytes: int64(len(blob))},
		Blob:     blob,
	}); err != nil {
		t.Fatalf("upload evidence: %v", err)
	}
	if _, err := server.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: &servicesv1.Claim{
		Id:         "receipt_gate_claim",
		Type:       "artifact.produced_by",
		Subject:    &servicesv1.Reference{Kind: "artifact", Digest: "sha256:receipt-gate-subject"},
		Predicate:  &servicesv1.Predicate{Relation: "produced_by"},
		SourceRefs: []*servicesv1.EvidenceRef{{Ref: "receipt_gate_evidence", Digest: digest}},
	}}); err != nil {
		t.Fatalf("create claim: %v", err)
	}
	if _, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: "receipt_gate_claim"}); err != nil {
		t.Fatalf("verify claim: %v", err)
	}
	if _, err := server.ExportClaimReceipt(ctx, &servicesv1.ExportClaimReceiptRequest{ClaimId: "receipt_gate_claim"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("receipt export error = %v, want FailedPrecondition", err)
	}

	audit, err := server.ListVerificationEvents(ctx, &servicesv1.ListVerificationEventsRequest{ClaimId: "receipt_gate_claim"})
	if err != nil {
		t.Fatalf("list verification events: %v", err)
	}
	if !audit.ChainValid || len(audit.Events) != 1 || audit.Events[0].GetEventHash() == "" {
		t.Fatalf("unexpected audit response: %+v", audit)
	}
}

func TestExchangeReceiptExportContainsProofAndAuditDigest(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)
	claimID := seedVerifiedProviderClaim(t, ctx, server)

	receipt, err := server.ExportClaimReceipt(ctx, &servicesv1.ExportClaimReceiptRequest{ClaimId: claimID})
	if err != nil {
		t.Fatalf("export receipt: %v", err)
	}
	if receipt.GetTrustState() != "verified" || receipt.GetReceiptDigest() != digestFor([]byte(receipt.GetReceiptJson())) {
		t.Fatalf("receipt metadata is not self-consistent: %+v", receipt)
	}
	var document map[string]interface{}
	if err := json.Unmarshal([]byte(receipt.GetReceiptJson()), &document); err != nil {
		t.Fatalf("decode receipt JSON: %v", err)
	}
	if document["schema_version"] != "xoscal-beta-receipt-v1" || document["trust_state"] != "verified" {
		t.Fatalf("receipt contract fields missing: %v", document)
	}
	evidence, ok := document["evidence"].([]interface{})
	if !ok || len(evidence) != 3 {
		t.Fatalf("receipt evidence = %v, want source, signature, and policy metadata", document["evidence"])
	}
	if _, hasBlob := evidence[0].(map[string]interface{})["blob"]; hasBlob {
		t.Fatal("receipt must not export raw evidence bytes")
	}
	audit, err := server.ListVerificationEvents(ctx, &servicesv1.ListVerificationEventsRequest{ClaimId: claimID})
	if err != nil {
		t.Fatalf("list verification events: %v", err)
	}
	if len(audit.Events) != 1 || audit.Events[0].GetEventHash() != receipt.GetVerificationEventHash() {
		t.Fatalf("receipt is not bound to audit head: receipt=%q audit=%+v", receipt.GetVerificationEventHash(), audit.Events)
	}
	if event, ok := document["verification_event"].(map[string]interface{}); !ok || event["event_hash"] != receipt.GetVerificationEventHash() {
		t.Fatalf("receipt event hash missing from payload: %v", document["verification_event"])
	}
}

func TestExchangeImportPreflightAndBatchAreAllOrNothing(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)
	goodBlob := []byte("good batch evidence")
	goodDigest := digestFor(goodBlob)
	badBlob := []byte("bad batch evidence")
	badDigest := digestFor([]byte("different bytes"))
	invalidBatch := &servicesv1.PreflightImportRequest{Records: []*servicesv1.ImportRecord{
		importRecord("batch_claim_good", "batch_evidence_good", goodBlob, goodDigest),
		importRecord("batch_claim_bad", "batch_evidence_bad", badBlob, badDigest),
	}}
	preflight, err := server.PreflightImport(ctx, invalidBatch)
	if err != nil {
		t.Fatalf("preflight import: %v", err)
	}
	if preflight.GetValid() || len(preflight.GetResults()) != 2 || preflight.GetResults()[1].GetStatus() != "invalid" {
		t.Fatalf("preflight should reject only the invalid batch: %+v", preflight)
	}
	if _, err := store.GetClaim(ctx, "batch_claim_good"); err != sql.ErrNoRows {
		t.Fatalf("preflight mutated claim store: %v", err)
	}
	rolledBack, err := server.ImportBatch(ctx, &servicesv1.ImportBatchRequest{Records: invalidBatch.GetRecords(), AllOrNothing: true})
	if err != nil {
		t.Fatalf("invalid batch response: %v", err)
	}
	if rolledBack.GetCommitted() {
		t.Fatal("invalid batch must not commit")
	}
	if _, err := store.GetEvidence(ctx, "batch_evidence_good"); err != sql.ErrNoRows {
		t.Fatalf("rolled-back batch left evidence behind: %v", err)
	}
	tooMany := &servicesv1.PreflightImportRequest{Records: make([]*servicesv1.ImportRecord, maxImportRecords+1)}
	bounded, err := server.PreflightImport(ctx, tooMany)
	if err != nil || bounded.GetValid() || len(bounded.GetResults()) != 1 || bounded.GetResults()[0].GetStatus() != "invalid" {
		t.Fatalf("oversized batch should fail closed: response=%+v err=%v", bounded, err)
	}
}

func TestExchangeImportBatchCommitsAndRepeatsIdempotently(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()
	server := NewExchangeServer(store)
	blob := []byte("idempotent batch evidence")
	digest := digestFor(blob)
	record := importRecord("batch_claim_one", "batch_evidence_one", blob, digest)
	preflight, err := server.PreflightImport(ctx, &servicesv1.PreflightImportRequest{Records: []*servicesv1.ImportRecord{record}})
	if err != nil || !preflight.GetValid() || preflight.GetResults()[0].GetStatus() != "ready" {
		t.Fatalf("valid preflight failed: response=%+v err=%v", preflight, err)
	}
	committed, err := server.ImportBatch(ctx, &servicesv1.ImportBatchRequest{Records: []*servicesv1.ImportRecord{record}, AllOrNothing: true})
	if err != nil || !committed.GetCommitted() || committed.GetResults()[0].GetStatus() != "imported" {
		t.Fatalf("batch did not commit: response=%+v err=%v", committed, err)
	}
	repeated, err := server.ImportBatch(ctx, &servicesv1.ImportBatchRequest{Records: []*servicesv1.ImportRecord{record}, AllOrNothing: true})
	if err != nil || !repeated.GetCommitted() || repeated.GetResults()[0].GetStatus() != "already_present" {
		t.Fatalf("repeat batch was not idempotent: response=%+v err=%v", repeated, err)
	}
	if _, err := server.ImportBatch(ctx, &servicesv1.ImportBatchRequest{Records: []*servicesv1.ImportRecord{record}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing all_or_nothing error = %v, want InvalidArgument", err)
	}
	claims, _, err := store.ListClaims(ctx, "", "", "", "", time.Time{}, 10, "")
	if err != nil || len(claims) != 1 {
		t.Fatalf("expected one idempotently stored claim, got claims=%v err=%v", claims, err)
	}
}

func importRecord(claimID, evidenceID string, blob []byte, digest string) *servicesv1.ImportRecord {
	return &servicesv1.ImportRecord{
		Claim: &servicesv1.Claim{
			Id:         claimID,
			Type:       "artifact.produced_by",
			Subject:    &servicesv1.Reference{Kind: "artifact", Id: claimID + ":subject"},
			Predicate:  &servicesv1.Predicate{Relation: "produced_by"},
			SourceRefs: []*servicesv1.EvidenceRef{{Ref: evidenceID, Digest: digest, MediaType: "text/plain", BomKind: "sbom"}},
		},
		Evidence: []*servicesv1.ImportEvidence{{
			Evidence: &servicesv1.Evidence{Id: evidenceID, MediaType: "text/plain", BomKind: "sbom", Digest: digest, SizeBytes: int64(len(blob))},
			Blob:     blob,
		}},
	}
}

func seedVerifiedProviderClaim(t *testing.T, ctx context.Context, server *ExchangeServer) string {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	sourceBlob := []byte("receipt source")
	policyBlob := []byte(`{"version":"xoscal-policy-v1","allowed_relations":["produced_by"],"allowed_bom_kinds":["sbom"],"allowed_issuer_kinds":["workload"]}`)
	sourceDigest := digestFor(sourceBlob)
	policyDigest := digestFor(policyBlob)
	issuerKey, err := json.Marshal(map[string]string{
		"algorithm":  "ed25519",
		"public_key": base64.StdEncoding.EncodeToString(publicKey),
	})
	if err != nil {
		t.Fatalf("encode issuer key: %v", err)
	}
	now := time.Now().UTC()
	claim := &servicesv1.Claim{
		Id:           "claim_receipt_verified",
		Type:         "artifact.produced_by",
		Subject:      &servicesv1.Reference{Kind: "artifact", Digest: "sha256:receipt-subject"},
		Predicate:    &servicesv1.Predicate{Relation: "produced_by"},
		Issuer:       &servicesv1.Identity{Kind: "workload", Id: "beta/receipt", KeyJson: string(issuerKey)},
		BomKind:      "sbom",
		ValidTime:    &servicesv1.TimeWindow{FromTime: timestamppb.New(now)},
		ObservedTime: timestamppb.New(now),
		SourceRefs: []*servicesv1.EvidenceRef{{
			Ref: "receipt_source", Digest: sourceDigest, MediaType: "text/plain", BomKind: "sbom",
		}},
		PolicyRefs: []*servicesv1.PolicyRef{{Type: "xoscal-json", Ref: "receipt_policy", Digest: policyDigest}},
	}
	claimDB := claimFromProto(claim)
	payload, err := canonicalClaimPayload(claimDB)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	signature := ed25519.Sign(privateKey, payload)
	signatureDigest := digestFor(signature)
	claim.ProofRefs = []*servicesv1.ProofRef{{Type: "signature", Ref: "receipt_signature", Digest: signatureDigest}}
	for _, evidence := range []*servicesv1.UploadEvidenceRequest{
		{Evidence: &servicesv1.Evidence{Id: "receipt_source", MediaType: "text/plain", BomKind: "sbom", Digest: sourceDigest, SizeBytes: int64(len(sourceBlob))}, Blob: sourceBlob},
		{Evidence: &servicesv1.Evidence{Id: "receipt_policy", MediaType: "application/json", BomKind: "policy", Digest: policyDigest, SizeBytes: int64(len(policyBlob))}, Blob: policyBlob},
		{Evidence: &servicesv1.Evidence{Id: "receipt_signature", MediaType: "application/octet-stream", BomKind: "signature", Digest: signatureDigest, SizeBytes: int64(len(signature))}, Blob: signature},
	} {
		if _, err := server.UploadEvidence(ctx, evidence); err != nil {
			t.Fatalf("upload receipt evidence: %v", err)
		}
	}
	if _, err := server.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: claim}); err != nil {
		t.Fatalf("create receipt claim: %v", err)
	}
	verified, err := server.VerifyClaim(ctx, &servicesv1.VerifyClaimRequest{ClaimId: claim.Id})
	if err != nil {
		t.Fatalf("verify receipt claim: %v", err)
	}
	if verified.GetTrustState() != "verified" {
		t.Fatalf("seed claim trust state = %q, want verified: proof=%+v diagnostics=%v", verified.GetTrustState(), verified.GetProofState(), verified.GetDiagnostics())
	}
	return claim.Id
}
