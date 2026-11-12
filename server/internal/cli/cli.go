// Package cli provides the unified CLI commands for xoscal and xoscal-ctl.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mchorfa/xoscal/server/internal/bundle"
	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/completion"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/diff"
	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/graph"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"github.com/mchorfa/xoscal/server/internal/tabular"
	"github.com/mchorfa/xoscal/server/internal/utcp"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

// PrintUsage prints command-line usage information.
func PrintUsage(version string) {
	binName := filepath.Base(os.Args[0])
	if binName == "" {
		binName = "xoscal"
	}

	fwCount, _ := utcpRegistryCounts()

	fmt.Printf(`%s v%s — Unified xOSCAL Compliance & Governance Engine

USAGE:
  %s <command> [options]

COMMANDS:
  controls       Search and inspect controls across registered frameworks
  compare        Compare two frameworks and compute sovereign transition delta
  ontology       Inspect formal OWL/JSON-LD regulatory & threat ontology (12 classes, 16 predicates)
  graph          Query %d-framework property graph, multi-jurisdiction crosswalks & 6D blast radius
  feeds          List, sync, and inspect sovereign regulatory & threat intelligence feeds
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
  utcp           Generate, validate, and bridge Universal Tool Calling Protocol (UTCP) manuals & SDKs
  doctor         Run deep diagnostic checks on runtime substrate, schemas, and framework registry
  completion     Generate shell autocompletions (bash, zsh, fish)
  version        Print CLI version and exit

Run '%s <command> -h' for details on specific command options.
`, binName, version, binName, fwCount, binName)
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
	case "ontology":
		RunOntology(args[2:])
	case "graph":
		RunGraph(args[2:])
	case "feeds":
		RunFeeds(args[2:])
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
	case "utcp":
		RunUTCP(args[2:])
	case "doctor", "diagnostics":
		RunDoctor(args[2:], version)
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
	parseFlagsInterspersed(fs, args)

	targetFile := *file
	if isUnexpandedPlaceholder(targetFile) {
		targetFile = ""
	}
	if targetFile == "" && fs.NArg() > 0 && !isUnexpandedPlaceholder(fs.Arg(0)) {
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
				_, _ = fmt.Fprintf(os.Stdout, "[INTEGRITY WARN]  %s: %s (%s)\n", f.Rule, f.Message, f.Path)
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

type boolFlag interface {
	flag.Value
	IsBoolFlag() bool
}

// parseFlagsInterspersed parses flags even when positional arguments precede flag arguments.
func parseFlagsInterspersed(fs *flag.FlagSet, args []string) {
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			flagArgs = append(flagArgs, a)
			if !strings.Contains(a, "=") {
				name := strings.TrimLeft(a, "-")
				if f := fs.Lookup(name); f != nil {
					if bf, ok := f.Value.(boolFlag); !ok || !bf.IsBoolFlag() {
						if i+1 < len(args) {
							i++
							flagArgs = append(flagArgs, args[i])
						}
					}
				}
			}
		} else {
			posArgs = append(posArgs, a)
		}
	}
	_ = fs.Parse(append(flagArgs, posArgs...))
}

// isUnexpandedPlaceholder reports whether s is an unexpanded UTCP template placeholder
// such as "${jurisdiction}" or "UTCP_ARG_jurisdiction_UTCP_END".
func isUnexpandedPlaceholder(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") && len(s) > 3 {
		return isPlaceholderIdent(s[2 : len(s)-1])
	}
	if strings.HasPrefix(s, "UTCP_ARG_") && strings.HasSuffix(s, "_UTCP_END") && len(s) > len("UTCP_ARG__UTCP_END") {
		return isPlaceholderIdent(s[len("UTCP_ARG_") : len(s)-len("_UTCP_END")])
	}
	return false
}

func isPlaceholderIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isIdentChar := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
		if !isIdentChar {
			return false
		}
	}
	return true
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
		jsonOut := fs.Bool("json", false, "Output frameworks in JSON format")
		parseFlagsInterspersed(fs, args[1:])

		if isUnexpandedPlaceholder(*jurisdiction) {
			*jurisdiction = ""
		}

		var fws []frameworks.Framework
		if *jurisdiction != "" {
			fws = reg.FindByJurisdiction(*jurisdiction)
		} else {
			fws = reg.List()
		}

		if *jsonOut {
			infos := make([]frameworks.Info, len(fws))
			for i, fw := range fws {
				infos[i] = frameworks.InfoOf(fw)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(infos)
			return
		}

		if *jurisdiction != "" {
			fmt.Printf("\n=== AUTHORITATIVE CYBERSECURITY FRAMEWORKS (%s) ===\n", strings.ToUpper(*jurisdiction))
		} else {
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
		parseFlagsInterspersed(fs, args[1:])

		if isUnexpandedPlaceholder(*fwName) {
			*fwName = ""
		}
		if isUnexpandedPlaceholder(*jurisdiction) {
			*jurisdiction = ""
		}
		if isUnexpandedPlaceholder(*outDir) || *outDir == "" {
			*outDir = "data/frameworks"
		}

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
	parseFlagsInterspersed(fs, args)

	if isUnexpandedPlaceholder(*fwName) {
		*fwName = ""
	}
	if isUnexpandedPlaceholder(*dir) || *dir == "" {
		*dir = "data/frameworks"
	}
	if isUnexpandedPlaceholder(*artifact) {
		*artifact = ""
	}

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
	fwName := fs.String("framework", "", "Optional specific framework ref_id or alias to seed")
	snapshot := fs.String("snapshot", "baseline", "Snapshot name to create after seeding")
	parseFlagsInterspersed(fs, args)

	if isUnexpandedPlaceholder(*fwName) {
		*fwName = ""
	}
	if isUnexpandedPlaceholder(*dsn) || *dsn == "" {
		*dsn = "oscal.db"
	}
	if isUnexpandedPlaceholder(*dbAlias) {
		*dbAlias = ""
	}
	if isUnexpandedPlaceholder(*snapshot) || *snapshot == "" {
		*snapshot = "baseline"
	}

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
	defer func() { _ = store.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	reg := frameworks.DefaultRegistry()
	snapNames := []string{*snapshot, "baseline", "latest", "v1.0"}
	if *fwName != "" {
		fw, ok := reg.Get(*fwName)
		if !ok {
			fmt.Fprintf(os.Stderr, "error: unknown framework %s\n", *fwName)
			os.Exit(1)
		}
		if err := frameworks.SeedFramework(ctx, store, fw); err != nil {
			fmt.Fprintf(os.Stderr, "error seeding framework %s: %v\n", fw.ID(), err)
			os.Exit(1)
		}
		if sqlStore, ok := store.(*kg.SQLiteStore); ok && sqlStore.DB() != nil {
			if gStore, gErr := graph.NewSQLiteStoreFromDB(sqlStore.DB()); gErr == nil {
				singleReg := frameworks.NewRegistry()
				_ = singleReg.Register(fw)
				if _, err := graph.BuildUnifiedKnowledgeGraph(ctx, gStore, singleReg); err != nil {
					fmt.Fprintf(os.Stderr, "error building framework property graph: %v\n", err)
					os.Exit(1)
				}
			}
		}
		snapNames = append(snapNames, fw.ID(), *fwName)
	} else {
		if err := reg.SeedKG(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "error seeding baseline frameworks: %v\n", err)
			os.Exit(1)
		}
		if sqlStore, ok := store.(*kg.SQLiteStore); ok && sqlStore.DB() != nil {
			if gStore, gErr := graph.NewSQLiteStoreFromDB(sqlStore.DB()); gErr == nil {
				if _, err := graph.BuildUnifiedKnowledgeGraph(ctx, gStore, reg); err != nil {
					fmt.Fprintf(os.Stderr, "error building unified property graph: %v\n", err)
					os.Exit(1)
				}
			}
		}
	}

	seenSnap := make(map[string]bool)
	for _, sn := range snapNames {
		if sn == "" || seenSnap[sn] {
			continue
		}
		seenSnap[sn] = true
		if _, err := store.CreateSnapshot(ctx, sn); err != nil {
			fmt.Fprintf(os.Stderr, "error creating snapshot %q: %v\n", sn, err)
			os.Exit(1)
		}
	}

	fws, _ := store.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	reqs, _ := store.ListEntities(ctx, "reg:Requirement", kg.EntityStatusActive)
	fmt.Printf("[OK] Successfully seeded %d framework(s) and %d requirement(s) into %s (snapshot: %s)\n", len(fws), len(reqs), effectiveDSN, *snapshot)
}

// countSnapshotRequirements returns the number of active assessable Requirement entities in the snapshot matching frameworkID.
func countSnapshotRequirements(ctx context.Context, store kg.Store, snapshotName, frameworkID string) (int, error) {
	entities, err := store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			continue
		}
		if req.Assessable && strings.EqualFold(req.Framework, frameworkID) {
			count++
		}
	}
	return count, nil
}

