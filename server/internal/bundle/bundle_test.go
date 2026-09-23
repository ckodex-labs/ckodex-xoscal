package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/scaffold"
)

func TestAuditBundle_CreateAndVerify(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Scaffold an artifact
	_, _, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	artPath := filepath.Join(tmpDir, "component-definition.json")

	// 2. Create evidence
	evidenceDir := filepath.Join(tmpDir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0750); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}
	evFile := filepath.Join(evidenceDir, "evidence-ac-2.txt")
	if err := os.WriteFile(evFile, []byte("mfa configuration evidence dump"), 0600); err != nil {
		t.Fatalf("write evidence: %v", err)
	}

	// 3. Package bundle
	bundlePath := filepath.Join(tmpDir, "audit-bundle.tar.gz")
	if err := CreateAuditBundle([]string{artPath}, evidenceDir, bundlePath); err != nil {
		t.Fatalf("CreateAuditBundle failed: %v", err)
	}

	// 4. Verify bundle
	manifest, err := VerifyBundle(bundlePath)
	if err != nil {
		t.Fatalf("VerifyBundle failed: %v", err)
	}

	if len(manifest.Artifacts) != 1 {
		t.Errorf("expected 1 artifact in manifest, got %d", len(manifest.Artifacts))
	}
	if len(manifest.Evidence) != 1 {
		t.Errorf("expected 1 evidence file in manifest, got %d", len(manifest.Evidence))
	}
	if manifest.BundleVersion != "1.0.0" {
		t.Errorf("unexpected bundle version: %s", manifest.BundleVersion)
	}
}
