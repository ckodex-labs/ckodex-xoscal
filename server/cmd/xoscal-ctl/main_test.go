package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/kg"
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
	if len(fws) != 5 {
		t.Fatalf("expected 5 frameworks, got %d", len(fws))
	}

	// Export single
	if err := canadianframeworks.Export(tmpDir, "cccs-itsp-10-033"); err != nil {
		t.Fatalf("Export(cccs-itsp-10-033) failed: %v", err)
	}

	catFile := filepath.Join(tmpDir, "cccs-itsp-10-033", "catalog.json")
	if err := dbutil.CheckDigestSidecar(catFile); err != nil {
		t.Fatalf("digest verification failed for exported catalog: %v", err)
	}

	profFile := filepath.Join(tmpDir, "cccs-itsp-10-033", "profile.json")
	if err := dbutil.CheckDigestSidecar(profFile); err != nil {
		t.Fatalf("digest verification failed for exported profile: %v", err)
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

	// 1. Seed single framework into database for fast, deterministic 7-snapshot CLI integration testing
	runSeed([]string{"-dsn", dbPath, "-framework", "nist-csf-2.0"})

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

	// 3. Test generate assessment-results-only with validation on v1.0
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

	// 4. Test full 7-layer generate from seeded "baseline" snapshot with schema validation
	fullOutDir := filepath.Join(tmpDir, "generated-full")
	runGenerate([]string{
		"-dsn", dbPath,
		"-snapshot", "baseline",
		"-framework", "nist-csf-2.0",
		"-out", fullOutDir,
		"-validate",
	})
	for _, f := range []string{
		"catalog.json",
		"profile.json",
		"component-definition.json",
		"system-security-plan.json",
		"assessment-plan.json",
		"assessment-results.json",
		"plan-of-action-and-milestones.json",
	} {
		artFile := filepath.Join(fullOutDir, f)
		if _, err := os.Stat(artFile); err != nil {
			t.Fatalf("expected %s generated in full 7-layer output, got err: %v", f, err)
		}
		if err := dbutil.CheckDigestSidecar(artFile); err != nil {
			t.Fatalf("expected valid .sha256 sidecar for generated %s, got err: %v", f, err)
		}
	}
}

func TestXoscalCtl_FullRegistrySeedAndGenerate(t *testing.T) {
	prevProcs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(prevProcs)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "full-registry.db")
	ctx := context.Background()

	start := time.Now()
	store, err := kg.NewSQLiteStore(dbPath, dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer func() { _ = store.Close() }()

	reg := frameworks.DefaultRegistry()
	if got := len(reg.List()); got != 36 {
		t.Fatalf("expected 36 registered frameworks, got %d", got)
	}
	t0 := time.Now()
	if err := reg.SeedKG(ctx, store); err != nil {
		t.Fatalf("SeedKG failed: %v", err)
	}
	seedDur := time.Since(t0)

	t1 := time.Now()
	if _, err := store.CreateSnapshot(ctx, "baseline"); err != nil {
		t.Fatalf("CreateSnapshot(baseline) failed: %v", err)
	}
	snapDur := time.Since(t1)

	t2 := time.Now()
	gen := oscal.NewGenerator(store)
	res, err := gen.GenerateAllArtifacts(ctx, "baseline", "nist_csf_2", nil)
	if err != nil {
		t.Fatalf("GenerateAllArtifacts(nist_csf_2) failed: %v", err)
	}
	genDur := time.Since(t2)
	if res.Catalog == nil || (len(res.Catalog.Groups) == 0 && len(res.Catalog.Controls) == 0) {
		t.Fatalf("expected non-empty catalog generated for nist_csf_2")
	}

	elapsed := time.Since(start)
	t.Logf("Timings (GOMAXPROCS=2): SeedKG=%v CreateSnapshot=%v GenerateAllArtifacts=%v Total=%v", seedDur, snapDur, genDur, elapsed)
	if genDur >= 10*time.Second {
		t.Fatalf("expected single-pass GenerateAllArtifacts to complete in < 10s under GOMAXPROCS=2, took %v", genDur)
	}
	if elapsed >= 60*time.Second {
		t.Fatalf("expected full registry seed + generate to complete in < 60s under GOMAXPROCS=2, took %v", elapsed)
	}
}
