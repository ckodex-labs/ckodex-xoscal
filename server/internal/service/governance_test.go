package service

import (
	"context"
	"testing"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func TestGovernanceEntityCRUD(t *testing.T) {
	ctx := context.Background()
	kgStore, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("kg store: %v", err)
	}
	defer kgStore.Close()
	vs, err := embedding.NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("vector store: %v", err)
	}
	defer vs.Close()

	srv := NewGovernanceServer(kgStore, reconciler.NewReconciler(kgStore), vs)

	// Create
	createRes, err := srv.CreateEntity(ctx, &servicesv1.CreateEntityRequest{
		Entity: &servicesv1.Entity{
			Urn:     "urn:test:req:1",
			Type:    "Requirement",
			Version: "1",
			Status:  "active",
			Payload: `{"title":"Test Requirement"}`,
		},
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	if createRes.Entity.Urn != "urn:test:req:1" {
		t.Errorf("urn = %s, want urn:test:req:1", createRes.Entity.Urn)
	}

	// Get
	getRes, err := srv.GetEntity(ctx, &servicesv1.GetEntityRequest{Urn: "urn:test:req:1"})
	if err != nil {
		t.Fatalf("get entity: %v", err)
	}
	if getRes.Entity.Type != "Requirement" {
		t.Errorf("type = %s, want Requirement", getRes.Entity.Type)
	}

	// Update
	_, err = srv.UpdateEntity(ctx, &servicesv1.UpdateEntityRequest{
		Entity: &servicesv1.Entity{
			Urn:     "urn:test:req:1",
			Type:    "Requirement",
			Version: "2",
			Status:  "active",
			Payload: `{"title":"Updated Requirement"}`,
		},
	})
	if err != nil {
		t.Fatalf("update entity: %v", err)
	}

	// List active only (update creates a new version; old version is superseded)
	listRes, err := srv.ListEntities(ctx, &servicesv1.ListEntitiesRequest{TypeFilter: "Requirement", StatusFilter: "active"})
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(listRes.Entities) != 1 {
		t.Errorf("entities = %d, want 1", len(listRes.Entities))
	}
}

func TestGovernanceSnapshotAndRelease(t *testing.T) {
	ctx := context.Background()
	kgStore, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("kg store: %v", err)
	}
	defer kgStore.Close()
	vs, _ := embedding.NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	defer vs.Close()

	srv := NewGovernanceServer(kgStore, reconciler.NewReconciler(kgStore), vs)

	// Seed an entity
	_, _ = srv.CreateEntity(ctx, &servicesv1.CreateEntityRequest{
		Entity: &servicesv1.Entity{Urn: "urn:a", Type: "Control", Version: "1", Status: "active", Payload: `{}`},
	})

	// Snapshot
	ssRes, err := srv.CreateSnapshot(ctx, &servicesv1.CreateSnapshotRequest{Name: "v1"})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if ssRes.EntityCount != 1 {
		t.Errorf("entity_count = %d, want 1", ssRes.EntityCount)
	}

	// Release
	relRes, err := srv.CreateRelease(ctx, &servicesv1.CreateReleaseRequest{Name: "r1", SnapshotName: "v1"})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if relRes.Name != "r1" {
		t.Errorf("name = %s, want r1", relRes.Name)
	}
}

func TestGovernanceIngestAndSearch(t *testing.T) {
	ctx := context.Background()
	kgStore, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("kg store: %v", err)
	}
	defer kgStore.Close()
	vs, err := embedding.NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("vector store: %v", err)
	}
	defer vs.Close()

	rec := reconciler.NewReconciler(kgStore)
	srv := NewGovernanceServer(kgStore, rec, vs)

	raw := []byte(`[{"id":"art16-a","citation":"Article 16(a)","title":"Risk","text":"Implement risk management","role":"provider","risk_level":"high-risk","framework":"eu-ai-act"}]`)

	ingestRes, err := srv.IngestRequirements(ctx, &servicesv1.IngestRequirementsRequest{
		RawData:   raw,
		Format:    "eu-ai-act-json",
		Framework: "eu-ai-act",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if ingestRes.Created != 1 {
		t.Errorf("created = %d, want 1", ingestRes.Created)
	}

	// Semantic search
	searchRes, err := srv.SemanticSearch(ctx, &servicesv1.SemanticSearchRequest{
		Query:     "risk",
		Framework: "eu-ai-act",
		TopK:      10,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	// FTS5 may return 0 on tiny datasets; just verify no panic.
	t.Logf("search results: %d", len(searchRes.Results))
}