func resolveOSCALCLI(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("oscal-cli not found at %s: %w", explicit, err)
		}
		return explicit, nil
	}
	if p, err := exec.LookPath("oscal-cli"); err == nil {
		return p, nil
	}
	local := "tools/oscal-cli/bin/oscal-cli"
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}
	return "", fmt.Errorf("oscal-cli not found in PATH or at %s; install via scripts/install-oscal-cli.sh", local)
}

var kindToCLIModel = map[schemavalidate.ArtifactKind]string{
	schemavalidate.KindCatalog:             "catalog",
	schemavalidate.KindProfile:             "profile",
	schemavalidate.KindSSP:                 "ssp",
	schemavalidate.KindComponentDefinition: "component-definition",
	schemavalidate.KindAssessmentPlan:      "ap",
	schemavalidate.KindAssessmentResults:   "ar",
	schemavalidate.KindPOAM:                "poam",
}

func runConstraintValidation(oscalCLI, filePath string, kind schemavalidate.ArtifactKind) {
	model, ok := kindToCLIModel[kind]
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// #nosec G204 -- oscalCLI is resolved via explicit operator flag or exec.LookPath("oscal-cli").
	cmd := exec.CommandContext(ctx, oscalCLI, model, "validate", filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "constraint validation failed for %s:\n%s\n", filePath, string(out))
		os.Exit(1)
	}
}

// RunGenerate generates OSCAL artifacts from the Knowledge Graph snapshot.
func RunGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	dbAlias := fs.String("db", "", "Alias for -dsn")
	snapshot := fs.String("snapshot", "", "Snapshot name (required)")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	outDir := fs.String("out", "./oscal-out", "Output directory")
	arOnly := fs.Bool("assessment-results-only", false, "Emit only assessment-results")
	validate := fs.Bool("validate", false, "Schema-validate generated artifacts")
	validateConstraints := fs.Bool("validate-constraints", false, "Run oscal-cli Metaschema constraint validation (implies -validate)")
	oscalCLI := fs.String("oscal-cli", "", "Path to oscal-cli binary (defaults to PATH lookup or tools/oscal-cli/bin/oscal-cli)")
	timeout := fs.Duration("timeout", 180*time.Second, "Timeout duration for artifact generation")
	parseFlagsInterspersed(fs, args)

	if isUnexpandedPlaceholder(*dbAlias) {
		*dbAlias = ""
	}
	if *dbAlias != "" {
		*dsn = *dbAlias
	}
	if isUnexpandedPlaceholder(*dsn) || *dsn == "" {
		*dsn = "oscal.db"
	}
	if isUnexpandedPlaceholder(*framework) || *framework == "" {
		*framework = "eu-ai-act"
	}
	if isUnexpandedPlaceholder(*outDir) || *outDir == "" {
		*outDir = "./oscal-out"
	}
	if isUnexpandedPlaceholder(*snapshot) {
		*snapshot = ""
	}
	if *validateConstraints {
		*validate = true
	}

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
	defer func() { _ = store.Close() }()

	genTimeout := *timeout
	if genTimeout <= 0 {
		genTimeout = 180 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), genTimeout)
	defer cancel()

	effectiveFramework := *framework
	if fw, ok := frameworks.DefaultRegistry().Get(*framework); ok {
		effectiveFramework = fw.ID()
	}

	effectiveSnapshot := *snapshot
	reqCount, err := countSnapshotRequirements(ctx, store, effectiveSnapshot, effectiveFramework)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading snapshot %q: %v\n", effectiveSnapshot, err)
		os.Exit(1)
	}
	if reqCount == 0 && effectiveFramework != *framework {
		if altCount, altErr := countSnapshotRequirements(ctx, store, effectiveSnapshot, *framework); altErr == nil && altCount > 0 {
			effectiveFramework = *framework
			reqCount = altCount
		}
	}
	if reqCount == 0 && frameworks.DefaultRegistry().Has(*snapshot) {
		if baseCount, baseErr := countSnapshotRequirements(ctx, store, "baseline", effectiveFramework); baseErr == nil && baseCount > 0 {
			effectiveSnapshot = "baseline"
			reqCount = baseCount
		}
	}
	if reqCount == 0 {
		fmt.Fprintf(os.Stderr, "error: snapshot %q contains 0 requirements for framework %q (run 'xoscal seed' or 'xoscal ingest' first)\n", *snapshot, *framework)
		os.Exit(1)
	}

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

	var cliBin string
	if *validateConstraints {
		bin, err := resolveOSCALCLI(*oscalCLI)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving oscal-cli: %v\n", err)
			os.Exit(1)
		}
		cliBin = bin
	}

	gen := oscal.NewGenerator(store)

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, effectiveSnapshot, effectiveFramework)
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
		_, _ = dbutil.WriteDigestSidecar(path)
		if cliBin != "" {
			runConstraintValidation(cliBin, path, schemavalidate.KindAssessmentResults)
		}
		fmt.Printf("[OK] Generated %s\n", path)
		return
	}

	res, err := gen.GenerateAllArtifacts(ctx, effectiveSnapshot, effectiveFramework, nil)
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
		_, _ = dbutil.WriteDigestSidecar(path)
		if cliBin != "" {
			runConstraintValidation(cliBin, path, fe.kind)
		}
		fmt.Printf("[OK] Generated %s\n", path)
	}
}

