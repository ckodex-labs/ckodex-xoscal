package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestSpecRegistryPopulateVerifyAndDrift(t *testing.T) {
	var drift atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if drift.Load() {
			_, _ = fmt.Fprintf(w, `{"$schema":"drifted","asset":%q}`, r.URL.Path)
			return
		}
		_, _ = fmt.Fprintf(w, `{"$schema":"pinned","asset":%q}`, r.URL.Path)
	}))
	defer srv.Close()

	origURL := schemaURLForModel
	t.Cleanup(func() { schemaURLForModel = origURL })
	schemaURLForModel = func(version, asset string) string {
		return srv.URL + "/" + version + "/" + asset
	}

	dir := t.TempDir()
	regPath := filepath.Join(dir, "spec-registry.yaml")
	initialYAML := `oscal_version: "1.2.3"
models:
  - model: catalog
    schema_asset: oscal_catalog_schema.json
    proto_package: oscal.catalog.v1
    schema_sha256: ""
`
	if err := os.WriteFile(regPath, []byte(initialYAML), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	// Unpopulated verify is inert (exit 0).
	if code, err := run("verify", regPath, ""); err != nil || code != 0 {
		t.Fatalf("expected code=0 on unpopulated verify, got code=%d err=%v", code, err)
	}

	// Populate pins hashes.
	if code, err := run("populate", regPath, "1.2.3"); err != nil || code != 0 {
		t.Fatalf("populate failed: code=%d err=%v", code, err)
	}

	// Verify succeeds (exit 0).
	if code, err := run("verify", regPath, ""); err != nil || code != 0 {
		t.Fatalf("verify failed: code=%d err=%v", code, err)
	}

	// Simulated upstream drift returns exit code 3.
	drift.Store(true)
	if code, err := run("verify", regPath, ""); err != nil || code != 3 {
		t.Fatalf("expected drift exit code 3, got code=%d err=%v", code, err)
	}

	// Invalid mode returns exit code 1.
	if code, err := run("invalid", regPath, ""); err == nil || code != 1 {
		t.Fatalf("expected invalid mode code 1, got code=%d err=%v", code, err)
	}
}
