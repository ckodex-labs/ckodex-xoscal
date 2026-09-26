package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/tabular"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

func TestXoscalCtl_ScaffoldAndVerify(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Test init
	scan, jsonBytes, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("ScaffoldWorkspace failed: %v", err)
	}
	if scan == nil || len(jsonBytes) == 0 {
		t.Fatalf("expected non-empty scaffold scan and JSON")
	}

	compPath := filepath.Join(tmpDir, "component-definition.json")

	// 2. Test verify
	state, err := vectorstate.EvaluateArtifact(compPath, "")
	if err != nil {
		t.Fatalf("EvaluateArtifact failed: %v", err)
	}
	if state.Lifecycle != vectorstate.ModeNormal {
		t.Errorf("expected ModeNormal, got %s", state.Lifecycle)
	}
	if state.TotalControls == 0 {
		t.Errorf("expected controls evaluated")
	}

	// 3. Test validate via schemavalidate
	data, _ := os.ReadFile(compPath)
	kind, err := resolveKind("", data)
	if err != nil {
		t.Fatalf("resolveKind failed: %v", err)
	}
	if kind.Name != "component-definition" {
		t.Errorf("expected kind component-definition, got %s", kind.Name)
	}
}

func TestXoscalCtl_DerogateLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	derogPath := filepath.Join(tmpDir, "derogations.json")

	store, err := derogation.Load(derogPath)
	if err != nil {
		t.Fatalf("derogation.Load failed: %v", err)
	}

	entry := derogation.DerogationEntry{
		ControlID:     "ac-2",
		Scope:         "service:auth",
		Authority:     "urn:xoscal:auth:ciso",
		Justification: "Hardware MFA tokens pending delivery",
		ExpiresAt:     time.Now().Add(14 * 24 * time.Hour),
	}

	if err := store.Add(entry); err != nil {
		t.Fatalf("store.Add failed: %v", err)
	}
	if err := store.Save(derogPath); err != nil {
		t.Fatalf("store.Save failed: %v", err)
	}

	// Verify loaded entry
	loaded, err := derogation.Load(derogPath)
	if err != nil {
		t.Fatalf("reload derogations: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 derogation entry, got %d", len(loaded.Entries))
	}
	active := loaded.GetActiveForControl("ac-2", time.Now())
	if active == nil {
		t.Fatalf("expected active derogation for ac-2")
	}

	// Revoke
	if err := loaded.Revoke(loaded.Entries[0].ID); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	if activeAfter := loaded.GetActiveForControl("ac-2", time.Now()); activeAfter != nil {
		t.Fatalf("expected nil active derogation after revocation")
	}
}

func TestXoscalCtl_TabularExportImport(t *testing.T) {
	tmpDir := t.TempDir()

	_, jsonBytes, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	// CSV export
	csvBytes, err := tabular.ExportToCSV(jsonBytes)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}
	if len(csvBytes) == 0 {
		t.Fatalf("exported CSV is empty")
	}

	// CSV import
	updatedJSON, err := tabular.ImportFromCSV(jsonBytes, csvBytes)
	if err != nil {
		t.Fatalf("ImportFromCSV failed: %v", err)
	}
	if len(updatedJSON) == 0 {
		t.Fatalf("imported JSON is empty")
	}
}

func TestXoscalCtl_FrameworksExportAndList(t *testing.T) {
	tmpDir := t.TempDir()

	fws := canadianframeworks.ListFrameworks()
	if len(fws) != 4 {
		t.Fatalf("expected 4 frameworks, got %d", len(fws))
	}

	// Export single
	if err := canadianframeworks.Export(tmpDir, "cccs-itsp-10-171"); err != nil {
		t.Fatalf("Export(cccs-itsp-10-171) failed: %v", err)
	}

	catFile := filepath.Join(tmpDir, "cccs-itsp-10-171", "catalog.json")
	if err := dbutil.CheckDigestSidecar(catFile); err != nil {
		t.Fatalf("digest verification failed for exported catalog: %v", err)
	}
}

func TestXoscalCtl_KindResolution(t *testing.T) {
	cat := canadianframeworks.BuildCyberSecureCanadaCatalog()
	data, err := oscal.ExportCatalogJSON(cat)
	if err != nil {
		t.Fatalf("export catalog: %v", err)
	}

	kind, err := resolveKind("", data)
	if err != nil {
		t.Fatalf("resolveKind failed: %v", err)
	}
	if kind.RootKey != "catalog" {
		t.Errorf("expected root key catalog, got %s", kind.RootKey)
	}

	explicitKind, err := kindFromName("component-definition")
	if err != nil {
		t.Fatalf("kindFromName failed: %v", err)
	}
	if explicitKind.RootKey != "component-definition" {
		t.Errorf("expected component-definition, got %s", explicitKind.RootKey)
	}
}

func TestXoscalCtl_GenerateAndIngest(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-kg.db")
	outDir := filepath.Join(tmpDir, "generated")

	// 1. Seed database
	runSeed([]string{"-dsn", dbPath})

	// 2. Prepare sample requirement JSON to test ingest
	sampleReq := `[
		{
			"id": "REQ-AI-001",
			"title": "High-risk AI System Risk Management",
			"description": "Establish, implement, document and maintain a risk management system.",
			"article": "Article 9",
			"category": "Risk Management"
		}
	]`
	reqPath := filepath.Join(tmpDir, "req.json")
	if err := os.WriteFile(reqPath, []byte(sampleReq), 0600); err != nil {
		t.Fatalf("write sample req: %v", err)
	}

	runIngest([]string{"-dsn", dbPath, "-input", reqPath, "-framework", "eu-ai-act"})

	// 3. Test generate with validation
	runGenerate([]string{
		"-dsn", dbPath,
		"-snapshot", "v1.0",
		"-framework", "eu-ai-act",
		"-out", outDir,
		"-assessment-results-only",
		"-validate",
	})

	arPath := filepath.Join(outDir, "assessment-results.json")
	if _, err := os.Stat(arPath); err != nil {
		t.Fatalf("expected assessment-results.json generated, got err: %v", err)
	}
}
