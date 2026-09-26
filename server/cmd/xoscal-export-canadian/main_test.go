package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func TestExportSpecificCanadianFramework(t *testing.T) {
	tmpDir := t.TempDir()

	for _, refID := range []string{"cccs-itsg-33", "cybersecure-canada", "cccs-itsp-10-171", "cccs-medium-cloud-pbmm"} {
		if err := canadianframeworks.Export(tmpDir, refID); err != nil {
			t.Fatalf("Export(%s) failed: %v", refID, err)
		}
		catPath := filepath.Join(tmpDir, refID, "catalog.json")
		if _, err := os.Stat(catPath); err != nil {
			t.Fatalf("missing catalog for %s: %v", refID, err)
		}
		if err := dbutil.CheckDigestSidecar(catPath); err != nil {
			t.Fatalf("checksum failed for %s: %v", refID, err)
		}
	}
}

func TestExportUnknownCanadianFramework(t *testing.T) {
	tmpDir := t.TempDir()
	err := canadianframeworks.Export(tmpDir, "unknown-fw")
	if err == nil {
		t.Fatal("expected error for unknown Canadian framework, got nil")
	}
}
