package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestUUIDFromURN_Determinism(t *testing.T) {
	urn1 := "urn:xoscal:comp:auth-service"
	urn2 := "urn:xoscal:comp:auth-service"
	urn3 := "urn:xoscal:comp:database"

	u1 := UUIDFromURN(urn1)
	u2 := UUIDFromURN(urn2)
	u3 := UUIDFromURN(urn3)

	if u1.Value != u2.Value {
		t.Fatalf("expected identical UUIDs for same URN, got %s and %s", u1.Value, u2.Value)
	}
	if u1.Value == u3.Value {
		t.Fatalf("expected distinct UUIDs for different URNs, got %s", u1.Value)
	}
}

func TestScanRepository_Local(t *testing.T) {
	// Scan current repo root
	root := "../../.."
	scan, err := ScanRepository(root)
	if err != nil {
		t.Fatalf("ScanRepository failed: %v", err)
	}

	if len(scan.Languages) == 0 {
		t.Errorf("expected at least one language detected")
	}
	if !scan.HasDocker {
		t.Errorf("expected Docker to be detected in repo")
	}
	if !scan.HasK8s {
		t.Errorf("expected K8s to be detected in repo")
	}
	if len(scan.Components) == 0 {
		t.Errorf("expected at least one component candidate")
	}
}

func TestScaffoldWorkspace_TempDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xoscal-scaffold-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create fake go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/demo-service\n\ngo 1.24\n"), 0600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	scan, jsonBytes, err := ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("ScaffoldWorkspace failed: %v", err)
	}

	if scan.Name != "demo-service" {
		t.Errorf("expected scan name 'demo-service', got '%s'", scan.Name)
	}

	// Verify schema directly on written file
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}

	if err := v.Validate(jsonBytes, schemavalidate.KindComponentDefinition); err != nil {
		t.Fatalf("generated component definition failed schema validation: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(tmpDir, ".xoscal", "xoscal.yaml")); err != nil {
		t.Errorf("xoscal.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "component-definition.json")); err != nil {
		t.Errorf("component-definition.json missing: %v", err)
	}
}

func TestScaffoldWorkspace_CanadianFrameworks(t *testing.T) {
	frameworks := []string{"cccs-itsg-33", "cccs-medium-cloud-pbmm", "cybersecure-canada", "cccs-itsp-10-171"}

	for _, fw := range frameworks {
		t.Run(fw, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "xoscal-scaffold-canadian-*")
			if err != nil {
				t.Fatalf("create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/canadian-service\n\ngo 1.24\n"), 0600); err != nil {
				t.Fatalf("write go.mod: %v", err)
			}

			_, jsonBytes, err := ScaffoldWorkspace(tmpDir, fw)
			if err != nil {
				t.Fatalf("ScaffoldWorkspace with framework %s failed: %v", fw, err)
			}

			v, err := schemavalidate.NewValidator()
			if err != nil {
				t.Fatalf("init validator: %v", err)
			}

			if err := v.Validate(jsonBytes, schemavalidate.KindComponentDefinition); err != nil {
				t.Fatalf("generated component definition for framework %s failed schema validation: %v", fw, err)
			}
		})
	}
}

