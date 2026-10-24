// Package cli provides the unified CLI commands for xoscal and xoscal-ctl.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mchorfa/xoscal/server/internal/bundle"
	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/completion"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/diff"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"github.com/mchorfa/xoscal/server/internal/tabular"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

// PrintUsage prints command-line usage information.
func PrintUsage(version string) {
	binName := filepath.Base(os.Args[0])
	if binName == "" {
		binName = "xoscal"
	}

	fmt.Printf(`%s v%s — Unified xOSCAL Compliance & Governance Engine

USAGE:
  %s <command> [options]

COMMANDS:
  controls       Search and inspect controls across registered frameworks
  compare        Compare two frameworks and compute sovereign transition delta
  serve          Start local zero-dependency xOSCAL Web Studio & Explorer
  ui             Launch local xOSCAL Web Studio in default browser
  quickstart     Guided onboarding walkthrough
  init           Auto-detect repository stack and scaffold OSCAL 1.2.3 Component Definition
  generate       Generate OSCAL 1.2.3 artifacts from Knowledge Graph snapshot
  ingest         Ingest raw regulatory requirements JSON into Knowledge Graph
  verify         Evaluate artifact against schemas and compute multi-dimensional Vector State
  validate       Fast standalone validation of OSCAL JSON artifacts against official schemas
  diff           Compute semantic compliance delta between base and PR head (GitOps)
  derogate       Manage formal time-bounded risk exceptions (Rule 23)
  tabular        Convert between OSCAL JSON and standard CSV/Excel tables for GRC analysts
  bundle-export  Package an air-gapped, verifiable audit bundle with offline HTML viewer
  frameworks     List, export, compare, or verify authoritative compliance frameworks
  verify-parity  Verify verbatim parity between framework definitions and created OSCAL artifacts
  seed           Seed Knowledge Graph database with authoritative baseline frameworks
  completion     Generate shell autocompletions (bash, zsh, fish)
  version        Print CLI version and exit

Run '%s <command> -h' for details on specific command options.
`, binName, version, binName, binName)
}

// Execute parses command arguments and runs the appropriate subcommand.
func Execute(args []string, version string) {
	if len(args) < 2 {
		PrintUsage(version)
		os.Exit(1)
	}

	cmd := args[1]
	switch cmd {
	case "controls":
		RunControls(args[2:])
	case "compare":
		RunCompare(args[2:])
	case "serve":
		RunServe(args[2:])
	case "ui":
		RunServe(append([]string{"-open"}, args[2:]...))
	case "quickstart":
		RunQuickstart(args[2:])
	case "completion":
		RunCompletion(args[2:])
	case "init":
		RunInit(args[2:])
	case "generate":
		RunGenerate(args[2:])
	case "ingest":
		RunIngest(args[2:])
	case "verify", "check":
		RunVerify(args[2:])
	case "validate":
		RunValidate(args[2:])
	case "diff":
		RunDiff(args[2:])
	case "derogate":
		RunDerogate(args[2:])
	case "tabular":
		RunTabular(args[2:])
	case "bundle-export", "export-bundle":
		RunBundleExport(args[2:])
	case "frameworks":
		RunFrameworks(args[2:])
	case "verify-parity":
		RunVerifyParity(args[2:])
	case "seed":
		RunSeed(args[2:])
	case "version", "-version", "--version":
		fmt.Printf("xoscal version %s\n", version)
	case "help", "-h", "-help", "--help":
		PrintUsage(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		PrintUsage(version)
		os.Exit(1)
	}
}

// RunInit scaffolds a new OSCAL component definition based on repo inspection.
func RunInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dir := fs.String("dir", ".", "Path to repository root")
	framework := fs.String("framework", "nist-sp-800-53-rev5", "Compliance framework baseline")
	_ = fs.Parse(args)

	fmt.Printf("Scanning repository at: %s ...\n", *dir)
	scan, jsonBytes, err := scaffold.ScaffoldWorkspace(*dir, *framework)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scaffolding workspace: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Detected Project: %s (Languages: %v, Docker: %v, K8s: %v, TF: %v)\n",
		scan.Name, scan.Languages, scan.HasDocker, scan.HasK8s, scan.HasTF)
	fmt.Printf("[OK] Mapped %d component(s) using virtual URNs -> deterministic UUIDv5\n", len(scan.Components))
	fmt.Printf("[OK] Verified compliance with official NIST OSCAL 1.2.3 JSON schema\n")
	fmt.Printf("[OK] Created workspace config: %s\n", filepath.Join(*dir, ".xoscal", "xoscal.yaml"))
	fmt.Printf("[OK] Emitted component definition: %s (%d bytes)\n",
		filepath.Join(*dir, "component-definition.json"), len(jsonBytes))
	fmt.Printf("\nNext step: Run 'xoscal verify' to check control posture.\n")
}

