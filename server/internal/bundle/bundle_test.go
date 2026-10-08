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

func TestAuditBundle_OutputWithinEvidenceDirectory(t *testing.T) {
	for _, previous := range []bool{false, true} {
		evidenceDir := t.TempDir()
		mustBundleWrite(t, filepath.Join(evidenceDir, "report.log"), []byte("ordinary evidence"))
		output := filepath.Join(evidenceDir, "bundle.tar.gz")
		if previous {
			mustBundleWrite(t, output, []byte("previous output"))
		}
		if err := CreateAuditBundle(nil, evidenceDir, output); err != nil {
			t.Fatal(err)
		}
		manifest, err := VerifyBundle(output)
		if err != nil {
			t.Fatal(err)
		}
		if len(manifest.Evidence) != 1 || manifest.Evidence[0].Path != "evidence/report.log" {
			t.Fatalf("output file became automatic evidence: %+v", manifest.Evidence)
		}
	}
}

func TestAuditBundle_OutputSymlinkIsReplacedWithoutReadingTarget(t *testing.T) {
	evidenceDir := t.TempDir()
	mustBundleWrite(t, filepath.Join(evidenceDir, "report.log"), []byte("ordinary evidence"))
	previous := filepath.Join(t.TempDir(), "previous-output")
	mustBundleWrite(t, previous, []byte("previous output target"))
	output := filepath.Join(evidenceDir, "bundle.tar.gz")
	if err := os.Symlink(previous, output); err != nil {
		t.Fatal(err)
	}
	if err := CreateAuditBundle(nil, evidenceDir, output); err != nil {
		t.Fatal(err)
	}
	manifest, err := VerifyBundle(output)
	if err != nil || len(manifest.Evidence) != 1 {
		t.Fatalf("output overlap: %+v %v", manifest, err)
	}
	info, err := os.Lstat(output)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("output symlink not replaced: %v", err)
	}
	data, err := os.ReadFile(previous)
	if err != nil || string(data) != "previous output target" {
		t.Fatalf("previous target changed: %q %v", data, err)
	}
}
