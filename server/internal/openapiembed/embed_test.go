package openapiembed_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/openapiembed"
)

func TestEmbeddedSpecsParityWithGeneratedProtoOpenAPI(t *testing.T) {
	protoPattern := filepath.Join("..", "..", "..", "proto", "oscal", "gen", "openapi", "services", "v1", "*.openapi.yaml")
	matches, err := filepath.Glob(protoPattern)
	if err != nil {
		t.Fatalf("glob %s: %v", protoPattern, err)
	}
	if len(matches) == 0 {
		t.Fatalf("no generated OpenAPI specs matched %s", protoPattern)
	}
	sort.Strings(matches)

	embeddedList := openapiembed.ListSpecs()
	if len(embeddedList) != len(matches) {
		t.Fatalf("ListSpecs() count = %d (%v), want %d", len(embeddedList), embeddedList, len(matches))
	}

	embeddedSet := make(map[string]bool, len(embeddedList))
	for _, n := range embeddedList {
		embeddedSet[n] = true
	}

	for _, diskPath := range matches {
		baseName := filepath.Base(diskPath)
		if !embeddedSet[baseName] {
			t.Errorf("ListSpecs() missing %q", baseName)
		}

		diskBytes, err := os.ReadFile(diskPath)
		if err != nil {
			t.Fatalf("read %s: %v", diskPath, err)
		}
		embedBytes, err := openapiembed.ReadSpec(baseName)
		if err != nil {
			t.Fatalf("ReadSpec(%q): %v", baseName, err)
		}

		diskSum := sha256.Sum256(diskBytes)
		embedSum := sha256.Sum256(embedBytes)
		if diskSum != embedSum {
			t.Errorf("SHA-256 mismatch for %s: proto=%x embedded=%x", baseName, diskSum, embedSum)
		}
	}

	// Reject path traversal and unknown specs
	for _, bad := range []string{"", "../embed.go", "../../main.go", "nonexistent.openapi.yaml"} {
		if _, err := openapiembed.ReadSpec(bad); err == nil {
			t.Errorf("ReadSpec(%q) succeeded, expected error", bad)
		}
	}
}