// RunVerify evaluates artifacts and outputs Vector State.
func RunVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	file := fs.String("file", "", "Path to OSCAL JSON artifact to evaluate")
	evidenceDir := fs.String("evidence-dir", "", "Optional path to evidence directory with SHA-256 sidecars")
	jsonOut := fs.Bool("json", false, "Output Vector State as JSON")
	_ = fs.Parse(args)

	targetFile := *file
	if targetFile == "" && fs.NArg() > 0 {
		targetFile = fs.Arg(0)
	}

	if targetFile == "" {
		candidates := []string{
			"component-definition.json",
			filepath.Join(".xoscal", "component-definition.json"),
			"ssp.json",
			"system-security-plan.json",
			"catalog.json",
			"profile.json",
			"assessment-results.json",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				targetFile = c
				break
			}
		}
	}

	if targetFile == "" {
		fmt.Fprintf(os.Stderr, "no OSCAL artifact specified and none found in current directory.\n\n")
		fmt.Fprintf(os.Stderr, "To scaffold a new component definition, run:\n")
		fmt.Fprintf(os.Stderr, "  xoscal init\n\n")
		fmt.Fprintf(os.Stderr, "Or specify an existing artifact:\n")
		fmt.Fprintf(os.Stderr, "  xoscal verify -file <path/to/artifact.json>\n")
		os.Exit(1)
	}

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		fallback := filepath.Join(".xoscal", targetFile)
		if _, err := os.Stat(fallback); err == nil {
			targetFile = fallback
		}
	}

	state, err := vectorstate.EvaluateArtifact(targetFile, *evidenceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error evaluating artifact: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(state)
	} else {
		fmt.Print(state.FormatTable(true))
	}

	if state.Lifecycle == vectorstate.ModeFailed || state.Lifecycle == vectorstate.ModeQuarantined {
		os.Exit(3)
	}
}

// RunDiff computes compliance delta between baseline and PR head.
func RunDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	base := fs.String("base", "main.component-definition.json", "Base OSCAL JSON artifact")
	head := fs.String("head", "component-definition.json", "Head OSCAL JSON artifact (PR branch)")
	markdown := fs.Bool("markdown", false, "Output as Markdown report for PR comment / CI step summary")
	failOnRegression := fs.Bool("fail-on-regression", false, "Exit code 3 if any compliance regression is detected")
	_ = fs.Parse(args)

	d, err := diff.CompareArtifacts(*base, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error comparing artifacts: %v\n", err)
		os.Exit(1)
	}

	if *markdown {
		fmt.Print(d.FormatMarkdown())
	} else {
		fmt.Printf("\n=== xOSCAL COMPLIANCE POSTURE DIFF ===\n")
		fmt.Printf("Base: %s | Head: %s\n\n", *base, *head)
		fmt.Printf("Vector Delta: Base <%s, %s, Anti=%d> -> Head <%s, %s, Anti=%d>\n",
			d.BaseVector.Presence, d.BaseVector.Lifecycle, d.BaseVector.AntiCount,
			d.HeadVector.Presence, d.HeadVector.Lifecycle, d.HeadVector.AntiCount)
		fmt.Printf("Summary: %d Added | %d Modified | %d Removed\n\n", d.AddedCount, d.ModifiedCount, d.RemovedCount)
		for _, delta := range d.Deltas {
			regrBadge := ""
			if delta.Regression {
				regrBadge = " [REGRESSION]"
			}
			fmt.Printf("  %-10s %-12s %s%s\n", delta.ChangeType, delta.ControlID, delta.Component, regrBadge)
		}
		fmt.Println()
	}

	if *failOnRegression && d.HasRegression {
		fmt.Fprintf(os.Stderr, "[ERROR] Compliance posture regressed! Blocking gate.\n")
		os.Exit(3)
	}
}

// RunDerogate manages time-bounded risk acceptances per Rule 23.
func RunDerogate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal derogate [add|list|revoke] [options]")
		os.Exit(1)
	}

	sub := args[0]
	derogPath := filepath.Join(".xoscal", "derogations.json")
	store, err := derogation.Load(derogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading derogations: %v\n", err)
		os.Exit(1)
	}

	switch sub {
	case "add":
		fs := flag.NewFlagSet("derogate add", flag.ExitOnError)
		ctrlID := fs.String("control", "", "Target control ID to derogate (e.g. sc-8)")
		scope := fs.String("scope", "repository", "Operational scope (e.g. service:worker)")
		authority := fs.String("authority", "", "Authorizing owner URN (e.g. urn:xoscal:auth:ciso-alice)")
		justification := fs.String("reason", "", "Formal risk justification")
		ttlDays := fs.Int("ttl-days", 30, "Time-to-live in days until automatic expiration")
		compensating := fs.String("compensating", "", "Comma-separated list of compensating controls")
		_ = fs.Parse(args[1:])

		if *ctrlID == "" || *authority == "" || *justification == "" {
			fmt.Fprintln(os.Stderr, "error: --control, --authority, and --reason are required")
			os.Exit(1)
		}

		var compList []string
		if *compensating != "" {
			for _, c := range strings.Split(*compensating, ",") {
				compList = append(compList, strings.TrimSpace(c))
			}
		}

		entry := derogation.DerogationEntry{
			ControlID:            *ctrlID,
			Scope:                *scope,
			Authority:            *authority,
			Justification:        *justification,
			CompensatingControls: compList,
			ExpiresAt:            time.Now().Add(time.Duration(*ttlDays) * 24 * time.Hour),
		}

		if err := store.Add(entry); err != nil {
			fmt.Fprintf(os.Stderr, "error adding derogation: %v\n", err)
			os.Exit(1)
		}

		if err := store.Save(derogPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving derogation: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Derogation registered for %s (Expires: %s, Authority: %s)\n",
			*ctrlID, entry.ExpiresAt.Format(time.RFC3339), *authority)

	case "list":
		now := time.Now()
		fmt.Printf("\n=== REGISTERED RISK DEROGATIONS (Rule 23) ===\n")
		fmt.Printf("%-20s %-10s %-12s %-25s %s\n", "ID", "CONTROL", "STATUS", "AUTHORITY", "EXPIRES IN")
		fmt.Println(strings.Repeat("-", 80))
		for _, e := range store.Entries {
			status := "ACTIVE"
			if e.Revoked {
				status = "REVOKED"
			} else if e.IsExpired(now) {
				status = "EXPIRED"
			}
			rem := e.TimeRemaining(now).Round(time.Hour).String()
			if status == "EXPIRED" {
				rem = "EXPIRED"
			}
			fmt.Printf("%-20s %-10s %-12s %-25s %s\n", e.ID, e.ControlID, status, e.Authority, rem)
		}
		fmt.Println()

	case "revoke":
		fs := flag.NewFlagSet("derogate revoke", flag.ExitOnError)
		id := fs.String("id", "", "Derogation ID to revoke")
		_ = fs.Parse(args[1:])

		if *id == "" {
			fmt.Fprintln(os.Stderr, "error: --id is required")
			os.Exit(1)
		}

		if err := store.Revoke(*id); err != nil {
			fmt.Fprintf(os.Stderr, "error revoking derogation: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(derogPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving store: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Derogation %s revoked successfully.\n", *id)

	default:
		fmt.Fprintf(os.Stderr, "unknown derogate subcommand: %s (expected 'add', 'list', or 'revoke')\n", sub)
		os.Exit(1)
	}
}

