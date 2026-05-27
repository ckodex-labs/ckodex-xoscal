package graph

import (
	"context"
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

	nodes, err := store.ListNodes(ctx, "artifact", "", time.Time{}, 10)
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 artifact node, got %d", len(nodes))
	}
}
