package frameworks_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
)

func TestVerifyArtifactParity_ITSP10033(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-itsp-10-033")
	if !ok {
		t.Fatalf("framework cccs-itsp-10-033 not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-itsp-10-033", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
		for _, d := range rep.Discrepancies {
			t.Logf("Discrepancy: [%s] Control %s: %s", d.Aspect, d.ControlID, d.Message)
		}
	}

	if !rep.SidecarMatch {
		t.Errorf("expected sidecar match to be true")
	}
	if !rep.SchemaValid {
		t.Errorf("expected OSCAL 1.2.3 schema valid, got errors: %v", rep.SchemaErrors)
	}
	if !rep.IntegrityValid {
		t.Errorf("expected semantic integrity valid, got errors: %v", rep.IntegrityErrors)
	}
	if rep.TotalExpectedControls != 1248 {
		t.Errorf("expected 1248 controls, got %d", rep.TotalExpectedControls)
	}
	if rep.MatchedControls != 1248 {
		t.Errorf("expected 1248 matched controls, got %d", rep.MatchedControls)
	}
	if rep.ParityScore < 99.9 {
		t.Errorf("expected parity score ~100%%, got %.2f%%", rep.ParityScore)
	}

	// Format outputs
	term := rep.FormatTerminal(false)
	if !strings.Contains(term, "xOSCAL FRAMEWORK VERBATIM PARITY REPORT") {
		t.Errorf("FormatTerminal missing header: %s", term)
	}

	md := rep.FormatMarkdown()
	if !strings.Contains(md, "xOSCAL Framework Verbatim Parity Report") {
		t.Errorf("FormatMarkdown missing header: %s", md)
	}
}

func TestVerifyArtifactParity_CyberSecureCanada(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cybersecure-canada", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
		for _, d := range rep.Discrepancies {
			t.Logf("Discrepancy: [%s] Control %s: %s", d.Aspect, d.ControlID, d.Message)
		}
	}
	if rep.TotalExpectedControls != 46 {
		t.Errorf("expected 46 controls, got %d", rep.TotalExpectedControls)
	}
}

func TestVerifyArtifactParity_CCCS_ITSG33(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-itsg-33")
	if !ok {
		t.Fatalf("framework cccs-itsg-33 not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-itsg-33", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
	}
	if rep.TotalExpectedControls != 42 {
		t.Errorf("expected 42 controls, got %d", rep.TotalExpectedControls)
	}
}

func TestVerifyArtifactParity_MediumCloudPBMM_Profile(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-medium-cloud-pbmm")
	if !ok {
		t.Fatalf("framework cccs-medium-cloud-pbmm not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-medium-cloud-pbmm", "profile.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY for profile, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
	}
	if rep.ArtifactKind != "profile" {
		t.Errorf("expected profile artifact kind, got %s", rep.ArtifactKind)
	}
}

func TestVerifyBytesParity_SyntheticDiscrepancies(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	baseBytes, err := fw.CatalogJSON()
	if err != nil {
		t.Fatalf("CatalogJSON: %v", err)
	}

	// 1. Verify exact in-memory bytes parity gives VERBATIM_PARITY
	rep, err := frameworks.VerifyBytesParity(fw, baseBytes, "memory://test.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity: %v", err)
	}
	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s", rep.Verdict)
	}

	// 2. Tampered Title
	var root map[string]interface{}
	if err := json.Unmarshal(baseBytes, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cat := root["catalog"].(map[string]interface{})
	groups := cat["groups"].([]interface{})
	firstGroup := groups[0].(map[string]interface{})
	ctrls := firstGroup["controls"].([]interface{})
	firstCtrl := ctrls[0].(map[string]interface{})
	firstCtrl["title"] = "Tampered Title That Does Not Match"

	tamperedBytes, _ := json.Marshal(root)
	repTampered, err := frameworks.VerifyBytesParity(fw, tamperedBytes, "memory://tampered.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity tampered: %v", err)
	}
	if repTampered.Verdict != frameworks.VerdictDeviationDetected {
		t.Errorf("expected DEVIATION_DETECTED on tampered title, got %s", repTampered.Verdict)
	}
	foundTitleDiscrepancy := false
	for _, d := range repTampered.Discrepancies {
		if d.Aspect == "Title" {
			foundTitleDiscrepancy = true
			break
		}
	}
	if !foundTitleDiscrepancy {
		t.Errorf("expected Title discrepancy, none found")
	}

	// 3. Missing control
	ctrlsRest := ctrls[1:]
	firstGroup["controls"] = ctrlsRest
	missingCtrlBytes, _ := json.Marshal(root)
	repMissing, err := frameworks.VerifyBytesParity(fw, missingCtrlBytes, "memory://missing.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity missing: %v", err)
	}
	if len(repMissing.MissingControls) == 0 {
		t.Errorf("expected MissingControls to have at least 1 item")
	}
}

func TestVerifyArtifactParity_SidecarMismatch(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	tmpDir := t.TempDir()
	catPath := filepath.Join(tmpDir, "catalog.json")
	sidecarPath := catPath + ".sha256"

	baseBytes, err := fw.CatalogJSON()
	if err != nil {
		t.Fatalf("CatalogJSON: %v", err)
	}

	if err := os.WriteFile(catPath, baseBytes, 0644); err != nil {
		t.Fatalf("WriteFile catPath: %v", err)
	}
	// Write wrong hash into sidecar
	wrongHash := hex.EncodeToString(sha256.New().Sum([]byte("corrupt")))
	if err := os.WriteFile(sidecarPath, []byte(wrongHash), 0644); err != nil {
		t.Fatalf("WriteFile sidecar: %v", err)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, catPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity: %v", err)
	}
	if rep.SidecarMatch {
		t.Errorf("expected sidecar match to be false on mismatch")
	}
	if rep.Verdict != frameworks.VerdictDeviationDetected {
		t.Errorf("expected DEVIATION_DETECTED on sidecar mismatch, got %s", rep.Verdict)
	}
}

func TestVerifyAllFrameworksParity(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	baseDir := filepath.Join("..", "..", "..", "data", "frameworks")

	reports, err := frameworks.VerifyAllFrameworksParity(reg, baseDir)
	if err != nil {
		t.Fatalf("VerifyAllFrameworksParity failed: %v", err)
	}

	if len(reports) == 0 {
		t.Fatalf("expected parity reports, got 0")
	}

	for _, r := range reports {
		t.Logf("Framework %s (%s): Verdict=%s Score=%.1f%% SchemaValid=%t IntegrityValid=%t",
			r.FrameworkID, r.ArtifactKind, r.Verdict, r.ParityScore, r.SchemaValid, r.IntegrityValid)
		if r.Verdict != frameworks.VerdictVerbatimParity {
			t.Errorf("framework %s (%s) failed verbatim parity: %d discrepancies",
				r.FrameworkID, r.ArtifactKind, len(r.Discrepancies))
		}
	}
}