// RunTabular bridges OSCAL and CSV for GRC analysts.
func RunTabular(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal tabular [export|import] [options]")
		os.Exit(1)
	}

	sub := args[0]
	switch sub {
	case "export":
		fs := flag.NewFlagSet("tabular export", flag.ExitOnError)
		inFile := fs.String("in", "component-definition.json", "Input OSCAL JSON artifact")
		outFile := fs.String("out", "controls.csv", "Output CSV file path")
		_ = fs.Parse(args[1:])

		data, err := os.ReadFile(*inFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading input file: %v\n", err)
			os.Exit(1)
		}

		csvBytes, err := tabular.ExportToCSV(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting to CSV: %v\n", err)
			os.Exit(1)
		}

		// #nosec G703 -- The local invoking operator selects --out; artifact fields cannot select this destination.
		if err := os.WriteFile(*outFile, csvBytes, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Exported %s -> %s for spreadsheet review.\n", *inFile, *outFile)

	case "import":
		fs := flag.NewFlagSet("tabular import", flag.ExitOnError)
		csvFile := fs.String("in", "controls.csv", "Edited CSV table path")
		baseFile := fs.String("base", "component-definition.json", "Base OSCAL JSON artifact to update")
		outFile := fs.String("out", "component-definition.updated.json", "Output updated OSCAL JSON path")
		_ = fs.Parse(args[1:])

		baseBytes, err := os.ReadFile(*baseFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading base OSCAL file: %v\n", err)
			os.Exit(1)
		}

		csvBytes, err := os.ReadFile(*csvFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading CSV file: %v\n", err)
			os.Exit(1)
		}

		updatedJSON, err := tabular.ImportFromCSV(baseBytes, csvBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error importing from CSV: %v\n", err)
			os.Exit(1)
		}

		// #nosec G703 -- The local invoking operator selects --out; imported CSV cells cannot select this destination.
		if err := os.WriteFile(*outFile, updatedJSON, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Imported edits from %s -> %s (OSCAL 1.2.3 schema validated).\n", *csvFile, *outFile)

	default:
		fmt.Fprintf(os.Stderr, "unknown tabular subcommand: %s (expected 'export' or 'import')\n", sub)
		os.Exit(1)
	}
}

// RunBundleExport packages artifacts and offline HTML viewer into a verifiable archive.
func RunBundleExport(args []string) {
	fs := flag.NewFlagSet("bundle-export", flag.ExitOnError)
	inFile := fs.String("in", "component-definition.json", "Primary OSCAL JSON artifact")
	evidenceDir := fs.String("evidence-dir", "", "Optional evidence directory")
	outBundle := fs.String("out", "audit-bundle.tar.gz", "Output archive path (.tar.gz)")
	_ = fs.Parse(args)

	if _, err := os.Stat(*inFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "artifact file %s does not exist\n", *inFile)
		os.Exit(1)
	}

	fmt.Printf("[INFO] Creating Air-Gapped Verifiable Audit Bundle: %s ...\n", *outBundle)
	if err := bundle.CreateAuditBundle([]string{*inFile}, *evidenceDir, *outBundle); err != nil {
		fmt.Fprintf(os.Stderr, "error creating bundle: %v\n", err)
		os.Exit(1)
	}

	manifest, err := bundle.VerifyBundle(*outBundle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bundle verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Packaged %d OSCAL artifact(s) with SHA-256 sidecars\n", len(manifest.Artifacts))
	fmt.Printf("[OK] Packaged %d evidence blob(s) with SHA-256 sidecars\n", len(manifest.Evidence))
	fmt.Printf("[OK] Embedded standalone offline audit-viewer.html\n")
	fmt.Printf("[OK] Generated manifest.json with cryptographic digests\n")
	fmt.Printf("[OK] Bundle ready for air-gapped auditor handoff: %s\n", *outBundle)
}

// RunValidate provides fast standalone validation against official NIST OSCAL schemas.
func RunValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to OSCAL JSON artifact to validate")
	kindFlag := fs.String("kind", "", "Artifact kind (catalog, profile, ssp, component-definition, assessment-plan, assessment-results, poam, mapping)")
	checkIntegrity := fs.Bool("integrity", false, "Perform deep semantic reference and UUID integrity checks")
	_ = fs.Parse(args)

	if *filePath == "" && fs.NArg() > 0 {
		*filePath = fs.Arg(0)
		if *kindFlag == "" && fs.NArg() > 1 {
			*kindFlag = fs.Arg(1)
		}
	}

	if *filePath == "" {
		candidates := []string{
			"component-definition.json",
			filepath.Join(".xoscal", "component-definition.json"),
			"ssp.json",
			"system-security-plan.json",
			"catalog.json",
			"profile.json",
			"assessment-results.json",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				*filePath = c
				break
			}
		}
	}

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "error: -file is required (or supply file path as first argument)")
		fs.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	v, err := schemavalidate.NewValidator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading schemas: %v\n", err)
		os.Exit(1)
	}

	kind, err := ResolveKind(*kindFlag, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := v.Validate(data, kind); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %s failed OSCAL %s schema validation\n", kind.Name, schemavalidate.SchemaVersion)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	if *checkIntegrity {
		findings, err := integrity.Check(data, kind)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running integrity check: %v\n", err)
			os.Exit(1)
		}

		hasErrors := false
		for _, f := range findings {
			if f.Severity == integrity.SeverityError {
				hasErrors = true
				fmt.Fprintf(os.Stderr, "[INTEGRITY ERROR] %s: %s (%s)\n", f.Rule, f.Message, f.Path)
			} else {
				fmt.Fprintf(os.Stdout, "[INTEGRITY WARN]  %s: %s (%s)\n", f.Rule, f.Message, f.Path)
			}
		}

		if hasErrors {
			fmt.Fprintf(os.Stderr, "FAILED: %s failed semantic reference integrity checks\n", kind.Name)
			os.Exit(3)
		}
		fmt.Printf("VALID: %s passes official OSCAL %s schema and semantic integrity checks\n", kind.Name, schemavalidate.SchemaVersion)
		return
	}

	fmt.Printf("VALID: %s passes official OSCAL %s schema validation\n", kind.Name, schemavalidate.SchemaVersion)
}

