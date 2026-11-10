package frameworks_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := frameworks.NewRegistry()

	fw := frameworks.New(frameworks.Config{
		ID:           "test-fw",
		Aliases:      []string{"alias-1", "alias-2"},
		Title:        "Test Framework",
		Version:      "1.0",
		Jurisdiction: "intl",
		Authority:    "Standards Org",
		ControlCount: 10,
		CatalogFn: func() ([]byte, error) {
			return []byte(`{"catalog":{}}`), nil
		},
	})

	if err := reg.Register(fw); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify duplicate ID
	if err := reg.Register(fw); err == nil {
		t.Fatalf("expected error on duplicate register, got nil")
	}

	// Retrieve by ID
	got, ok := reg.Get("test-fw")
	if !ok || got.ID() != "test-fw" {
		t.Fatalf("expected test-fw by ID, got %v", got)
	}

	// Retrieve by alias
	gotAlias, ok := reg.Get("alias-1")
	if !ok || gotAlias.ID() != "test-fw" {
		t.Fatalf("expected test-fw by alias-1, got %v", gotAlias)
	}

	// Check Has
	if !reg.Has("alias-2") {
		t.Fatalf("expected Has(alias-2) to be true")
	}
	if reg.Has("non-existent") {
		t.Fatalf("expected Has(non-existent) to be false")
	}
}

func TestRegistry_DefaultRegistryPopulation(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	list := reg.List()

	if len(list) == 0 {
		t.Fatalf("expected default registry to contain registered frameworks, got 0")
	}

	canadianFws := reg.FindByJurisdiction("ca")
	if len(canadianFws) < 5 {
		t.Fatalf("expected at least 5 Canadian frameworks in default registry, got %d", len(canadianFws))
	}
}

func TestRegistry_SchemaConformance(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	for _, f := range reg.List() {
		f := f
		t.Run(f.ID()+"/Catalog", func(t *testing.T) {
			catData, err := f.CatalogJSON()
			if err != nil {
				t.Fatalf("CatalogJSON() failed: %v", err)
			}
			if len(catData) == 0 {
				t.Fatalf("CatalogJSON() returned empty data")
			}
			if err := frameworks.ValidateSchemaCached(catData, schemavalidate.KindCatalog); err != nil {
				t.Fatalf("catalog for %s failed OSCAL 1.2.3 schema validation: %v", f.ID(), err)
			}
		})

		if f.HasProfile() {
			t.Run(f.ID()+"/Profile", func(t *testing.T) {
				profData, err := f.ProfileJSON()
				if err != nil {
					t.Fatalf("ProfileJSON() failed: %v", err)
				}
				if len(profData) == 0 {
					t.Fatalf("ProfileJSON() returned empty data")
				}
				if err := frameworks.ValidateSchemaCached(profData, schemavalidate.KindProfile); err != nil {
					t.Fatalf("profile for %s failed OSCAL 1.2.3 schema validation: %v", f.ID(), err)
				}
			})
		}
	}
}

func TestRegistry_Export(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	tmpDir := t.TempDir()

	// Export single
	if err := reg.Export(tmpDir, "cccs-itsp-10-033"); err != nil {
		t.Fatalf("Export(cccs-itsp-10-033) failed: %v", err)
	}

	catFile := filepath.Join(tmpDir, "cccs-itsp-10-033", "catalog.json")
	if _, err := os.Stat(catFile); err != nil {
		t.Fatalf("missing exported catalog: %s", catFile)
	}
	if err := dbutil.CheckDigestSidecar(catFile); err != nil {
		t.Fatalf("digest verification failed for catalog: %v", err)
	}

	profFile := filepath.Join(tmpDir, "cccs-itsp-10-033", "profile.json")
	if _, err := os.Stat(profFile); err != nil {
		t.Fatalf("missing exported profile: %s", profFile)
	}
	if err := dbutil.CheckDigestSidecar(profFile); err != nil {
		t.Fatalf("digest verification failed for profile: %v", err)
	}
}

func TestRegistry_SeedKG(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-kg.db")

	store, err := kg.NewSQLiteStore(dbPath, dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	f, ok := frameworks.DefaultRegistry().Get("cybersecure-canada")
	if !ok {
		t.Fatalf("cybersecure-canada framework not found")
	}

	if err := frameworks.SeedFramework(ctx, store, f); err != nil {
		t.Fatalf("SeedFramework failed: %v", err)
	}

	fws, err := store.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	if err != nil {
		t.Fatalf("list frameworks: %v", err)
	}
	if len(fws) != 1 {
		t.Fatalf("expected 1 framework entity, got %d", len(fws))
	}

	reqs, err := store.ListEntities(ctx, "reg:Requirement", kg.EntityStatusActive)
	if err != nil {
		t.Fatalf("list requirements: %v", err)
	}
	if len(reqs) == 0 {
		t.Fatalf("expected requirements entities, got 0")
	}

	// Also seed cccs-itsp-10-033 (which has base controls, enhancements, and ODPs) and verify kg_graph_nodes & kg_graph_edges
	f33, ok := frameworks.DefaultRegistry().Get("cccs-itsp-10-033")
	if !ok {
		t.Fatalf("cccs-itsp-10-033 framework not found")
	}
	if err := frameworks.SeedFramework(ctx, store, f33); err != nil {
		t.Fatalf("SeedFramework(cccs-itsp-10-033) failed: %v", err)
	}

	db := store.(*kg.SQLiteStore).DB()
	var labelsJSON string
	err = db.QueryRowContext(ctx, `SELECT labels_json FROM kg_graph_nodes WHERE id = ?`, "urn:xoscal:requirement:cccs-itsp-10-033:ac-2").Scan(&labelsJSON)
	if err != nil {
		t.Fatalf("query ac-2 graph node: %v", err)
	}
	if labelsJSON == "" || labelsJSON == `{"framework":"cccs-itsp-10-033"}` {
		t.Errorf("expected full labels_json on parent control ac-2 (not overwritten by enhancement stub), got %s", labelsJSON)
	}

	for _, rel := range []string{"BELONGS_TO_FRAMEWORK", "IN_CONTROL_FAMILY", "ENHANCES_CONTROL", "DEFINES_ODP"} {
		var edgeCount int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM kg_graph_edges WHERE relation = ?`, rel).Scan(&edgeCount); err != nil {
			t.Fatalf("count %s edges: %v", rel, err)
		}
		if edgeCount < 100 {
			t.Errorf("expected >= 100 %s edges preserved in kg_graph_edges without ON DELETE CASCADE wipeout, got %d", rel, edgeCount)
		}
	}
}
