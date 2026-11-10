package cli_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/cli"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
)

// captureOutput captures stdout during f's execution without deadlocking on >64KB outputs.
func captureOutput(f func()) string {
	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	f()

	_ = w.Close()
	os.Stdout = origOut
	<-done
	_ = r.Close()

	return buf.String()
}

func TestPrintUsage(t *testing.T) {
	out := captureOutput(func() {
		cli.PrintUsage("v0.4.0-test")
	})
	if !strings.Contains(out, "Unified xOSCAL Compliance & Governance Engine") {
		t.Errorf("expected usage title in output")
	}
	if !strings.Contains(out, "controls") || !strings.Contains(out, "compare") || !strings.Contains(out, "serve") {
		t.Errorf("expected new commands in usage output")
	}
}

func TestQuickstart_NonDestructive(t *testing.T) {
	out := captureOutput(func() {
		cli.RunQuickstart([]string{})
	})
	if !strings.Contains(out, "xOSCAL DEVELOPER ONBOARDING QUICKSTART") {
		t.Errorf("expected quickstart header in output")
	}
	if !strings.Contains(out, "Ready to scaffold") {
		t.Errorf("expected ready to scaffold in output")
	}
}

func TestVerifyParity_CLI(t *testing.T) {
	out := captureOutput(func() {
		cli.RunVerifyParity([]string{
			"-framework", "cybersecure-canada",
			"-dir", "../../../data/frameworks",
			"-strict=false",
		})
	})
	if !strings.Contains(out, "xOSCAL FRAMEWORK VERBATIM PARITY REPORT") {
		t.Errorf("expected parity report header, got: %s", out)
	}
	if !strings.Contains(out, "VERBATIM_PARITY") {
		t.Errorf("expected VERBATIM_PARITY in output, got: %s", out)
	}
}

func TestVerifyParity_CLI_JSON(t *testing.T) {
	out := captureOutput(func() {
		cli.RunVerifyParity([]string{
			"-framework", "cybersecure-canada",
			"-dir", "../../../data/frameworks",
			"-json",
			"-strict=false",
		})
	})
	if !strings.Contains(out, `"verdict": "VERBATIM_PARITY"`) {
		t.Errorf("expected JSON verdict VERBATIM_PARITY, got: %s", out)
	}
}

func TestDoctor_CLI(t *testing.T) {
	out := captureOutput(func() {
		cli.RunDoctor([]string{}, "v0.4.0-test")
	})
	if !strings.Contains(out, "xOSCAL Doctor: Deep Diagnostics") {
		t.Errorf("expected doctor header, got: %s", out)
	}
	if !strings.Contains(out, "SYSTEM_OPERATIONAL") {
		t.Errorf("expected SYSTEM_OPERATIONAL verdict, got: %s", out)
	}
}

func TestDoctor_CLI_JSON(t *testing.T) {
	out := captureOutput(func() {
		cli.RunDoctor([]string{"-json"}, "v0.4.0-test")
	})
	if !strings.Contains(out, `"verdict": "SYSTEM_OPERATIONAL"`) {
		t.Errorf("expected JSON verdict SYSTEM_OPERATIONAL, got: %s", out)
	}
	if !strings.Contains(out, `"schema_engine_tier1"`) {
		t.Errorf("expected schema_engine_tier1 in JSON, got: %s", out)
	}
	if !strings.Contains(out, `"ontology_graph_feeds"`) {
		t.Errorf("expected ontology_graph_feeds check in doctor JSON, got: %s", out)
	}
}

func TestFrameworksList_JSON_AndPlaceholders(t *testing.T) {
	outAll := captureOutput(func() {
		cli.RunFrameworks([]string{"list", "-json"})
	})
	var allInfos []frameworks.Info
	if err := json.Unmarshal([]byte(outAll), &allInfos); err != nil {
		t.Fatalf("failed to unmarshal frameworks list -json: %v\nout=%s", err, outAll)
	}
	if len(allInfos) < 5 {
		t.Errorf("expected at least 5 frameworks, got %d", len(allInfos))
	}

	outCA := captureOutput(func() {
		cli.RunFrameworks([]string{"list", "-jurisdiction", "ca", "-json"})
	})
	var caInfos []frameworks.Info
	if err := json.Unmarshal([]byte(outCA), &caInfos); err != nil {
		t.Fatalf("failed to unmarshal frameworks list -jurisdiction ca -json: %v", err)
	}
	if len(caInfos) != 5 {
		t.Errorf("expected 5 Canadian frameworks, got %d", len(caInfos))
	}

	outPlaceholder := captureOutput(func() {
		cli.RunFrameworks([]string{"list", "-jurisdiction", "${jurisdiction}", "-json"})
	})
	var phInfos []frameworks.Info
	if err := json.Unmarshal([]byte(outPlaceholder), &phInfos); err != nil {
		t.Fatalf("failed to unmarshal frameworks list with placeholder: %v", err)
	}
	if len(phInfos) != len(allInfos) {
		t.Errorf("expected unexpanded ${jurisdiction} to return all %d frameworks, got %d", len(allInfos), len(phInfos))
	}
}