// RunFrameworks lists, exports, compares, or verifies compliance frameworks.
func RunFrameworks(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal frameworks [list|export|compare|verify|seed] [options]")
		os.Exit(1)
	}

	sub := args[0]
	reg := frameworks.DefaultRegistry()

	switch sub {
	case "list":
		fs := flag.NewFlagSet("frameworks list", flag.ExitOnError)
		jurisdiction := fs.String("jurisdiction", "", "Optional filter by jurisdiction code (e.g. 'ca', 'us')")
		_ = fs.Parse(args[1:])

		var fws []frameworks.Framework
		if *jurisdiction != "" {
			fws = reg.FindByJurisdiction(*jurisdiction)
			fmt.Printf("\n=== AUTHORITATIVE CYBERSECURITY FRAMEWORKS (%s) ===\n", strings.ToUpper(*jurisdiction))
		} else {
			fws = reg.List()
			fmt.Printf("\n=== AUTHORITATIVE CYBERSECURITY FRAMEWORKS ===\n")
		}

		fmt.Printf("%-24s %-6s %-14s %-16s %s\n", "REF ID", "LOCALE", "CONTROLS", "VERSION", "NAME")
		fmt.Println(strings.Repeat("-", 96))
		for _, fw := range fws {
			profStr := ""
			if fw.HasProfile() {
				profStr = " (+ Profile)"
			}
			ctrlCountStr := fmt.Sprintf("%d%s", fw.ControlCount(), profStr)
			fmt.Printf("%-24s %-6s %-14s %-16s %s\n", fw.ID(), strings.ToUpper(fw.Jurisdiction()), ctrlCountStr, fw.Version(), fw.Title())
		}
		fmt.Println()

	case "export":
		fs := flag.NewFlagSet("frameworks export", flag.ExitOnError)
		outDir := fs.String("out", "data/frameworks", "Output destination directory")
		fwName := fs.String("framework", "", "Optional specific framework ref_id or alias")
		jurisdiction := fs.String("jurisdiction", "", "Optional filter by jurisdiction (e.g. 'ca')")
		_ = fs.Parse(args[1:])

		if *fwName != "" {
			if !reg.Has(*fwName) {
				fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
				os.Exit(1)
			}
			if err := reg.Export(*outDir, *fwName); err != nil {
				fmt.Fprintf(os.Stderr, "error exporting %s: %v\n", *fwName, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Exported %s -> %s\n", *fwName, *outDir)
			return
		}

		if *jurisdiction != "" {
			if err := reg.ExportJurisdiction(*outDir, *jurisdiction); err != nil {
				fmt.Fprintf(os.Stderr, "error exporting frameworks for jurisdiction %s: %v\n", *jurisdiction, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Exported frameworks for jurisdiction %s -> %s\n", *jurisdiction, *outDir)
			return
		}

		if err := reg.ExportAll(*outDir); err != nil {
			fmt.Fprintf(os.Stderr, "error exporting frameworks: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Exported all registered frameworks -> %s\n", *outDir)

	case "compare":
		RunCompare(args[1:])

	case "verify":
		RunVerifyParity(args[1:])

	case "seed":
		RunSeed(args[1:])

	default:
		fmt.Fprintf(os.Stderr, "unknown frameworks subcommand: %s (expected 'list', 'export', 'compare', 'verify', or 'seed')\n", sub)
		os.Exit(1)
	}
}

// RunVerifyParity verifies verbatim parity between framework definitions and created OSCAL artifacts.
func RunVerifyParity(args []string) {
	fs := flag.NewFlagSet("verify-parity", flag.ExitOnError)
	fwName := fs.String("framework", "", "Optional specific framework ref_id or alias (e.g. 'cccs-itsp-10-033')")
	dir := fs.String("dir", "data/frameworks", "Directory containing exported framework artifacts")
	artifact := fs.String("artifact", "", "Direct path to an OSCAL JSON artifact to verify")
	asJSON := fs.Bool("json", false, "Output results in JSON format")
	asMarkdown := fs.Bool("markdown", false, "Output results in Markdown format")
	verbose := fs.Bool("v", false, "Verbose output showing individual discrepancies")
	strict := fs.Bool("strict", true, "Exit with non-zero status if any deviation is detected")
	_ = fs.Parse(args)

	reg := frameworks.DefaultRegistry()

	var reports []*frameworks.ParityReport

	if *artifact != "" {
		if *fwName == "" {
			fmt.Fprintf(os.Stderr, "error: -framework must be specified when using -artifact\n")
			os.Exit(1)
		}
		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}
		rep, err := frameworks.VerifyArtifactParity(fw, *artifact)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error verifying parity: %v\n", err)
			os.Exit(1)
		}
		reports = append(reports, rep)
	} else if *fwName != "" {
		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}

		artifactNames := []string{
			"catalog.json",
			"profile.json",
			"component-definition.json",
			"system-security-plan.json",
			"assessment-plan.json",
			"assessment-results.json",
			"plan-of-action-and-milestones.json",
		}

		foundAny := false
		for _, name := range artifactNames {
			artPath := filepath.Join(*dir, fw.ID(), name)
			// #nosec G703,G304 -- artPath is resolved within the operator-selected directory.
			if _, err := os.Stat(artPath); err == nil {
				foundAny = true
				rep, err := frameworks.VerifyArtifactParity(fw, artPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error verifying %s %s: %v\n", fw.ID(), name, err)
					os.Exit(1)
				}
				reports = append(reports, rep)
			}
		}

		if !foundAny {
			catData, err := fw.CatalogJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error loading in-memory catalog for %s: %v\n", fw.ID(), err)
				os.Exit(1)
			}
			rep, err := frameworks.VerifyBytesParity(fw, catData, "memory://"+fw.ID()+"/catalog.json")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error verifying in-memory parity: %v\n", err)
				os.Exit(1)
			}
			reports = append(reports, rep)
		}
	} else {
		var err error
		reports, err = frameworks.VerifyAllFrameworksParity(reg, *dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error verifying frameworks parity: %v\n", err)
			os.Exit(1)
		}
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if len(reports) == 1 {
			_ = enc.Encode(reports[0])
		} else {
			_ = enc.Encode(reports)
		}
	} else if *asMarkdown {
		for _, r := range reports {
			fmt.Println(r.FormatMarkdown())
			fmt.Println()
		}
	} else {
		for _, r := range reports {
			fmt.Print(r.FormatTerminal(*verbose))
		}
	}

	hasDeviation := false
	for _, r := range reports {
		if r.Verdict != frameworks.VerdictVerbatimParity {
			hasDeviation = true
			break
		}
	}

	if hasDeviation && *strict {
		os.Exit(1)
	}
}