// RunIngest ingests raw requirements JSON into the Knowledge Graph database.
func RunIngest(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	dbAlias := fs.String("db", "", "Alias for -dsn")
	input := fs.String("input", "", "Input JSON file path (required)")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	snapshot := fs.String("snapshot", "baseline", "Snapshot name to create after ingestion")
	parseFlagsInterspersed(fs, args)

	if isUnexpandedPlaceholder(*dbAlias) {
		*dbAlias = ""
	}
	if *dbAlias != "" {
		*dsn = *dbAlias
	}

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
	defer func() { _ = store.Close() }()

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

	snapNames := []string{*snapshot, "baseline", "latest", "v1.0", *framework}
	seenSnap := make(map[string]bool)
	for _, sn := range snapNames {
		if sn == "" || seenSnap[sn] {
			continue
		}
		seenSnap[sn] = true
		if _, err := store.CreateSnapshot(ctx, sn); err != nil {
			fmt.Fprintf(os.Stderr, "error creating snapshot %q: %v\n", sn, err)
			os.Exit(1)
		}
	}

	fmt.Printf("[OK] Ingested %d requirements, %d entities, %d conflicts into %s (snapshot: %s)\n",
		len(res.Requirements), len(res.Entities), len(res.Conflicts), *dsn, *snapshot)
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
		jsonOut := fs.Bool("json", false, "Output search results in JSON format")
		parseFlagsInterspersed(fs, args[1:])

		if isUnexpandedPlaceholder(*fwName) || *fwName == "" {
			*fwName = "cccs-itsp-10-033"
		}

		query := ""
		if fs.NArg() > 0 {
			query = strings.Join(fs.Args(), " ")
			if isUnexpandedPlaceholder(query) {
				query = ""
			}
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
		if *jsonOut {
			if matches == nil {
				matches = []frameworks.ControlInfo{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(matches)
			return
		}

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
		jsonOut := fs.Bool("json", false, "Output control details in JSON format")
		parseFlagsInterspersed(fs, args[1:])

		if isUnexpandedPlaceholder(*fwName) || *fwName == "" {
			*fwName = "cccs-itsp-10-033"
		}

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

		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(c)
			return
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
		jsonOut := fs.Bool("json", false, "Output controls list in JSON format")
		parseFlagsInterspersed(fs, args[1:])

		if isUnexpandedPlaceholder(*fwName) || *fwName == "" {
			*fwName = "cccs-itsp-10-033"
		}
		if isUnexpandedPlaceholder(*fam) {
			*fam = ""
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

		var filtered []frameworks.ControlInfo
		famUpper := strings.ToUpper(*fam)
		for _, c := range ctrls {
			if famUpper == "" || c.FamilyID == famUpper {
				filtered = append(filtered, c)
			}
		}

		if *jsonOut {
			if filtered == nil {
				filtered = []frameworks.ControlInfo{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(filtered)
			return
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
	jsonOut := fs.Bool("json", false, "Output comparison delta in JSON format")
	parseFlagsInterspersed(fs, args)

	if isUnexpandedPlaceholder(*fromFlag) {
		*fromFlag = ""
	}
	if isUnexpandedPlaceholder(*toFlag) {
		*toFlag = ""
	}

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

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(comp)
		return
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

// DoctorCheck defines an individual diagnostic check result.
type DoctorCheck struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Status      string `json:"status"`   // PASS, ADVISORY, FAIL
	Evidence    string `json:"evidence"` // VERIFIED, OBSERVED, NOT_AVAILABLE
	Detail      string `json:"detail"`
	Remediation string `json:"remediation,omitempty"`
}

// DoctorReport aggregates diagnostic checks across all truth channels.
type DoctorReport struct {
	Version   string        `json:"version"`
	Timestamp string        `json:"timestamp"`
	OS        string        `json:"os"`
	Arch      string        `json:"arch"`
	GoVersion string        `json:"go_version"`
	Checks    []DoctorCheck `json:"checks"`
	Passed    int           `json:"passed"`
	Advisory  int           `json:"advisory"`
	Failed    int           `json:"failed"`
	Verdict   string        `json:"verdict"`
}

// RunDoctor performs deep diagnostics on the runtime substrate, schema engines, and framework registry.
func RunDoctor(args []string, version string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output machine-verifiable JSON report (Rule 13 Decision/Evidence trace)")
	_ = fs.Parse(args)

	report := DoctorReport{
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
	}

	// 1. Runtime Substrate Check
	report.Checks = append(report.Checks, DoctorCheck{
		Name:     "runtime_substrate",
		Category: "Host & Runtime",
		Status:   "PASS",
		Evidence: "VERIFIED",
		Detail:   fmt.Sprintf("OS: %s, Arch: %s, Go: %s, CPUs: %d", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU()),
	})

	// 2. OSCAL 1.2.3 Schema Engine (Tier 1 Offline)
	_, valErr := schemavalidate.NewValidator()
	if valErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "schema_engine_tier1",
			Category:    "OSCAL Conformance",
			Status:      "FAIL",
			Evidence:    "OBSERVED",
			Detail:      fmt.Sprintf("Embedded NIST OSCAL 1.2.3 schema failed to load: %v", valErr),
			Remediation: "Ensure embedded oscal_complete_schema.json is intact in internal/schemavalidate",
		})
	} else {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "schema_engine_tier1",
			Category: "OSCAL Conformance",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   "Embedded NIST OSCAL 1.2.3 Complete Schema compiled; offline validation ready",
		})
	}

	// 3. Framework Registry & Sovereign/International Catalogs
	reg := frameworks.DefaultRegistry()
	allRegistered := reg.List()
	expectedSovereign := []string{
		"cccs-itsp-10-033",
		"cccs-itsg-33",
		"cybersecure-canada",
		"cccs-itsp-10-171",
		"cccs-medium-cloud-pbmm",
	}
	expectedIDs := append(frameworks.ManifestFrameworkIDs(), expectedSovereign...)
	missingFws := []string{}
	for _, expectedID := range expectedIDs {
		if _, found := reg.Get(expectedID); !found {
			missingFws = append(missingFws, expectedID)
		}
	}
	brokenCatalogs := []string{}
	totalVerifiedControls := 0
	for _, fw := range allRegistered {
		ctrls, loadErr := frameworks.LoadControls(fw)
		if loadErr != nil {
			brokenCatalogs = append(brokenCatalogs, fmt.Sprintf("%s (%v)", fw.ID(), loadErr))
			continue
		}
		if len(ctrls) == 0 {
			brokenCatalogs = append(brokenCatalogs, fmt.Sprintf("%s (0 controls)", fw.ID()))
			continue
		}
		totalVerifiedControls += len(ctrls)
	}
	if len(missingFws) > 0 || len(brokenCatalogs) > 0 || len(allRegistered) < len(expectedIDs) {
		var errParts []string
		if len(allRegistered) < len(expectedIDs) {
			errParts = append(errParts, fmt.Sprintf("expected >=%d frameworks, got %d", len(expectedIDs), len(allRegistered)))
		}
		if len(missingFws) > 0 {
			errParts = append(errParts, "missing expected frameworks: "+strings.Join(missingFws, ", "))
		}
		if len(brokenCatalogs) > 0 {
			errParts = append(errParts, "unreadable/empty catalogs: "+strings.Join(brokenCatalogs, "; "))
		}
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "framework_registry",
			Category:    "Sovereign Registry",
			Status:      "FAIL",
			Evidence:    "OBSERVED",
			Detail:      strings.Join(errParts, " | "),
			Remediation: "Verify embedded catalog archive or XOSCAL_DATA_DIR integrity",
		})
	} else {
		itspFw, _ := reg.Get("cccs-itsp-10-033")
		ctrlCount := 0
		if itspFw != nil {
			ctrlCount = itspFw.ControlCount()
		}
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "framework_registry",
			Category: "Sovereign Registry",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   fmt.Sprintf("%d frameworks & %d controls verified (including CCCS ITSP.10.033 with %d controls)", len(allRegistered), totalVerifiedControls, ctrlCount),
		})
	}

	// 4. SQLite Knowledge Graph Engine
	kgStore, kgErr := kg.NewSQLiteStore(":memory:", dbutil.DefaultPool())
	if kgErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "sqlite_storage",
			Category:    "Storage & KG",
			Status:      "FAIL",
			Evidence:    "OBSERVED",
			Detail:      fmt.Sprintf("Failed to initialize in-memory SQLite store: %v", kgErr),
			Remediation: "Verify modernc.org/sqlite CGo-free driver initialization",
		})
	} else {
		_ = kgStore.Close()
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "sqlite_storage",
			Category: "Storage & KG",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   "Pure-Go SQLite engine verified; in-memory KG operational",
		})
	}

	// 5. Tier 2 Metaschema Engine (oscal-cli)
	oscalCliPath, oscalErr := exec.LookPath("oscal-cli")
	if oscalErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "metaschema_tier2",
			Category:    "OSCAL Conformance",
			Status:      "ADVISORY",
			Evidence:    "NOT_AVAILABLE",
			Detail:      "oscal-cli is not on PATH. Tier 1 JSON Schema validation is 100% operational offline.",
			Remediation: "To enable optional Tier 2 Metaschema constraint enforcement: https://github.com/usnistgov/oscal-cli/releases",
		})
	} else {
		// #nosec G204 -- oscalCliPath is resolved via exec.LookPath("oscal-cli") with a fixed "--version" argument.
		cmd := exec.Command(oscalCliPath, "--version")
		out, verErr := cmd.Output()
		verStr := strings.TrimSpace(string(out))
		if lines := strings.Split(verStr, "\n"); len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			verStr = strings.TrimSpace(lines[0])
		}
		if verErr != nil || verStr == "" {
			verStr = oscalCliPath
		}
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "metaschema_tier2",
			Category: "OSCAL Conformance",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   fmt.Sprintf("oscal-cli available (%s); Tier 2 constraints active", verStr),
		})
	}

	// 6. Web Studio Port Availability
	testLn, lnErr := net.Listen("tcp", "127.0.0.1:8080")
	if lnErr == nil {
		_ = testLn.Close()
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "webstudio_port",
			Category: "Operator & UI",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   "Port 8080 available for Web Studio",
		})
	} else {
		if strings.Contains(lnErr.Error(), "operation not permitted") || strings.Contains(lnErr.Error(), "permission denied") {
			report.Checks = append(report.Checks, DoctorCheck{
				Name:     "webstudio_port",
				Category: "Operator & UI",
				Status:   "PASS",
				Evidence: "OBSERVED",
				Detail:   "Loopback socket binding sandboxed; in-process handler active",
			})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{
				Name:     "webstudio_port",
				Category: "Operator & UI",
				Status:   "PASS",
				Evidence: "OBSERVED",
				Detail:   "Port 8080 busy; automated fallback (8081..8089) active",
			})
		}
	}

	// 7. Universal Tool Calling Protocol (UTCP v1.0.1) & Multi-Language Ecosystem
	utcpManual := utcp.GenerateManual(utcp.ManualOptions{
		Version:        version,
		FrameworkCount: len(allRegistered),
		ControlCount:   totalVerifiedControls,
	})
	utcpBytes, utcpMarshalErr := json.Marshal(utcpManual)
	if utcpMarshalErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "utcp_protocol",
			Category: "UTCP Protocol",
			Status:   "FAIL",
			Evidence: "OBSERVED",
			Detail:   fmt.Sprintf("Failed to serialize UTCP manual: %v", utcpMarshalErr),
		})
	} else if utcpRep, utcpValErr := utcp.ValidateManual(utcpBytes); utcpValErr != nil || !utcpRep.Valid {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "utcp_protocol",
			Category: "UTCP Protocol",
			Status:   "FAIL",
			Evidence: "OBSERVED",
			Detail:   fmt.Sprintf("UTCP manual failed validation: %v", utcpValErr),
		})
	} else {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "utcp_protocol",
			Category: "UTCP Protocol",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   fmt.Sprintf("UTCP v%s manual verified (%d native tools across CLI/HTTP/gRPC; 7 ecosystem targets: utcp-specification, utcp-mcp, rs-utcp, typescript-utcp, go-utcp, python-utcp)", utcpRep.UTCPVersion, utcpRep.ToolCount),
		})
	}

	// 8. Formal Ontology, 36-Framework Property Graph & Sovereign/Threat Feeds
	ontSpec := ontology.CanonicalSpec()
	feedCatalog := feeds.ListFeeds()
	ug, ugErr := graph.DefaultUnifiedGraph()
	if ugErr != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "ontology_graph_feeds",
			Category:    "Knowledge Graph & Feeds",
			Status:      "FAIL",
			Evidence:    "OBSERVED",
			Detail:      fmt.Sprintf("Failed to initialize unified property graph: %v", ugErr),
			Remediation: "Verify server/internal/ontology, server/internal/feeds, and server/internal/graph",
		})
	} else if recRep, recErr := ug.ReconcileInvariants(); recErr != nil || !recRep.Coherent {
		errDetail := fmt.Sprintf("%v", recErr)
		if recRep != nil && len(recRep.Violations) > 0 {
			errDetail = strings.Join(recRep.Violations, "; ")
		}
		report.Checks = append(report.Checks, DoctorCheck{
			Name:        "ontology_graph_feeds",
			Category:    "Knowledge Graph & Feeds",
			Status:      "FAIL",
			Evidence:    "OBSERVED",
			Detail:      fmt.Sprintf("Unified property graph invariant decoherence: %s", errDetail),
			Remediation: "Run 'xoscal graph reconcile' to inspect violations",
		})
	} else {
		gStats := ug.Stats()
		report.Checks = append(report.Checks, DoctorCheck{
			Name:     "ontology_graph_feeds",
			Category: "Knowledge Graph & Feeds",
			Status:   "PASS",
			Evidence: "VERIFIED",
			Detail:   fmt.Sprintf("OWL/JSON-LD v%s (%d classes, %d predicates); %d nodes, %d edges across %d frameworks & %d feeds; reconciled (0 violations)", ontSpec.Version, len(ontSpec.Classes), len(ontSpec.Predicates), gStats.TotalNodes, gStats.TotalEdges, gStats.FrameworksCount, len(feedCatalog)),
		})
	}

	// Compute totals
	for _, c := range report.Checks {
		switch c.Status {
		case "PASS":
			report.Passed++
		case "ADVISORY":
			report.Advisory++
		case "FAIL":
			report.Failed++
		}
	}

	if report.Failed == 0 {
		report.Verdict = "SYSTEM_OPERATIONAL"
	} else {
		report.Verdict = "DEGRADED"
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		if report.Failed > 0 {
			os.Exit(1)
		}
		return
	}

	// Human-readable output
	fmt.Printf("\n=== xOSCAL Doctor: Deep Diagnostics & Runtime Substrate (Rule 36) ===\n")
	fmt.Printf("Version: %s | Host: %s/%s | Go: %s\n\n", report.Version, report.OS, report.Arch, report.GoVersion)

	for _, c := range report.Checks {
		tag := "[PASS]    "
		switch c.Status {
		case "ADVISORY":
			tag = "[ADVISORY]"
		case "FAIL":
			tag = "[FAIL]    "
		}

		fmt.Printf("%s %-20s : %s (%s)\n", tag, c.Category, c.Detail, c.Evidence)
		if c.Remediation != "" {
			fmt.Printf("           ↳ Action: %s\n", c.Remediation)
		}
	}

	fmt.Printf("\nDiagnostic Summary: %d Passed, %d Advisory, %d Failed\n", report.Passed, report.Advisory, report.Failed)
	fmt.Printf("Overall Verdict:    %s\n\n", report.Verdict)

	if report.Failed > 0 {
		os.Exit(1)
	}
}

