package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunProvenanceManifest(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "artifacts.json")
	outPath := filepath.Join(dir, "provenance.json")

	input := `[
  {
    "name": "xoscal_0.4.1_linux_amd64.tar.gz",
    "digest": "sha256:e65a98a0d21b83d989ddb3de039adc9ed0be63540c0a2016a88a0c31a4c0e700"
  }
]`
	if err := os.WriteFile(inPath, []byte(input), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if err := run(inPath, outPath); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("invalid provenance JSON: %v", err)
	}
	if len(parsed) != 1 || parsed[0]["name"] != "xoscal_0.4.1_linux_amd64.tar.gz" {
		t.Fatalf("unexpected provenance content: %v", parsed)
	}

	if err := run(filepath.Join(dir, "missing.json"), outPath); err == nil {
		t.Fatal("expected error for missing input file")
	}
}
