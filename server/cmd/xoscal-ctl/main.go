// Command xoscal-ctl is the unified developer and compliance operator CLI for xOSCAL.
// It provides frictionless Day-0 onboarding (init), continuous Vector State verification (verify),
// GitOps semantic compliance diffing (diff), formal risk derogation management (derogate),
// GRC analyst spreadsheet bridging (tabular), and air-gapped audit offboarding (bundle-export).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/bundle"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/diff"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/tabular"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

const version = "1.0.0"

func printUsage() {
	fmt.Printf(`xoscal-ctl v%s — Unified xOSCAL Compliance & Governance Engine

USAGE:
  xoscal-ctl <command> [options]

COMMANDS:
  init           Auto-detect repository stack and scaffold OSCAL 1.2.3 Component Definition
  verify         Evaluate artifact against schemas and compute multi-dimensional Vector State
  diff           Compute semantic compliance delta between base and PR head (GitOps)
  derogate       Manage formal time-bounded risk exceptions (Rule 23)
  tabular        Convert between OSCAL JSON and standard CSV/Excel tables for GRC analysts
  bundle-export  Package an air-gapped, verifiable audit bundle with offline HTML viewer
  version        Print CLI version and exit

Run 'xoscal-ctl <command> -h' for details on specific command options.
`, version)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "derogate":
		runDerogate(os.Args[2:])
	case "tabular":
		runTabular(os.Args[2:])
	case "bundle-export":
		runBundleExport(os.Args[2:])
	case "version", "-version", "--version":
		fmt.Printf("xoscal-ctl version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// runInit scaffolds a new OSCAL component definition based on repo inspection.
func runInit(args []string) {
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
	fmt.Printf("\nNext step: Run 'xoscal-ctl verify' to check control posture.\n")
}

// runVerify evaluates artifacts and outputs Vector State.
func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	file := fs.String("file", "component-definition.json", "Path to OSCAL JSON artifact to evaluate")
	evidenceDir := fs.String("evidence-dir", "", "Optional path to evidence directory with SHA-256 sidecars")
	jsonOut := fs.Bool("json", false, "Output Vector State as JSON")
	_ = fs.Parse(args)

	if _, err := os.Stat(*file); os.IsNotExist(err) {
		fallback := filepath.Join(".xoscal", *file)
		if _, err := os.Stat(fallback); err == nil {
			*file = fallback
		}
	}

	state, err := vectorstate.EvaluateArtifact(*file, *evidenceDir)
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

// runDiff computes compliance delta between baseline and PR head.
func runDiff(args []string) {
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

// runDerogate manages time-bounded risk acceptances per Rule 23.
func runDerogate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal-ctl derogate [add|list|revoke] [options]")
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

// runTabular bridges OSCAL and CSV for GRC analysts.
func runTabular(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: xoscal-ctl tabular [export|import] [options]")
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

// runBundleExport packages artifacts and offline HTML viewer into a verifiable archive.
func runBundleExport(args []string) {
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
