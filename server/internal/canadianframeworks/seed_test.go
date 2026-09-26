package canadianframeworks

import (
	"context"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
)

func TestSeedKG(t *testing.T) {
	ctx := context.Background()
	store, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("create in-memory kg store: %v", err)
	}
	defer store.Close()

	if err := SeedKG(ctx, store); err != nil {
		t.Fatalf("SeedKG failed: %v", err)
	}

	// Verify framework entities
	fws, err := store.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	if err != nil {
		t.Fatalf("list framework entities: %v", err)
	}
	if len(fws) != 4 {
		t.Fatalf("expected 4 framework entities, got %d", len(fws))
	}

	// Verify requirement entities
	reqs, err := store.ListEntities(ctx, "reg:Requirement", kg.EntityStatusActive)
	if err != nil {
		t.Fatalf("list requirement entities: %v", err)
	}
	if len(reqs) < 300 {
		t.Errorf("expected > 300 requirement entities, got %d", len(reqs))
	}

	// Test idempotency: running SeedKG again should succeed without duplicating entities
	if err := SeedKG(ctx, store); err != nil {
		t.Fatalf("second SeedKG failed: %v", err)
	}
	fwsAfter, _ := store.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	if len(fwsAfter) != 4 {
		t.Errorf("expected 4 frameworks after second seed, got %d", len(fwsAfter))
	}

	// Test snapshot and catalog generation
	if _, err := store.CreateSnapshot(ctx, "test-snap"); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	gen := oscal.NewGenerator(store)
	cat, err := gen.GenerateCatalog(ctx, "test-snap", "cccs-itsp-10-171")
	if err != nil {
		t.Fatalf("generate catalog from snapshot: %v", err)
	}
	t.Logf("cat: controls=%d, groups=%d", len(cat.Controls), len(cat.Groups))
	if cat == nil || (len(cat.Controls) == 0 && len(cat.Groups) == 0) {
		t.Errorf("generated catalog missing controls and groups")
	}
}

func TestListFrameworksAndIsCanadian(t *testing.T) {
	fws := ListFrameworks()
	if len(fws) != 4 {
		t.Fatalf("expected 4 frameworks, got %d", len(fws))
	}

	for _, fw := range fws {
		if !IsCanadian(fw.RefID) {
			t.Errorf("expected %s to be recognized as Canadian", fw.RefID)
		}
		if fw.Locale != "ca" {
			t.Errorf("expected %s locale to be 'ca', got %s", fw.RefID, fw.Locale)
		}
	}

	if !IsCanadian("itsp.10.171") {
		t.Errorf("expected alias itsp.10.171 to be recognized as Canadian")
	}
	if IsCanadian("nist-sp-800-53") {
		t.Errorf("nist-sp-800-53 should not be recognized as Canadian")
	}
}