// RunSeed seeds the Knowledge Graph database with authoritative baseline frameworks.
func RunSeed(args []string) {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	dsn := fs.String("dsn", "oscal.db", "SQLite database path or DSN")
	dbAlias := fs.String("db", "", "Alias for -dsn")
	_ = fs.Parse(args)

	effectiveDSN := *dsn
	if *dbAlias != "" {
		effectiveDSN = *dbAlias
	}

	fmt.Printf("[INFO] Seeding Knowledge Graph at %s ...\n", effectiveDSN)
	store, err := kg.NewSQLiteStore(effectiveDSN, dbutil.PoolConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening KG store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := frameworks.DefaultRegistry().SeedKG(ctx, store); err != nil {
		fmt.Fprintf(os.Stderr, "error seeding baseline frameworks: %v\n", err)
		os.Exit(1)
	}

	fws, _ := store.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	reqs, _ := store.ListEntities(ctx, "reg:Requirement", kg.EntityStatusActive)
	fmt.Printf("[OK] Successfully seeded %d framework(s) and %d requirement(s) into %s\n", len(fws), len(reqs), effectiveDSN)
}

// RunGenerate generates OSCAL artifacts from the Knowledge Graph snapshot.
func RunGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	snapshot := fs.String("snapshot", "", "Snapshot name (required)")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	outDir := fs.String("out", "./oscal-out", "Output directory")
	arOnly := fs.Bool("assessment-results-only", false, "Emit only assessment-results")
	validate := fs.Bool("validate", false, "Schema-validate generated artifacts")
	_ = fs.Parse(args)

	if *snapshot == "" {
		fmt.Fprintln(os.Stderr, "error: -snapshot is required")
		fs.Usage()
		os.Exit(1)
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := os.MkdirAll(*outDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	var validator *schemavalidate.Validator
	if *validate {
		v, err := schemavalidate.NewValidator()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error initializing schema validator: %v\n", err)
			os.Exit(1)
		}
		validator = v
	}

	gen := oscal.NewGenerator(store)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, *snapshot, *framework)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating assessment results: %v\n", err)
			os.Exit(1)
		}
		path := filepath.Join(*outDir, "assessment-results.json")
		data, err := oscal.ExportAssessmentResultsJSON(ar)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting assessment results: %v\n", err)
			os.Exit(1)
		}
		if validator != nil {
			if err := validator.Validate(data, schemavalidate.KindAssessmentResults); err != nil {
				fmt.Fprintf(os.Stderr, "validation failed for %s: %v\n", path, err)
				os.Exit(1)
			}
		}
		// #nosec G703,G304,G306 -- path is beneath operator-selected output directory.
		if err := os.WriteFile(path, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Generated %s\n", path)
		return
	}

	res, err := gen.GenerateAllArtifacts(ctx, *snapshot, *framework, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating artifacts: %v\n", err)
		os.Exit(1)
	}

	type fileEntry struct {
		name string
		data []byte
		kind schemavalidate.ArtifactKind
	}
	var files []fileEntry

	if res.Catalog != nil {
		d, err := oscal.ExportCatalogJSON(res.Catalog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting catalog: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"catalog.json", d, schemavalidate.KindCatalog})
	}
	if res.Profile != nil {
		d, err := oscal.ExportProfileJSON(res.Profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting profile: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"profile.json", d, schemavalidate.KindProfile})
	}
	if res.ComponentDefinition != nil {
		d, err := oscal.ExportComponentDefinitionJSON(res.ComponentDefinition)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting component definition: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"component-definition.json", d, schemavalidate.KindComponentDefinition})
	}
	if res.SSP != nil {
		d, err := oscal.ExportSSPJSON(res.SSP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting SSP: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"system-security-plan.json", d, schemavalidate.KindSSP})
	}
	if res.AssessmentPlan != nil {
		d, err := oscal.ExportAssessmentPlanJSON(res.AssessmentPlan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting assessment plan: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"assessment-plan.json", d, schemavalidate.KindAssessmentPlan})
	}
	if res.AssessmentResults != nil {
		d, err := oscal.ExportAssessmentResultsJSON(res.AssessmentResults)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting assessment results: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"assessment-results.json", d, schemavalidate.KindAssessmentResults})
	}
	if res.POAM != nil {
		d, err := oscal.ExportPOAMJSON(res.POAM)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting POAM: %v\n", err)
			os.Exit(1)
		}
		files = append(files, fileEntry{"plan-of-action-and-milestones.json", d, schemavalidate.KindPOAM})
	}

	for _, fe := range files {
		path := filepath.Join(*outDir, fe.name)
		if validator != nil {
			if err := validator.Validate(fe.data, fe.kind); err != nil {
				fmt.Fprintf(os.Stderr, "validation failed for %s: %v\n", path, err)
				os.Exit(1)
			}
		}
		// #nosec G703,G304,G306 -- path is beneath operator-selected output directory.
		if err := os.WriteFile(path, fe.data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Generated %s\n", path)
	}
}

