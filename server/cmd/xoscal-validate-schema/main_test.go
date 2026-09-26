package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestValidateSchema_ValidCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	catPath := filepath.Join(tmpDir, "catalog.json")

	cat := canadianframeworks.BuildCyberSecureCanadaCatalog()
	data, err := oscal.ExportCatalogJSON(cat)
	if err != nil {
		t.Fatalf("export catalog: %v", err)
	}

	if err := os.WriteFile(catPath, data, 0600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	kind, err := resolveKind("", data)
	if err != nil {
		t.Fatalf("resolveKind: %v", err)
	}

	if err := v.Validate(data, kind); err != nil {
		t.Errorf("expected valid catalog, got error: %v", err)
	}
}

func TestValidateSchema_InvalidJSON(t *testing.T) {
	invalidData := []byte(`{"catalog": {"invalid": true}}`)

	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	kind, err := resolveKind("", invalidData)
	if err != nil {
		t.Fatalf("resolveKind: %v", err)
	}

	if err := v.Validate(invalidData, kind); err == nil {
		t.Errorf("expected validation failure for incomplete catalog, got nil")
	}
}
