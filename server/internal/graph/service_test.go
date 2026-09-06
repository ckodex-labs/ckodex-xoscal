package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/transparency"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGraphProjectionUsesClaimIdentityAndTrustState(t *testing.T) {
	ctx := context.Background()
	graphStore, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new graph store: %v", err)
	}
	defer graphStore.Close()
	exchangeStore, err := transparency.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new exchange store: %v", err)
	}
	defer exchangeStore.Close()

	exchange := transparency.NewExchangeServer(exchangeStore)
	blob := []byte("graph-evidence")
	sum := sha256.Sum256(blob)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	_, err = exchange.UploadEvidence(ctx, &servicesv1.UploadEvidenceRequest{
		Evidence: &servicesv1.Evidence{Id: "evidence_graph_1", BomKind: "sbom", Digest: digest, SizeBytes: int64(len(blob))},
		Blob:     blob,
	})
	if err != nil {
		t.Fatalf("upload evidence: %v", err)
	}
	_, err = exchange.CreateClaim(ctx, &servicesv1.CreateClaimRequest{Claim: &servicesv1.Claim{
		Id:      "claim_graph_1",
		Type:    "artifact.depends_on",
		Subject: &servicesv1.Reference{Kind: "artifact", Id: "app"},
		Predicate: &servicesv1.Predicate{
			Relation: "depends_on",
		},
		ObjectOpt:  &servicesv1.Claim_Object{Object: &servicesv1.Reference{Kind: "component", Id: "lib"}},
		SourceRefs: []*servicesv1.EvidenceRef{{Ref: "evidence_graph_1", Digest: digest, BomKind: "sbom"}},
	}})
	if err != nil {
		t.Fatalf("create claim: %v", err)
	}

	server := NewGraphServer(graphStore, exchangeStore)
	projected, err := server.ProjectEdge(ctx, &servicesv1.ProjectEdgeRequest{ClaimId: "claim_graph_1", EdgeId: "edge_graph_1"})
	if err != nil {
		t.Fatalf("project edge: %v", err)
	}
	if projected.Edge.GetFromNode() != "artifact:app" || projected.Edge.GetToNode() != "component:lib" {
		t.Fatalf("unexpected edge endpoints: %s -> %s", projected.Edge.GetFromNode(), projected.Edge.GetToNode())
	}
	if projected.Edge.GetRelation() != "depends_on" || projected.Edge.GetTrustState() != "candidate" {
		t.Fatalf("unexpected edge relation/trust: %s/%s", projected.Edge.GetRelation(), projected.Edge.GetTrustState())
	}
	if projected.GetProjectionEvent().GetEventHash() == "" || projected.GetProjectionEvent().GetEdgeId() != "edge_graph_1" {
		t.Fatalf("projection response is missing its audit event: %+v", projected.GetProjectionEvent())
	}
	audit, err := server.ListProjectionEvents(ctx, &servicesv1.ListProjectionEventsRequest{ClaimId: "claim_graph_1"})
	if err != nil {
		t.Fatalf("list projection events: %v", err)
	}
	if !audit.GetChainValid() || len(audit.GetEvents()) != 1 || audit.GetEvents()[0].GetEventHash() != projected.GetProjectionEvent().GetEventHash() {
		t.Fatalf("unexpected projection audit response: %+v", audit)
	}

	explained, err := server.ExplainClaim(ctx, &servicesv1.ExplainClaimRequest{ClaimId: "claim_graph_1"})
	if err != nil {
		t.Fatalf("explain claim: %v", err)
	}
	if explained.GetClaim().GetId() != "claim_graph_1" || explained.GetTrustState() != "candidate" {
		t.Fatalf("unexpected explanation: claim=%v trust=%q", explained.GetClaim().GetId(), explained.GetTrustState())
	}

	closure, err := server.VerifyClosure(ctx, &servicesv1.VerifyClosureRequest{SubjectNode: "artifact:app", Purpose: "beta-review"})
	if err != nil {
		t.Fatalf("verify closure: %v", err)
	}
	if closure.GetVerdict() || closure.GetTrustState() != "incomplete" || closure.GetEdgesFailed() != 1 {
		t.Fatalf("closure must fail closed: verdict=%v trust=%q failed=%d diagnostics=%v", closure.GetVerdict(), closure.GetTrustState(), closure.GetEdgesFailed(), closure.GetDiagnostics())
	}

	defaultIDProjection, err := server.ProjectEdge(ctx, &servicesv1.ProjectEdgeRequest{ClaimId: "claim_graph_1"})
	if err != nil {
		t.Fatalf("project edge with derived ID: %v", err)
	}
	if defaultIDProjection.GetEdge().GetId() != defaultProjectionEdgeID("claim_graph_1", "artifact:app", "component:lib", "depends_on") {
		t.Fatalf("derived edge ID is not content-stable: %q", defaultIDProjection.GetEdge().GetId())
	}
	if _, err := server.ProjectEdge(ctx, &servicesv1.ProjectEdgeRequest{ClaimId: "claim_graph_1"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("replaying a derived projection returned %v, want AlreadyExists", err)
	}
}

func TestGraphProjectionAuditChainGate(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new graph store: %v", err)
	}
	defer store.Close()

	from := &Node{ID: "node:chain:from", Kind: "artifact", URN: "urn:chain:from", CreatedAt: time.Now().UTC()}
	to := &Node{ID: "node:chain:to", Kind: "component", URN: "urn:chain:to", CreatedAt: time.Now().UTC()}
	edge := &Edge{
		ID: "edge_chain_1", FromNode: from.ID, ToNode: to.ID, Relation: "depends_on",
		ClaimID: "claim_chain_1", EvidenceDigest: "sha256:chain", TrustState: "candidate",
		ValidFrom: time.Now().UTC(), Weight: 1,
	}
	if _, err := store.CreateProjection(ctx, from, to, edge); err != nil {
		t.Fatalf("create projection: %v", err)
	}
	server := NewGraphServer(store)
	valid, err := server.ListProjectionEvents(ctx, &servicesv1.ListProjectionEventsRequest{EdgeId: edge.ID})
	if err != nil || !valid.GetChainValid() || len(valid.GetEvents()) != 1 {
		t.Fatalf("valid projection audit response: response=%+v err=%v", valid, err)
	}

	sqliteStore := store.(*SQLiteStore)
	if _, err := sqliteStore.db.Exec(`DROP TRIGGER graph_projection_events_no_update`); err != nil {
		t.Fatalf("disable test-only update guard: %v", err)
	}
	if _, err := sqliteStore.db.Exec(`UPDATE graph_projection_events SET event_hash = 'sha256:tampered' WHERE event_id = ?`, valid.GetEvents()[0].GetEventId()); err != nil {
		t.Fatalf("tamper projection audit event: %v", err)
	}
	_, err = server.ListProjectionEvents(ctx, &servicesv1.ListProjectionEventsRequest{EdgeId: edge.ID})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("tampered audit chain error = %v, want FailedPrecondition", err)
	}
}
