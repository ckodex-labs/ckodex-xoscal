package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportFrameworksOfflineAuthoritative(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	outDir := filepath.Join(dir, "out")
	dsn := filepath.Join(dir, "export.db")

	manifestYAML := `source_repo: github.com/intuitem/ciso-assistant-community/backend/library/libraries
frameworks:
  - ref_id: cybersecure-canada
    name: CyberSecure Canada
    jurisdiction: ca
    filename: cybersecure-canada.json
  - ref_id: cccs-medium-cloud-pbmm
    name: CCCS Medium Cloud PBMM
    jurisdiction: ca
    filename: cccs-medium-cloud-pbmm.json
`
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := run(manifestPath, outDir, dsn, false); err != nil {
		t.Fatalf("offline export failed: %v", err)
	}

	for _, rel := range []string{
		"cybersecure-canada/catalog.json",
		"cybersecure-canada/catalog.json.sha256",
		"cccs-medium-cloud-pbmm/catalog.json",
		"cccs-medium-cloud-pbmm/catalog.json.sha256",
		"cccs-medium-cloud-pbmm/profile.json",
		"cccs-medium-cloud-pbmm/profile.json.sha256",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing exported file %s: %v", rel, err)
		}
	}

	if err := run(filepath.Join(dir, "missing.yaml"), outDir, dsn, false); err == nil {
		t.Fatal("expected error for missing manifest")
	}
}