// RunIngest ingests raw requirements JSON into the Knowledge Graph database.
func RunIngest(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	input := fs.String("input", "", "Input JSON file path (required)")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	_ = fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: -input is required")
		fs.Usage()
		os.Exit(1)
	}

	raw, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	rec := reconciler.NewReconciler(store)
	pipeline := ingestion.NewPipeline(
		&ingestion.EUAIActParser{},
		ingestion.NewNormalizer(*framework),
		rec,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := pipeline.Run(ctx, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running ingestion pipeline: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Ingested %d requirements, %d entities, %d conflicts into %s\n",
		len(res.Requirements), len(res.Entities), len(res.Conflicts), *dsn)
	for _, c := range res.Conflicts {
		fmt.Printf("  Conflict: %s (%s)\n", c.Description, c.Type)
	}
}

// RunControls provides interactive search, retrieval, and family listing for compliance controls.
func RunControls(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal controls [search|get|list] [options]")
		os.Exit(1)
	}

	sub := args[0]
	reg := frameworks.DefaultRegistry()

	switch sub {
	case "search":
		fs := flag.NewFlagSet("controls search", flag.ExitOnError)
		fwName := fs.String("framework", "cccs-itsp-10-033", "Framework identifier")
		limit := fs.Int("limit", 25, "Maximum results to display")
		_ = fs.Parse(args[1:])

		query := ""
		if fs.NArg() > 0 {
			query = strings.Join(fs.Args(), " ")
		}

		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}

		ctrls, err := frameworks.LoadControls(fw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading controls: %v\n", err)
			os.Exit(1)
		}

		matches := frameworks.SearchControls(ctrls, query, *limit)
		fmt.Printf("\n=== SEARCH RESULTS IN %s (Query: '%s', Matches: %d) ===\n", fw.ID(), query, len(matches))
		fmt.Printf("%-14s %-8s %-14s %s\n", "CONTROL ID", "FAMILY", "SOVEREIGN", "TITLE")
		fmt.Println(strings.Repeat("-", 80))
		for _, c := range matches {
			sovereign := "Standard"
			if c.IsCanadian {
				sovereign = "CA SOVEREIGN"
			}
			fmt.Printf("%-14s %-8s %-14s %s\n", strings.ToUpper(c.ID), c.FamilyID, sovereign, c.Title)
		}
		fmt.Println()

	case "get":
		fs := flag.NewFlagSet("controls get", flag.ExitOnError)
		fwName := fs.String("framework", "cccs-itsp-10-033", "Framework identifier")
		_ = fs.Parse(args[1:])

		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "error: control ID is required (e.g. 'xoscal controls get ac-2')")
			os.Exit(1)
		}
		ctrlID := fs.Arg(0)

		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}

		ctrls, err := frameworks.LoadControls(fw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading controls: %v\n", err)
			os.Exit(1)
		}

		c, found := frameworks.GetControl(ctrls, ctrlID)
		if !found {
			fmt.Fprintf(os.Stderr, "error: control '%s' not found in framework %s\n", ctrlID, *fwName)
			os.Exit(1)
		}

		sovereignTag := ""
		if c.IsCanadian {
			sovereignTag = " [CANADIAN SOVEREIGN REQUIREMENT]"
		}

		fmt.Printf("\n================================================================================\n")
		fmt.Printf("CONTROL %s: %s%s\n", strings.ToUpper(c.ID), c.Title, sovereignTag)
		fmt.Printf("Framework: %s | Family: %s (%s)\n", fw.Title(), c.FamilyID, c.FamilyTitle)
		fmt.Printf("================================================================================\n")

		if len(c.Statements) > 0 {
			fmt.Printf("\nCONTROL STATEMENT:\n")
			for _, stmt := range c.Statements {
				fmt.Printf("  %s\n", stmt)
			}
		}

		if c.CanadianGuidance != "" {
			fmt.Printf("\nCANADIAN SOVEREIGN GUIDANCE (GC DISCUSSION):\n")
			fmt.Printf("  %s\n", c.CanadianGuidance)
		} else if c.Guidance != "" {
			fmt.Printf("\nDISCUSSION & GUIDANCE:\n")
			fmt.Printf("  %s\n", c.Guidance)
		}

		if len(c.Parameters) > 0 {
			fmt.Printf("\nORGANIZATION-DEFINED PARAMETERS (ODP):\n")
			for k, v := range c.Parameters {
				fmt.Printf("  * %-20s: %s\n", k, v)
			}
		}

		if len(c.Enhancements) > 0 {
			fmt.Printf("\nCONTROL ENHANCEMENTS (%d):\n", len(c.Enhancements))
			fmt.Printf("  %s\n", strings.Join(c.Enhancements, ", "))
		}
		fmt.Println()

	case "list":
		fs := flag.NewFlagSet("controls list", flag.ExitOnError)
		fwName := fs.String("framework", "cccs-itsp-10-033", "Framework identifier")
		fam := fs.String("family", "", "Optional filter by family code (e.g. AC, IA, SC)")
		_ = fs.Parse(args[1:])

		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}

		ctrls, err := frameworks.LoadControls(fw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading controls: %v\n", err)
			os.Exit(1)
		}

		var filtered []frameworks.ControlInfo
		famUpper := strings.ToUpper(*fam)
		for _, c := range ctrls {
			if famUpper == "" || c.FamilyID == famUpper {
				filtered = append(filtered, c)
			}
		}

		famHeader := ""
		if famUpper != "" {
			famHeader = fmt.Sprintf(" (Family: %s)", famUpper)
		}
		fmt.Printf("\n=== CONTROLS IN %s%s (%d Controls) ===\n", fw.ID(), famHeader, len(filtered))
		fmt.Printf("%-14s %-8s %-14s %s\n", "CONTROL ID", "FAMILY", "SOVEREIGN", "TITLE")
		fmt.Println(strings.Repeat("-", 80))
		for _, c := range filtered {
			sovereign := "Standard"
			if c.IsCanadian {
				sovereign = "CA SOVEREIGN"
			}
			fmt.Printf("%-14s %-8s %-14s %s\n", strings.ToUpper(c.ID), c.FamilyID, sovereign, c.Title)
		}
		fmt.Println()

	default:
		fmt.Fprintf(os.Stderr, "unknown controls subcommand: %s (expected 'search', 'get', or 'list')\n", sub)
		os.Exit(1)
	}
}