func TestControls_JSON_TrailingFlags_AndPlaceholders(t *testing.T) {
	// 1. controls search with trailing flags and -json
	outSearch := captureOutput(func() {
		cli.RunControls([]string{"search", "account", "-framework", "cccs-itsp-10-033", "-json"})
	})
	var searchRes []frameworks.ControlInfo
	if err := json.Unmarshal([]byte(outSearch), &searchRes); err != nil {
		t.Fatalf("failed to unmarshal controls search -json: %v\nout=%s", err, outSearch)
	}
	if len(searchRes) == 0 {
		t.Errorf("expected search results for 'account'")
	}

	// 2. controls search with unexpanded ${framework} placeholder
	outSearchPH := captureOutput(func() {
		cli.RunControls([]string{"search", "-framework", "${framework}", "-json", "account"})
	})
	var searchPHRes []frameworks.ControlInfo
	if err := json.Unmarshal([]byte(outSearchPH), &searchPHRes); err != nil {
		t.Fatalf("failed to unmarshal controls search with ${framework}: %v", err)
	}
	if len(searchPHRes) != len(searchRes) {
		t.Errorf("expected ${framework} fallback to match cccs-itsp-10-033 (%d != %d)", len(searchPHRes), len(searchRes))
	}

	// 3. controls get with trailing -framework and -json (exact UTCP manual invocation)
	outGet := captureOutput(func() {
		cli.RunControls([]string{"get", "ac-2", "-framework", "cccs-itsp-10-033", "-json"})
	})
	var ctrl frameworks.ControlInfo
	if err := json.Unmarshal([]byte(outGet), &ctrl); err != nil {
		t.Fatalf("failed to unmarshal controls get -json: %v\nout=%s", err, outGet)
	}
	if !strings.EqualFold(ctrl.ID, "ac-2") {
		t.Errorf("expected control ac-2, got %q", ctrl.ID)
	}

	// 4. controls list with -json and unexpanded ${family}
	outList := captureOutput(func() {
		cli.RunControls([]string{"list", "-framework", "cybersecure-canada", "-family", "${family}", "-json"})
	})
	var listRes []frameworks.ControlInfo
	if err := json.Unmarshal([]byte(outList), &listRes); err != nil {
		t.Fatalf("failed to unmarshal controls list -json: %v\nout=%s", err, outList)
	}
	if len(listRes) != 46 {
		t.Errorf("expected 46 controls in cybersecure-canada, got %d", len(listRes))
	}
}

func TestCompare_JSON_LeadingAndTrailingFlags(t *testing.T) {
	outLeading := captureOutput(func() {
		cli.RunCompare([]string{"-json", "cccs-itsg-33", "cccs-itsp-10-033"})
	})
	var compLeading frameworks.FrameworkComparison
	if err := json.Unmarshal([]byte(outLeading), &compLeading); err != nil {
		t.Fatalf("failed to unmarshal compare -json leading: %v\nout=%s", err, outLeading)
	}
	if compLeading.FromFramework != "cccs-itsg-33" || compLeading.ToFramework != "cccs-itsp-10-033" {
		t.Errorf("unexpected comparison IDs: %s -> %s", compLeading.FromFramework, compLeading.ToFramework)
	}

	outTrailing := captureOutput(func() {
		cli.RunCompare([]string{"cccs-itsg-33", "cccs-itsp-10-033", "-json"})
	})
	var compTrailing frameworks.FrameworkComparison
	if err := json.Unmarshal([]byte(outTrailing), &compTrailing); err != nil {
		t.Fatalf("failed to unmarshal compare -json trailing: %v\nout=%s", err, outTrailing)
	}
	if compTrailing.ToTotal != compLeading.ToTotal {
		t.Errorf("leading and trailing -json output mismatch: %d != %d", compTrailing.ToTotal, compLeading.ToTotal)
	}
}

