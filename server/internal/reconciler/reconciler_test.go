package reconciler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func newTestStore(t *testing.T) kg.Store {
	s, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return s
}

func TestPropose_NoConflict(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	payload, _ := json.Marshal(map[string]string{"text": "requirement A"})
	proposal := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: payload}

	conflicts, err := r.Propose(ctx, []*kg.Entity{proposal})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestPropose_DuplicateConflict(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	payload, _ := json.Marshal(map[string]string{"text": "original"})
	existing := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: payload}
	if err := store.CreateEntity(ctx, existing); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	newPayload, _ := json.Marshal(map[string]string{"text": "updated"})
	proposal := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: newPayload}

	conflicts, err := r.Propose(ctx, []*kg.Entity{proposal})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeDuplicate {
		t.Fatalf("expected duplicate conflict, got %s", conflicts[0].Type)
	}

	// Verify persisted.
	listed, err := r.ListConflicts(ctx)
	if err != nil {
		t.Fatalf("list conflicts: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 persisted conflict, got %d", len(listed))
	}
}

func TestPropose_VersionMismatch(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	payload, _ := json.Marshal(map[string]interface{}{"text": "req", "version": "1.0"})
	existing := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: payload}
	if err := store.CreateEntity(ctx, existing); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	newPayload, _ := json.Marshal(map[string]interface{}{"text": "req", "version": "2.0"})
	proposal := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: newPayload}

	conflicts, err := r.Propose(ctx, []*kg.Entity{proposal})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeVersionMismatch {
		t.Fatalf("expected version mismatch, got %s", conflicts[0].Type)
	}
}

func TestPropose_SemanticDrift(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	payload, _ := json.Marshal(map[string]interface{}{"text": "same text", "risk_level": "high", "role": "provider"})
	existing := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: payload}
	if err := store.CreateEntity(ctx, existing); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	newPayload, _ := json.Marshal(map[string]interface{}{"text": "same text", "risk_level": "low", "role": "provider"})
	proposal := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: newPayload}

	conflicts, err := r.Propose(ctx, []*kg.Entity{proposal})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeSemanticDrift {
		t.Fatalf("expected semantic drift, got %s", conflicts[0].Type)
	}
}

func TestEvaluateBatch_MappingGap(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	// Create two mappings to the same control.
	m1Payload, _ := json.Marshal(kg.Mapping{URN: "urn:test:map:1", Type: "reg:Mapping", From: "urn:req:a", To: "urn:ctrl:x"})
	m2Payload, _ := json.Marshal(kg.Mapping{URN: "urn:test:map:2", Type: "reg:Mapping", From: "urn:req:b", To: "urn:ctrl:x"})

	m1 := &kg.Entity{URN: "urn:test:map:1", Type: "reg:Mapping", Payload: m1Payload}
	m2 := &kg.Entity{URN: "urn:test:map:2", Type: "reg:Mapping", Payload: m2Payload}

	if err := store.CreateEntity(ctx, m1); err != nil {
		t.Fatalf("create m1: %v", err)
	}

	conflicts, err := r.EvaluateBatch(ctx, []*kg.Entity{m2})
	if err != nil {
		t.Fatalf("evaluate batch: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 mapping gap conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeMappingGap {
		t.Fatalf("expected mapping gap, got %s", conflicts[0].Type)
	}
}

func TestResolveConflict(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	defer store.Close()
	r := NewReconciler(store)

	payload, _ := json.Marshal(map[string]string{"text": "original"})
	existing := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: payload}
	if err := store.CreateEntity(ctx, existing); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	newPayload, _ := json.Marshal(map[string]string{"text": "updated"})
	proposal := &kg.Entity{URN: "urn:test:req:a", Type: "reg:Requirement", Payload: newPayload}
	conflicts, err := r.Propose(ctx, []*kg.Entity{proposal})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	// Resolve it.
	if err := r.ResolveConflict(ctx, conflicts[0].ID, []byte(`{"action":"merge"}`)); err != nil {
		t.Fatalf("resolve conflict: %v", err)
	}

	// List should be empty now.
	listed, err := r.ListConflicts(ctx)
	if err != nil {
		t.Fatalf("list conflicts: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected 0 unresolved conflicts after resolution, got %d", len(listed))
	}
}