// RunCompare computes transition delta between two compliance frameworks.
func RunCompare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	fromFlag := fs.String("from", "", "Base framework (e.g. cccs-itsg-33)")
	toFlag := fs.String("to", "", "Target framework (e.g. cccs-itsp-10-033)")
	markdown := fs.Bool("markdown", false, "Output Markdown transition report")
	_ = fs.Parse(args)

	fromID := *fromFlag
	toID := *toFlag

	if fromID == "" && fs.NArg() > 0 {
		fromID = fs.Arg(0)
	}
	if toID == "" && fs.NArg() > 1 {
		toID = fs.Arg(1)
	}

	if fromID == "" || toID == "" {
		fmt.Fprintln(os.Stderr, "Usage: xoscal compare <from-framework> <to-framework> [options]")
		fmt.Fprintln(os.Stderr, "Example: xoscal compare cccs-itsg-33 cccs-itsp-10-033")
		os.Exit(1)
	}

	reg := frameworks.DefaultRegistry()
	fromFw, ok := reg.Get(fromID)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: framework '%s' not registered\n", fromID)
		os.Exit(1)
	}

	toFw, ok := reg.Get(toID)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: framework '%s' not registered\n", toID)
		os.Exit(1)
	}

	comp, err := frameworks.CompareFrameworks(fromFw, toFw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error comparing frameworks: %v\n", err)
		os.Exit(1)
	}

	if *markdown {
		fmt.Print(comp.FormatMarkdown())
	} else {
		fmt.Print(comp.FormatTerminal())
	}
}

