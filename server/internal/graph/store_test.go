package graph

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestStore_NodeEdgeLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	// Create nodes
	n1 := &Node{ID: "node:artifact:sha256:abc", Kind: "artifact", URN: "urn:test:artifact:a", CreatedAt: time.Now().UTC()}
	n2 := &Node{ID: "node:component:pkg:maven:test:1.0", Kind: "component", URN: "urn:test:component:b", CreatedAt: time.Now().UTC()}
	if err := store.CreateNode(ctx, n1); err != nil {
		t.Fatalf("create node 1: %v", err)
	}
	if err := store.CreateNode(ctx, n2); err != nil {
		t.Fatalf("create node 2: %v", err)
	}

	// Create edge
	e := &Edge{
		ID:             "edge_01HXTEST",
		FromNode:       n1.ID,
		ToNode:         n2.ID,
		Relation:       "depends_on",
		ClaimID:        "claim_01HXTEST",
		EvidenceDigest: "sha256:abc",
		TrustState:     "candidate",
		Weight:         1.0,
		ValidFrom:      time.Now().UTC(),
	}
	if err := store.CreateEdge(ctx, e); err != nil {
		t.Fatalf("create edge: %v", err)
	}

	// Get edge
	got, err := store.GetEdge(ctx, e.ID)
	if err != nil {
		t.Fatalf("get edge: %v", err)
	}
	if got.ID != e.ID {
		t.Fatalf("expected edge id %s, got %s", e.ID, got.ID)
	}

	// List edges from node
	edges, err := store.ListEdgesFrom(ctx, n1.ID, nil, "", 10)
	if err != nil {
		t.Fatalf("list edges from: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge from n1, got %d", len(edges))
	}

	// List edges to node
	edges, err = store.ListEdgesTo(ctx, n2.ID, nil, "", 10)
	if err != nil {
		t.Fatalf("list edges to: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge to n2, got %d", len(edges))
	}
	verified := *e
	verified.ID = "edge_01HXVERIFIED"
	verified.TrustState = "verified"
	if err := store.CreateEdge(ctx, &verified); err != nil {
		t.Fatalf("create verified edge: %v", err)
	}
	verifiedEdges, err := store.ListEdgesFrom(ctx, n1.ID, nil, "verified", 10)
	if err != nil {
		t.Fatalf("list verified edges from: %v", err)
	}
	if len(verifiedEdges) != 1 || verifiedEdges[0].ID != verified.ID {
		t.Fatalf("expected only verified edge, got %#v", verifiedEdges)
	}

	rollbackFrom := &Node{ID: "node:rollback:from", Kind: "artifact", URN: "urn:test:rollback:from", CreatedAt: time.Now().UTC()}
	rollbackTo := &Node{ID: "node:rollback:to", Kind: "component", URN: "urn:test:rollback:to", CreatedAt: time.Now().UTC()}
	rollbackEdge := *e
	rollbackEdge.FromNode = rollbackFrom.ID
	rollbackEdge.ToNode = rollbackTo.ID
	rollbackEdge.ID = e.ID // force the edge insert to fail after node inserts
	if _, err := store.CreateProjection(ctx, rollbackFrom, rollbackTo, &rollbackEdge); err == nil {
		t.Fatal("expected duplicate edge to abort projection")
	}
	if _, err := store.GetNode(ctx, rollbackFrom.ID); err != sql.ErrNoRows {
		t.Fatalf("rollback left source node behind: %v", err)
	}
	if _, err := store.GetNode(ctx, rollbackTo.ID); err != sql.ErrNoRows {
		t.Fatalf("rollback left target node behind: %v", err)
	}

	// Delete edge
	if err := store.DeleteEdge(ctx, e.ID); err != nil {
		t.Fatalf("delete edge: %v", err)
	}
	_, err = store.GetEdge(ctx, e.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestStore_ListNodes(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	n1 := &Node{ID: "node:artifact:1", Kind: "artifact", URN: "urn:a", CreatedAt: time.Now().UTC()}
	n2 := &Node{ID: "node:component:2", Kind: "component", URN: "urn:b", CreatedAt: time.Now().UTC()}
	store.CreateNode(ctx, n1)
	store.CreateNode(ctx, n2)

	nodes, _, err := store.ListNodes(ctx, "artifact", "", time.Time{}, 10, "")
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 artifact node, got %d", len(nodes))
	}
}

func TestStore_ProjectionAuditIsAppendOnly(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	from := &Node{ID: "node:audit:from", Kind: "artifact", URN: "urn:audit:from", CreatedAt: time.Now().UTC()}
	to := &Node{ID: "node:audit:to", Kind: "component", URN: "urn:audit:to", CreatedAt: time.Now().UTC()}
	edge := &Edge{
		ID: "edge_audit_1", FromNode: from.ID, ToNode: to.ID, Relation: "depends_on",
		ClaimID: "claim_audit_1", EvidenceDigest: "sha256:audit", TrustState: "candidate",
		ValidFrom: time.Now().UTC(), Weight: 1,
	}
	event, err := store.CreateProjection(ctx, from, to, edge)
	if err != nil {
		t.Fatalf("create projection: %v", err)
	}
	if event.Sequence != 1 || event.EventHash == "" || event.PreviousHash != "" {
		t.Fatalf("unexpected first projection event: %+v", event)
	}
	if err := store.VerifyProjectionChain(ctx); err != nil {
		t.Fatalf("verify projection chain: %v", err)
	}
	events, err := store.ListProjectionEvents(ctx, edge.ClaimID, "")
	if err != nil || len(events) != 1 || events[0].EventHash != event.EventHash {
		t.Fatalf("unexpected projection event history: events=%+v err=%v", events, err)
	}

	sqliteStore := store.(*SQLiteStore)
	if _, err := sqliteStore.db.Exec(`UPDATE graph_projection_events SET trust_state = 'verified' WHERE event_id = ?`, event.EventID); err == nil {
		t.Fatal("expected projection audit update to be rejected")
	}
	if _, err := sqliteStore.db.Exec(`DELETE FROM graph_projection_events WHERE event_id = ?`, event.EventID); err == nil {
		t.Fatal("expected projection audit delete to be rejected")
	}
	if err := store.VerifyProjectionChain(ctx); err != nil {
		t.Fatalf("append-only rejection should preserve chain: %v", err)
	}
}