func TestSeedAndGenerate_DBAlias_AndPlaceholders(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "seed.db")
	outDir := filepath.Join(tmpDir, "out")

	_ = captureOutput(func() {
		cli.RunSeed([]string{
			"-db", dbPath,
			"-framework", "cybersecure-canada",
			"-snapshot", "${snapshot}",
		})
	})

	_ = captureOutput(func() {
		cli.RunGenerate([]string{
			"-db", dbPath,
			"-snapshot", "baseline",
			"-framework", "cybersecure-canada",
			"-out", outDir,
			"-validate",
		})
	})

	catPath := filepath.Join(outDir, "catalog.json")
	if _, err := os.Stat(catPath); err != nil {
		t.Fatalf("expected generated catalog.json at %s: %v", catPath, err)
	}
	if _, err := os.Stat(catPath + ".sha256"); err != nil {
		t.Fatalf("expected generated catalog.json.sha256 sidecar: %v", err)
	}

	// Verify RunVerify with positional path + trailing -json flag
	outVerify := captureOutput(func() {
		cli.RunVerify([]string{catPath, "-json"})
	})
	if !strings.Contains(outVerify, `"presence"`) {
		t.Errorf("expected JSON VectorState output from RunVerify with trailing -json, got: %s", outVerify)
	}

	// Verify UTCP_ARG_<param>_UTCP_END placeholder handling and --flag=value syntax
	outUTCPArg := captureOutput(func() {
		cli.RunControls([]string{"search", "--framework=UTCP_ARG_framework_UTCP_END", "--json", "--", "account"})
	})
	var utcpSearchRes []frameworks.ControlInfo
	if err := json.Unmarshal([]byte(outUTCPArg), &utcpSearchRes); err != nil {
		t.Fatalf("failed to unmarshal controls search with UTCP_ARG placeholder and -- terminator: %v\nout=%s", err, outUTCPArg)
	}
	if len(utcpSearchRes) == 0 {
		t.Errorf("expected search results for 'account' with UTCP_ARG_framework_UTCP_END fallback")
	}
}

func TestOntology_Graph_Feeds_CLI(t *testing.T) {
	outOnt := captureOutput(func() {
		cli.RunOntology([]string{"spec", "-json"})
	})
	if !strings.Contains(outOnt, `"https://xoscal.dev/ns/ontology/v1#"`) {
		t.Errorf("expected ontology namespace in RunOntology spec -json, got: %s", outOnt)
	}

	outClasses := captureOutput(func() {
		cli.RunOntology([]string{"classes", "-json"})
	})
	if !strings.Contains(outClasses, `"threat:KEVAdvisory"`) {
		t.Errorf("expected threat:KEVAdvisory in RunOntology classes -json")
	}

	outPreds := captureOutput(func() {
		cli.RunOntology([]string{"predicates", "-json"})
	})
	if !strings.Contains(outPreds, `"ATTACKS_CONTROL"`) {
		t.Errorf("expected ATTACKS_CONTROL in RunOntology predicates -json")
	}

	outCW := captureOutput(func() {
		cli.RunGraph([]string{"crosswalk", "-control", "ac-2", "-json"})
	})
	if !strings.Contains(outCW, `"canonical_control_id": "ac-2"`) {
		t.Errorf("expected canonical_control_id ac-2 in RunGraph crosswalk -json, got: %s", outCW)
	}

	outBR := captureOutput(func() {
		cli.RunGraph([]string{"blast-radius", "-target", "CVE-2024-3400", "-depth", "2", "-json"})
	})
	if !strings.Contains(outBR, `"CVE-2024-3400"`) {
		t.Errorf("expected CVE-2024-3400 in RunGraph blast-radius -json, got: %s", outBR)
	}

	outExport := captureOutput(func() {
		cli.RunGraph([]string{"export", "-target", "CVE-2024-3400", "-format", "cypher"})
	})
	if !strings.Contains(outExport, "MERGE") {
		t.Errorf("expected Cypher MERGE statements in RunGraph export, got: %s", outExport)
	}

	outRec := captureOutput(func() {
		cli.RunGraph([]string{"reconcile", "-json"})
	})
	if !strings.Contains(outRec, `"coherent": true`) {
		t.Errorf("expected coherent: true in RunGraph reconcile -json, got: %s", outRec)
	}

	outFeeds := captureOutput(func() {
		cli.RunFeeds([]string{"list", "-json"})
	})
	if !strings.Contains(outFeeds, `"cisa-kev-active-threats"`) {
		t.Errorf("expected cisa-kev-active-threats in RunFeeds list -json, got: %s", outFeeds)
	}

	cpDir := filepath.Join(t.TempDir(), "cp")
	outSync := captureOutput(func() {
		cli.RunFeeds([]string{"sync", "-offline", "-checkpoint-dir", cpDir, "-json"})
	})
	if !strings.Contains(outSync, `"bundle_sha256"`) || !strings.Contains(outSync, `"operational_mode": "NORMAL"`) {
		t.Errorf("expected bundle_sha256 and operational_mode NORMAL in RunFeeds sync, got: %s", outSync)
	}

	outVerifyCP := captureOutput(func() {
		cli.RunFeeds([]string{"verify-checkpoint", "-checkpoint-dir", cpDir, "-json"})
	})
	if !strings.Contains(outVerifyCP, `"verdict": "VERIFIED"`) || !strings.Contains(outVerifyCP, `"ledger_entries": 1`) {
		t.Errorf("expected VERIFIED and ledger_entries 1 in RunFeeds verify-checkpoint -json, got: %s", outVerifyCP)
	}
}