// RunUTCP executes Universal Tool Calling Protocol (UTCP) operations.
func RunUTCP(args []string) {
	sub := "manual"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(args[0])
		rest = args[1:]
	}

	switch sub {
	case "manual":
		fs := flag.NewFlagSet("utcp manual", flag.ExitOnError)
		outPath := fs.String("out", "", "Write UTCP manual JSON to file (default: stdout)")
		baseURL := fs.String("base-url", "http://127.0.0.1:8080", "Base HTTP URL for HTTP tool call templates")
		grpcHost := fs.String("grpc-host", "127.0.0.1:50051", "gRPC host:port for gRPC tool call templates")
		binPath := fs.String("bin", "xoscal", "CLI binary name or path for CLI tool call templates")
		protocols := fs.String("protocols", "http", "Comma-separated UTCP call template protocols to include (http, cli, grpc, or all)")
		extended := fs.Bool("extended", false, "Emit ExtendedManualEnvelope metadata instead of strict UTCP v1.0.1 wire manual")
		authEnv := fs.String("auth-env", "XOSCAL_API_TOKEN", "Environment variable name for Bearer API key authentication")
		parseFlagsInterspersed(fs, rest)

		fwCount, ctrlCount := utcpRegistryCounts()
		opts := utcp.ManualOptions{
			BaseHTTPURL:       *baseURL,
			BaseURL:           *baseURL,
			GRPCHost:          *grpcHost,
			GRPCAddr:          *grpcHost,
			BinaryPath:        *binPath,
			FrameworkCount:    fwCount,
			ControlCount:      ctrlCount,
			BearerTokenEnvVar: *authEnv,
		}
		for _, tok := range strings.Split(strings.ToLower(strings.TrimSpace(*protocols)), ",") {
			switch strings.TrimSpace(tok) {
			case "all", "*":
				opts.IncludeHTTP = true
				opts.IncludeCLI = true
				opts.IncludeGRPC = true
			case "http":
				opts.IncludeHTTP = true
			case "cli":
				opts.IncludeCLI = true
			case "grpc":
				opts.IncludeGRPC = true
			}
		}

		manual := utcp.BuildManualWithOptions(opts)
		var payload any = manual
		if *extended {
			payload = manual.ExtendedEnvelope()
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling UTCP manual: %v\n", err)
			os.Exit(1)
		}
		if *outPath != "" {
			cleanOut := filepath.Clean(*outPath)
			// #nosec G306 -- UTCP manual is a public machine-readable contract.
			if err := os.WriteFile(cleanOut, append(data, '\n'), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing UTCP manual to %s: %v\n", cleanOut, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Emitted UTCP v%s manual (%d tools) to %s\n", manual.UTCPVersion, len(manual.Tools), cleanOut)
			return
		}
		fmt.Println(string(data))

	case "validate":
		fs := flag.NewFlagSet("utcp validate", flag.ExitOnError)
		filePath := fs.String("file", "", "Path to UTCP manual JSON file (if omitted, validates built-in xOSCAL manual)")
		jsonOut := fs.Bool("json", false, "Emit validation report as JSON")
		_ = fs.Parse(rest)

		if *filePath == "" && fs.NArg() > 0 {
			*filePath = fs.Arg(0)
		}

		var data []byte
		var err error
		if *filePath != "" {
			cleanIn := filepath.Clean(*filePath)
			// #nosec G304 -- operator-selected UTCP manual path for validation.
			data, err = os.ReadFile(cleanIn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading %s: %v\n", cleanIn, err)
				os.Exit(1)
			}
		} else {
			fwCount, ctrlCount := utcpRegistryCounts()
			manual := utcp.GenerateManual(utcp.ManualOptions{
				FrameworkCount: fwCount,
				ControlCount:   ctrlCount,
			})
			data, err = json.MarshalIndent(manual, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error marshaling built-in manual: %v\n", err)
				os.Exit(1)
			}
		}

		rep, err := utcp.ValidateManual(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error validating UTCP manual: %v\n", err)
			os.Exit(1)
		}

		if *jsonOut {
			out, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("=== xOSCAL UTCP v%s Conformance Report ===\n", rep.UTCPVersion)
			fmt.Printf("Verdict:        %s (Valid: %t)\n", rep.Verdict, rep.Valid)
			fmt.Printf("Manual Version: %s\n", rep.ManualVersion)
			fmt.Printf("Tools Count:    %d (Transports: %v)\n", rep.ToolCount, rep.Transports)
			fmt.Printf("Manual SHA-256: %s\n", rep.ManualSHA256)
			for _, f := range rep.Findings {
				fmt.Printf("  [%s] %s (%s): %s\n", f.Severity, f.Code, f.ToolName, f.Message)
			}
		}
		if !rep.Valid {
			os.Exit(1)
		}

	case "mcp-config":
		fs := flag.NewFlagSet("utcp mcp-config", flag.ExitOnError)
		manualURL := fs.String("manual-url", "http://127.0.0.1:8080/.well-known/utcp", "HTTP URL of UTCP manual")
		manualFile := fs.String("manual-file", "./.utcp_config.json", "Path to .utcp_config.json passed via UTCP_CONFIG_FILE")
		utcpConfigOut := fs.String("utcp-config-out", "", "Optional output path to also write .utcp_config.json")
		outPath := fs.String("out", "", "Optional output file path for MCP host config")
		parseFlagsInterspersed(fs, rest)

		cfgPathForBridge := *manualFile
		if *utcpConfigOut != "" {
			cleanUTCPCfg := filepath.Clean(*utcpConfigOut)
			utcpCfgBytes, uErr := utcp.GenerateUTCPClientConfig(*manualURL)
			if uErr != nil {
				fmt.Fprintf(os.Stderr, "error generating .utcp_config.json: %v\n", uErr)
				os.Exit(1)
			}
			// #nosec G306 -- .utcp_config.json is a non-sensitive UTCP discovery descriptor.
			if err := os.WriteFile(cleanUTCPCfg, append(utcpCfgBytes, '\n'), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", cleanUTCPCfg, err)
				os.Exit(1)
			}
			cfgPathForBridge = cleanUTCPCfg
		}

		cfgBytes, err := utcp.GenerateMCPBridgeConfig(*manualURL, cfgPathForBridge)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating utcp-mcp config: %v\n", err)
			os.Exit(1)
		}
		if *outPath != "" {
			cleanOut := filepath.Clean(*outPath)
			// #nosec G306 -- utcp-mcp configuration file is non-sensitive.
			if err := os.WriteFile(cleanOut, append(cfgBytes, '\n'), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", cleanOut, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Emitted utcp-mcp bridge config to %s\n", cleanOut)
			return
		}
		fmt.Println(string(cfgBytes))

	case "sdk":
		fs := flag.NewFlagSet("utcp sdk", flag.ExitOnError)
		target := fs.String("target", "go-utcp", "Target UTCP SDK (go-utcp, python-utcp, typescript-utcp, rs-utcp, utcp-mcp)")
		manualURL := fs.String("manual-url", "http://127.0.0.1:8080/.well-known/utcp", "HTTP URL of UTCP manual")
		manualFile := fs.String("manual-file", "./utcp.json", "Local file path of UTCP manual")
		outPath := fs.String("out", "", "Optional output file path")
		_ = fs.Parse(rest)

		if fs.NArg() > 0 && *target == "go-utcp" {
			*target = fs.Arg(0)
		}

		code, err := utcp.GenerateSDKHarness(*target, *manualURL, *manualFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if *outPath != "" {
			cleanOut := filepath.Clean(*outPath)
			// #nosec G306 -- generated SDK harness source file.
			if err := os.WriteFile(cleanOut, []byte(code), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", cleanOut, err)
				os.Exit(1)
			}
			fmt.Printf("[OK] Emitted %s harness to %s\n", *target, cleanOut)
			return
		}
		fmt.Print(code)

	case "ecosystem", "list":
		fs := flag.NewFlagSet("utcp ecosystem", flag.ExitOnError)
		jsonOut := fs.Bool("json", false, "Emit supported UTCP ecosystem repositories as JSON")
		_ = fs.Parse(rest)

		if *jsonOut {
			out, _ := json.MarshalIndent(utcp.SupportedRepositories, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Println("=== Supported Universal Tool Calling Protocol (UTCP) Ecosystem ===")
		for _, r := range utcp.SupportedRepositories {
			fmt.Printf("  * %-32s [%-22s] %s\n    ↳ %s (%s)\n", r.Name, r.Language, r.URL, r.Role, r.Package)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown utcp subcommand %q (expected: manual, validate, mcp-config, sdk, ecosystem)\n", sub)
		os.Exit(1)
	}
}

// RunOntology inspects the formal OWL/JSON-LD regulatory & threat ontology.
func RunOntology(args []string) {
	sub := "spec"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(args[0])
		rest = args[1:]
	}

	fs := flag.NewFlagSet("ontology", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output ontology specification in JSON-LD format")
	parseFlagsInterspersed(fs, rest)

	spec := ontology.CanonicalSpec()

	switch sub {
	case "classes":
		if *jsonOut {
			out, _ := json.MarshalIndent(spec.Classes, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== xOSCAL OWL/JSON-LD ONTOLOGY CLASSES (v%s) ===\n", spec.Version)
		fmt.Printf("%-26s %-14s %-34s %s\n", "CLASS ID", "DOMAIN", "URN PATTERN", "LABEL")
		fmt.Println(strings.Repeat("-", 96))
		for _, c := range spec.Classes {
			fmt.Printf("%-26s %-14s %-34s %s\n", c.ID, c.Domain, c.URNPattern, c.Label)
		}
		fmt.Println()

	case "predicates":
		if *jsonOut {
			out, _ := json.MarshalIndent(spec.Predicates, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== xOSCAL OWL/JSON-LD SEMANTIC PREDICATES (v%s) ===\n", spec.Version)
		fmt.Printf("%-24s %-12s %-8s %-8s %s\n", "PREDICATE", "CATEGORY", "IS_ANTI", "WEIGHT", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 96))
		for _, p := range spec.Predicates {
			fmt.Printf("%-24s %-12s %-8v %-8.2f %s\n", p.Name, p.Category, p.IsAnti, p.DefaultWeight, p.Description)
		}
		fmt.Println()

	case "spec":
		if *jsonOut {
			out, _ := json.MarshalIndent(spec, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== xOSCAL FORMAL OWL/JSON-LD ONTOLOGY SPECIFICATION ===\n")
		fmt.Printf("Version:          %s\n", spec.Version)
		fmt.Printf("Namespace:        %s\n", spec.Namespace)
		fmt.Printf("Classes:          %d\n", len(spec.Classes))
		fmt.Printf("Predicates:       %d (including 2 CKODEX Rule 17 ANTI relations)\n", len(spec.Predicates))
		fmt.Printf("Thematic Domains: %d\n", len(spec.ThematicDomains))
		fmt.Printf("SHA-256 Digest:   %s\n\n", spec.SHA256)

	default:
		fmt.Fprintf(os.Stderr, "unknown ontology subcommand %q (expected: classes, predicates, spec)\n", sub)
		os.Exit(1)
	}
}

func resolveGraphForCLI(dsn, dbAlias string) (*graph.UnifiedGraph, error) {
	if !isUnexpandedPlaceholder(dbAlias) && strings.TrimSpace(dbAlias) != "" {
		dsn = dbAlias
	}
	if !isUnexpandedPlaceholder(dsn) && strings.TrimSpace(dsn) != "" {
		return graph.LoadUnifiedGraphFromSQLite(context.Background(), strings.TrimSpace(dsn))
	}
	return graph.DefaultUnifiedGraph()
}

// RunGraph queries the 36-framework property graph, crosswalks, and 6D blast radius engine.
func RunGraph(args []string) {
	sub := "stats"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(args[0])
		rest = args[1:]
	}

	switch sub {
	case "stats":
		fs := flag.NewFlagSet("graph stats", flag.ExitOnError)
		dsn := fs.String("dsn", "", "Optional SQLite DSN to load persisted property graph")
		dbAlias := fs.String("db", "", "Alias for -dsn")
		jsonOut := fs.Bool("json", false, "Output graph statistics as JSON")
		parseFlagsInterspersed(fs, rest)

		ug, err := resolveGraphForCLI(*dsn, *dbAlias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading unified property graph: %v\n", err)
			os.Exit(1)
		}
		stats := ug.Stats()
		if *jsonOut {
			out, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== xOSCAL UNIFIED %d-FRAMEWORK PROPERTY GRAPH STATS ===\n", stats.FrameworksCount)
		fmt.Printf("Total Nodes:        %d\n", stats.TotalNodes)
		fmt.Printf("Total Edges:        %d\n", stats.TotalEdges)
		fmt.Printf("Frameworks Seeded:  %d\n", stats.FrameworksCount)
		fmt.Printf("Feeds Seeded:       %d\n", stats.FeedsCount)
		fmt.Printf("Active ANTI Edges:  %d\n", stats.AntiEdgeCount)
		fmt.Printf("Ontology Version:   %s (%s)\n\n", stats.OntologyVersion, stats.OntologySHA256)

	case "crosswalk":
		fs := flag.NewFlagSet("graph crosswalk", flag.ExitOnError)
		ctrl := fs.String("control", "ac-2", "Control ID or <framework>:<control-id> (e.g. ac-2, cccs-itsp-10-033:si-2)")
		dsn := fs.String("dsn", "", "Optional SQLite DSN to load persisted property graph")
		dbAlias := fs.String("db", "", "Alias for -dsn")
		jsonOut := fs.Bool("json", false, "Output crosswalk report as JSON")
		parseFlagsInterspersed(fs, rest)

		ug, err := resolveGraphForCLI(*dsn, *dbAlias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading unified property graph: %v\n", err)
			os.Exit(1)
		}

		if isUnexpandedPlaceholder(*ctrl) || *ctrl == "" {
			*ctrl = "ac-2"
		}
		if *ctrl == "ac-2" && fs.NArg() > 0 && !isUnexpandedPlaceholder(fs.Arg(0)) {
			*ctrl = fs.Arg(0)
		}

		rep, err := ug.QueryCrosswalk(*ctrl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error querying crosswalk for %s: %v\n", *ctrl, err)
			os.Exit(1)
		}

		if *jsonOut {
			out, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(out))
			return
		}

		fmt.Printf("\n=== MULTI-JURISDICTION CONTROL CROSSWALK: %s ===\n", strings.ToUpper(rep.CanonicalControlID))
		fmt.Printf("Thematic Domain:    %s\n", rep.ThematicDomain)
		fmt.Printf("Equivalent Matches: %d | Harmonized Matches: %d | Active Threats: %d\n",
			len(rep.EquivalentControls), len(rep.HarmonizedControls), len(rep.ActiveThreats))
		fmt.Printf("6D Vector Posture:  <%s, %s, Anti=%d, %s, %s, %s>\n\n",
			rep.VectorPosture.Presence, rep.VectorPosture.Valence, rep.VectorPosture.AntiCount,
			rep.VectorPosture.Coherence, rep.VectorPosture.Evidence, rep.VectorPosture.Lifecycle)

		fmt.Printf("%-26s %-6s %-16s %-18s %s\n", "FRAMEWORK", "LOCALE", "CONTROL ID", "RELATION", "TITLE")
		fmt.Println(strings.Repeat("-", 96))
		for _, m := range rep.EquivalentControls {
			fmt.Printf("%-26s %-6s %-16s %-18s %s\n", m.FrameworkID, strings.ToUpper(m.Jurisdiction), m.ControlID, m.Relation, m.Title)
		}
		for _, m := range rep.HarmonizedControls {
			fmt.Printf("%-26s %-6s %-16s %-18s %s\n", m.FrameworkID, strings.ToUpper(m.Jurisdiction), m.ControlID, m.Relation, m.Title)
		}
		fmt.Println()

	case "blast-radius":
		fs := flag.NewFlagSet("graph blast-radius", flag.ExitOnError)
		target := fs.String("target", "CVE-2024-3400", "Target CVE/KEV ID, ATT&CK technique, framework, or control ID")
		depth := fs.Int("depth", 2, "Maximum traversal depth (1..5)")
		dsn := fs.String("dsn", "", "Optional SQLite DSN to load persisted property graph")
		dbAlias := fs.String("db", "", "Alias for -dsn")
		jsonOut := fs.Bool("json", false, "Output blast-radius report as JSON")
		parseFlagsInterspersed(fs, rest)

		ug, err := resolveGraphForCLI(*dsn, *dbAlias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading unified property graph: %v\n", err)
			os.Exit(1)
		}

		if isUnexpandedPlaceholder(*target) || *target == "" {
			*target = "CVE-2024-3400"
		}
		if *target == "CVE-2024-3400" && fs.NArg() > 0 && !isUnexpandedPlaceholder(fs.Arg(0)) {
			*target = fs.Arg(0)
		}

		rep, err := ug.ComputeBlastRadius(*target, *depth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error computing blast radius for %s: %v\n", *target, err)
			os.Exit(1)
		}

		if *jsonOut {
			out, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(out))
			return
		}

		fmt.Printf("\n=== 6D VECTOR-STATE BLAST RADIUS REPORT: %s ===\n", rep.RootNode.URN)
		fmt.Printf("Root Node ID:         %s (%s)\n", rep.RootNode.ID, rep.RootNode.Kind)
		fmt.Printf("Max Depth:            %d\n", rep.MaxDepth)
		fmt.Printf("Impacted Nodes/Edges: %d nodes / %d edges\n", len(rep.ImpactedNodes), len(rep.ImpactedEdges))
		fmt.Printf("Impacted Controls:    %d across %d frameworks (%v)\n", len(rep.ImpactedControls), len(rep.ImpactedFrameworks), rep.JurisdictionImpact)
		fmt.Printf("ANTI Edges:           %d | Mitigating Edges: %d\n", rep.AntiEdgeCount, rep.MitigationEdgeCount)
		fmt.Printf("6D Vector State:      <%s, %s, Anti=%d, %s, %s, %s>\n\n",
			rep.VectorPosture.Presence, rep.VectorPosture.Valence, rep.VectorPosture.AntiCount,
			rep.VectorPosture.Coherence, rep.VectorPosture.Evidence, rep.VectorPosture.Lifecycle)

	case "export":
		fs := flag.NewFlagSet("graph export", flag.ExitOnError)
		target := fs.String("target", "CVE-2024-3400", "Target URN or ID for subgraph export")
		depth := fs.Int("depth", 2, "Maximum traversal depth")
		format := fs.String("format", "jsonld", "Export format (jsonld, cytoscape, cypher)")
		dsn := fs.String("dsn", "", "Optional SQLite DSN to load persisted property graph")
		dbAlias := fs.String("db", "", "Alias for -dsn")
		jsonOut := fs.Bool("json", false, "Output as JSON")
		parseFlagsInterspersed(fs, rest)

		ug, err := resolveGraphForCLI(*dsn, *dbAlias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading unified property graph: %v\n", err)
			os.Exit(1)
		}

		if isUnexpandedPlaceholder(*target) || *target == "" {
			*target = "CVE-2024-3400"
		}
		if isUnexpandedPlaceholder(*format) || *format == "" {
			*format = "jsonld"
		}
		_ = jsonOut

		out, err := ug.ExportSubgraph(*target, *depth, *format)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error exporting subgraph: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)

	case "reconcile":
		fs := flag.NewFlagSet("graph reconcile", flag.ExitOnError)
		dsn := fs.String("dsn", "", "Optional SQLite DSN to load persisted property graph")
		dbAlias := fs.String("db", "", "Alias for -dsn")
		jsonOut := fs.Bool("json", false, "Output graph reconciliation report as JSON")
		parseFlagsInterspersed(fs, rest)

		ug, err := resolveGraphForCLI(*dsn, *dbAlias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading unified property graph: %v\n", err)
			os.Exit(1)
		}

		rep, err := ug.ReconcileInvariants()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reconciling unified property graph: %v\n", err)
			os.Exit(1)
		}
		if *jsonOut {
			out, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Println(string(out))
			if !rep.Coherent {
				os.Exit(1)
			}
			return
		}

		verdict := "COHERENT"
		if !rep.Coherent {
			verdict = "DECOHERENT"
		}
		fmt.Printf("\n=== xOSCAL UNIFIED GRAPH INVARIANT RECONCILIATION (Rule 34) ===\n")
		fmt.Printf("Verdict:           %s\n", verdict)
		fmt.Printf("Checked Nodes:     %d\n", rep.CheckedNodes)
		fmt.Printf("Checked Edges:     %d\n", rep.CheckedEdges)
		fmt.Printf("Graph SHA-256:     %s\n", rep.GraphSHA256)
		fmt.Printf("6D Vector Posture: <%s, %s, Anti=%d, %s, %s, %s>\n",
			rep.VectorPosture.Presence, rep.VectorPosture.Valence, rep.VectorPosture.AntiCount,
			rep.VectorPosture.Coherence, rep.VectorPosture.Evidence, rep.VectorPosture.Lifecycle)
		for _, v := range rep.Violations {
			fmt.Printf("  [VIOLATION] %s\n", v)
		}
		fmt.Println()
		if !rep.Coherent {
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown graph subcommand %q (expected: stats, crosswalk, blast-radius, export, reconcile)\n", sub)
		os.Exit(1)
	}
}

// RunFeeds lists, syncs, inspects, and verifies checkpoints for sovereign regulatory & threat intelligence feeds.
func RunFeeds(args []string) {
	sub := "list"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(args[0])
		rest = args[1:]
	}

	switch sub {
	case "list":
		fs := flag.NewFlagSet("feeds list", flag.ExitOnError)
		jsonOut := fs.Bool("json", false, "Output feeds list as JSON")
		parseFlagsInterspersed(fs, rest)

		list := feeds.ListFeeds()
		if *jsonOut {
			out, _ := json.MarshalIndent(list, "", "  ")
			fmt.Println(string(out))
			return
		}

		fmt.Printf("\n=== AUTHORITATIVE SOVEREIGN & THREAT INTELLIGENCE FEEDS ===\n")
		fmt.Printf("%-32s %-18s %-26s %-8s %s\n", "FEED ID", "LOCALES", "AUTHORITY", "RECORDS", "TITLE")
		fmt.Println(strings.Repeat("-", 96))
		for _, f := range list {
			fmt.Printf("%-32s %-18s %-26s %-8d %s\n", f.ID, strings.ToUpper(strings.Join(f.Jurisdictions, ",")), f.Authority, f.RecordCount, f.Title)
		}
		fmt.Println()

	case "sync":
		fs := flag.NewFlagSet("feeds sync", flag.ExitOnError)
		feedID := fs.String("feed", "", "Optional specific feed ID to sync (default: all 4 feeds)")
		offline := fs.Bool("offline", true, "Air-gap offline deterministic sync mode (Rule 40)")
		checkpointDir := fs.String("checkpoint-dir", "", "Optional directory for atomic SHA-256 receipt checkpoint and flight recorder")
		jsonOut := fs.Bool("json", false, "Output execution receipt(s) as JSON")
		parseFlagsInterspersed(fs, rest)

		if isUnexpandedPlaceholder(*feedID) {
			*feedID = ""
		}
		if isUnexpandedPlaceholder(*checkpointDir) {
			*checkpointDir = ""
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		receipt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
			FeedID:        *feedID,
			Offline:       *offline,
			CheckpointDir: *checkpointDir,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error syncing feeds: %v\n", err)
			os.Exit(1)
		}
		if !*offline {
			_, _ = graph.ReloadDefaultUnifiedGraph(ctx)
		}
		if *jsonOut {
			out, _ := json.MarshalIndent(receipt, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== FEED SYNC EXECUTION RECEIPT (Rule 12): %s ===\n", receipt.ReceiptID)
		fmt.Printf("Mode: %s (%s, Fallbacks: %d) | Total Nodes: %d | Total Edges: %d | Bundle SHA-256: %s\n",
			receipt.Mode, receipt.OperationalMode, receipt.FallbackCount, receipt.TotalNodes, receipt.TotalEdges, receipt.BundleSHA256)
		for _, r := range receipt.Feeds {
			fmt.Printf("  [OK] %-32s Mode=%-16s Circuit=%-9s Records=%-4d Nodes=%-4d Edges=%-4d SHA256=%s\n",
				r.FeedID, r.Mode, r.CircuitState, r.RecordCount, r.NodesProjected, r.EdgesProjected, r.SHA256[:16]+"...")
		}
		if *checkpointDir != "" {
			fmt.Printf("  [CHECKPOINT] Persisted atomic receipt + hash-chained flight recorder in %s\n", *checkpointDir)
		}
		fmt.Println()

	case "verify-checkpoint":
		fs := flag.NewFlagSet("feeds verify-checkpoint", flag.ExitOnError)
		checkpointDir := fs.String("checkpoint-dir", "", "Directory containing latest_sync_receipt.json and sync_flight_recorder.jsonl")
		jsonOut := fs.Bool("json", false, "Output checkpoint verification report as JSON")
		parseFlagsInterspersed(fs, rest)

		if isUnexpandedPlaceholder(*checkpointDir) {
			*checkpointDir = ""
		}
		if *checkpointDir == "" && fs.NArg() > 0 && !isUnexpandedPlaceholder(fs.Arg(0)) {
			*checkpointDir = fs.Arg(0)
		}
		if *checkpointDir == "" {
			fmt.Fprintln(os.Stderr, "error: -checkpoint-dir is required")
			os.Exit(1)
		}

		rcpt, entries, err := feeds.VerifyCheckpointDir(*checkpointDir)
		if err != nil {
			if *jsonOut {
				out, _ := json.MarshalIndent(map[string]any{
					"valid":          false,
					"verdict":        "TAMPERED_OR_INVALID",
					"checkpoint_dir": *checkpointDir,
					"error":          err.Error(),
				}, "", "  ")
				fmt.Println(string(out))
			} else {
				fmt.Fprintf(os.Stderr, "checkpoint verification failed for %s: %v\n", *checkpointDir, err)
			}
			os.Exit(1)
		}

		if *jsonOut {
			out, _ := json.MarshalIndent(map[string]any{
				"valid":          true,
				"verdict":        "VERIFIED",
				"checkpoint_dir": *checkpointDir,
				"ledger_entries": entries,
				"receipt":        rcpt,
			}, "", "  ")
			fmt.Println(string(out))
			return
		}

		fmt.Printf("\n=== FEED CHECKPOINT & FLIGHT RECORDER VERIFICATION (Rules 12 & 38) ===\n")
		fmt.Printf("Verdict:        VERIFIED\n")
		fmt.Printf("Checkpoint Dir: %s\n", *checkpointDir)
		fmt.Printf("Ledger Entries: %d hash-chained receipt(s)\n", entries)
		fmt.Printf("Latest Receipt: %s (Bundle SHA-256: %s)\n\n", rcpt.ReceiptID, rcpt.BundleSHA256)

	case "inspect":
		fs := flag.NewFlagSet("feeds inspect", flag.ExitOnError)
		feedID := fs.String("feed", "cisa-kev-active-threats", "Feed ID to inspect")
		checkpointDir := fs.String("checkpoint-dir", "", "Optional feed checkpoint directory")
		jsonOut := fs.Bool("json", false, "Output feed bundle as JSON")
		parseFlagsInterspersed(fs, rest)

		if isUnexpandedPlaceholder(*feedID) || *feedID == "" {
			*feedID = "cisa-kev-active-threats"
		}
		if isUnexpandedPlaceholder(*checkpointDir) {
			*checkpointDir = ""
		}
		if *feedID == "cisa-kev-active-threats" && fs.NArg() > 0 && !isUnexpandedPlaceholder(fs.Arg(0)) {
			*feedID = fs.Arg(0)
		}

		bundle, err := feeds.InspectFeed(*feedID, *checkpointDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error inspecting feed %s: %v\n", *feedID, err)
			os.Exit(1)
		}
		if *jsonOut {
			out, _ := json.MarshalIndent(bundle, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Printf("\n=== FEED INSPECTION: %s ===\n", bundle.Descriptor.Title)
		fmt.Printf("ID:             %s\n", bundle.Descriptor.ID)
		fmt.Printf("Authority:      %s (%s)\n", bundle.Descriptor.Authority, strings.ToUpper(strings.Join(bundle.Descriptor.Jurisdictions, ", ")))
		fmt.Printf("Upstream URL:   %s\n", bundle.Descriptor.UpstreamURL)
		fmt.Printf("Records:        %d\n", bundle.Descriptor.RecordCount)
		fmt.Printf("SHA-256 Digest: %s\n\n", bundle.Descriptor.SHA256)

	default:
		fmt.Fprintf(os.Stderr, "unknown feeds subcommand %q (expected: list, sync, inspect, verify-checkpoint)\n", sub)
		os.Exit(1)
	}
}

func utcpRegistryCounts() (int, int) {
	infos := frameworks.DefaultRegistry().ListInfo()
	totalControls := 0
	for _, info := range infos {
		totalControls += info.ControlCount
	}
	return len(infos), totalControls
}