// RunServe starts the local zero-dependency Web Studio server.
func RunServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := fs.String("host", "127.0.0.1", "Host address to bind")
	port := fs.Int("port", 8080, "Port to listen on")
	openBrowser := fs.Bool("open", false, "Automatically open in default web browser")
	_ = fs.Parse(args)

	srv, err := webstudio.NewServer(webstudio.Config{
		Host:     *host,
		Port:     *port,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing web studio: %v\n", err)
		os.Exit(1)
	}

	addr, err := srv.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting web studio: %v\n", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("\n=== xOSCAL Dev Studio & Sovereign Framework Explorer ===\n")
	fmt.Printf("Listening at: %s\n", url)
	fmt.Printf("Press Ctrl+C to stop.\n\n")

	if *openBrowser {
		_ = webstudio.OpenBrowser(url)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Printf("\nShutting down xOSCAL Dev Studio ...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}

// RunCompletion generates shell autocompletion scripts for bash, zsh, or fish.
func RunCompletion(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal completion [bash|zsh|fish]")
		os.Exit(1)
	}

	sh := args[0]
	binName := filepath.Base(os.Args[0])
	if binName == "" {
		binName = "xoscal"
	}

	if err := completion.Generate(sh, binName, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error generating %s completion: %v\n", sh, err)
		os.Exit(1)
	}
}

// RunQuickstart guides developers through onboarding and key Day-2 operations.
func RunQuickstart(args []string) {
	fmt.Printf("\n=== xOSCAL DEVELOPER ONBOARDING QUICKSTART ===\n\n")
	fmt.Printf("1. Checking Environment & Pinned NIST OSCAL Schemas ...\n")
	fmt.Printf("   [OK] Embedded Schema Version: %s\n", schemavalidate.SchemaVersion)
	fmt.Printf("   [OK] Registered Sovereign Frameworks: %d\n", len(frameworks.DefaultRegistry().List()))

	fmt.Printf("\n2. Scanning Current Working Directory ...\n")
	scan, err := scaffold.ScanRepository(".")
	if err != nil {
		fmt.Printf("   [WARN] Could not scan current dir: %v\n", err)
	} else {
		fmt.Printf("   [OK] Detected Project: %s (Languages: %v, Docker: %v, K8s: %v, TF: %v)\n",
			scan.Name, scan.Languages, scan.HasDocker, scan.HasK8s, scan.HasTF)
		fmt.Printf("   [OK] Ready to scaffold: %d detected component(s)\n", len(scan.Components))
		fmt.Printf("   Run 'xoscal init' to scaffold component-definition.json and .xoscal workspace config.\n")
	}

	fmt.Printf("\n3. Recommended Developer Commands:\n")
	fmt.Printf("   * Search Sovereign Controls:   xoscal controls search \"cryptography\"\n")
	fmt.Printf("   * Inspect Control Statement:   xoscal controls get ac-2\n")
	fmt.Printf("   * Compare Framework Versions:  xoscal compare cccs-itsg-33 cccs-itsp-10-033\n")
	fmt.Printf("   * Launch Local Web Explorer:   xoscal serve --open\n")
	fmt.Printf("   * Verify Vector State:         xoscal verify\n")
	fmt.Printf("   * Generate Shell Completion:   source <(xoscal completion zsh)\n\n")
}

// ResolveKind resolves the target OSCAL artifact kind from a flag or payload inspection.
func ResolveKind(kindFlag string, data []byte) (schemavalidate.ArtifactKind, error) {
	if kindFlag != "" {
		return KindFromName(kindFlag)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return schemavalidate.ArtifactKind{}, fmt.Errorf("unmarshal JSON: %w", err)
	}

	for _, k := range []schemavalidate.ArtifactKind{
		schemavalidate.KindCatalog,
		schemavalidate.KindProfile,
		schemavalidate.KindSSP,
		schemavalidate.KindComponentDefinition,
		schemavalidate.KindAssessmentPlan,
		schemavalidate.KindAssessmentResults,
		schemavalidate.KindPOAM,
		schemavalidate.KindMapping,
	} {
		if _, ok := doc[k.RootKey]; ok {
			return k, nil
		}
	}

	return schemavalidate.ArtifactKind{}, fmt.Errorf("could not auto-detect artifact kind; specify -kind explicitly")
}

// KindFromName resolves the artifact kind from a user-supplied name.
func KindFromName(name string) (schemavalidate.ArtifactKind, error) {
	switch strings.ToLower(name) {
	case "catalog":
		return schemavalidate.KindCatalog, nil
	case "profile":
		return schemavalidate.KindProfile, nil
	case "ssp", "system-security-plan":
		return schemavalidate.KindSSP, nil
	case "component-definition", "component":
		return schemavalidate.KindComponentDefinition, nil
	case "assessment-plan", "ap":
		return schemavalidate.KindAssessmentPlan, nil
	case "assessment-results", "ar":
		return schemavalidate.KindAssessmentResults, nil
	case "poam", "plan-of-action-and-milestones":
		return schemavalidate.KindPOAM, nil
	case "mapping", "mapping-collection":
		return schemavalidate.KindMapping, nil
	default:
		return schemavalidate.ArtifactKind{}, fmt.Errorf("unknown kind: %s", name)
	}
}
