package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/tabular"
)

func TestCompareArtifacts_Detection(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Scaffold base artifact
	_, baseJSON, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold base: %v", err)
	}

	basePath := filepath.Join(tmpDir, "base.json")
	// Save as base.json
	headPath := filepath.Join(tmpDir, "head.json")

	// 2. Modify one control via tabular bridge
	csvBytes, err := tabular.ExportToCSV(baseJSON)
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}

	// Change ac-2 prose
	modifiedCSV := strings.Replace(string(csvBytes), "Account management implemented via service authentication and credential checks.", "FIPS 140-3 validated enterprise MFA credentials.", 1)

	updatedHeadJSON, err := tabular.ImportFromCSV(baseJSON, []byte(modifiedCSV))
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}

	if err := osWrite(basePath, baseJSON); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := osWrite(headPath, updatedHeadJSON); err != nil {
		t.Fatalf("write head: %v", err)
	}

	// 3. Compare artifacts
	complianceDiff, err := CompareArtifacts(basePath, headPath)
	if err != nil {
		t.Fatalf("CompareArtifacts failed: %v", err)
	}

	if complianceDiff.ModifiedCount != 1 {
		t.Errorf("expected 1 modified control, got %d", complianceDiff.ModifiedCount)
	}
	if len(complianceDiff.Deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(complianceDiff.Deltas))
	}
	if complianceDiff.Deltas[0].ControlID != "ac-2" {
		t.Errorf("expected modified control to be ac-2, got %s", complianceDiff.Deltas[0].ControlID)
	}

	// 4. Test Markdown Formatting
	md := complianceDiff.FormatMarkdown()
	if !strings.Contains(md, "xOSCAL Compliance Impact Report") {
		t.Errorf("markdown report missing header")
	}
	if !strings.Contains(md, "ac-2") {
		t.Errorf("markdown report missing ac-2 delta")
	}
}

func osWrite(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
