package fetcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFetchFromLocalCache(t *testing.T) {
	tmpDir := t.TempDir()
	f := &GitHubFetcher{
		owner:    "intuitem",
		repo:     "ciso-assistant-community",
		path:     "backend/library/libraries",
		cacheDir: tmpDir,
	}

	sampleYAML := []byte("ref_id: nist-csf-2.0\nname: NIST CSF 2.0\n")
	cacheFile := filepath.Join(tmpDir, "backend_library_libraries_nist-csf-2.0.yaml_abc1234.yaml")
	if err := os.WriteFile(cacheFile, sampleYAML, 0600); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	files, err := f.fetchFromLocalCache(parseFilter("nist-csf-2.0,cmmc-2.0"))
	if err != nil {
		t.Fatalf("fetchFromLocalCache failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 cached file, got %d", len(files))
	}
	if files[0].RefID != "nist-csf-2.0" {
		t.Errorf("expected RefID nist-csf-2.0, got %s", files[0].RefID)
	}
	if string(files[0].Content) != string(sampleYAML) {
		t.Errorf("unexpected content: %q", string(files[0].Content))
	}
}
