// Command xoscal-validate-constraints performs full Metaschema-constraint
// validation on OSCAL artifacts by wrapping the NIST oscal-cli tool.
//
// This is the notch beyond the embedded JSON Schema (AJV-equivalent) validator
// in xoscal-validate-schema. JSON Schema validation catches structural/shape
// errors; oscal-cli validate additionally enforces Metaschema constraints:
//
//   - <allowed-values> — enumerated prop names, link rels, class tokens
//   - <has-cardinality> — min/max occurrences of assemblies
//   - <index-has-key> — cross-reference integrity (UUID refs, role-id refs)
//   - <is-unique> — no duplicate IDs within a scope
//   - <matches> — regex constraints on token/string values
//
// Usage:
//
//	xoscal-validate-constraints -file <path> [-kind <kind>] [-oscal-cli <path>]
//	xoscal-validate-constraints -dir <dir>  [-oscal-cli <path>]
//
// If -kind is not specified, the root key in the JSON is auto-detected.
// If -oscal-cli is not specified, PATH is searched for "oscal-cli".
//
// By default, the embedded JSON Schema validator runs first as a fast pre-check.
// Use -skip-schema to skip it and go straight to oscal-cli.
//
// Exit codes: 0 = valid, 1 = error (tool/usage), 3 = validation failure.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// modelCLIName maps an OSCAL root key to the subcommand oscal-cli 1.0.3
// expects. Assessment Plan and Assessment Results use the short subcommands
// ap and ar; they are not named after their OSCAL root keys.
var modelCLIName = map[string]string{
	"catalog":                       "catalog",
	"profile":                       "profile",
	"system-security-plan":          "ssp",
	"component-definition":          "component-definition",
	"assessment-plan":               "ap",
	"assessment-results":            "ar",
	"plan-of-action-and-milestones": "poam",
}

// rootKeyOrder is the deterministic set of OSCAL roots supported by the
// pinned oscal-cli release. A JSON object with more than one of these roots is
// ambiguous and must not be accepted by auto-detection.
var rootKeyOrder = []string{
	"catalog",
	"profile",
	"system-security-plan",
	"component-definition",
	"assessment-plan",
	"assessment-results",
	"plan-of-action-and-milestones",
}

func main() {
	filePath := flag.String("file", "", "path to a single OSCAL JSON artifact to validate")
	dirPath := flag.String("dir", "", "directory of OSCAL JSON artifacts to validate (recursive)")
	kindFlag := flag.String("kind", "", "artifact kind (catalog, profile, ssp, component-definition, assessment-plan, assessment-results, poam)")
	oscalCLIPath := flag.String("oscal-cli", "", "path to the oscal-cli binary (default: search PATH)")
	skipSchema := flag.Bool("skip-schema", false, "skip the embedded JSON Schema pre-check")
	flag.Parse()

	if (*filePath == "") == (*dirPath == "") {
		fmt.Fprintln(os.Stderr, "error: exactly one of -file or -dir is required")
		flag.Usage()
		os.Exit(1)
	}

	cliPath, err := resolveOSCALCLI(*oscalCLIPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var schemaValidator *schemavalidate.Validator
	if !*skipSchema {
		schemaValidator, err = schemavalidate.NewValidator()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load embedded schema validator: %v (continuing with oscal-cli only)\n", err)
		}
	}

	var files []string
	switch {
	case *filePath != "":
		files = []string{*filePath}
	case *dirPath != "":
		err = filepath.WalkDir(*dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".json") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error walking directory: %v\n", err)
			os.Exit(1)
		}
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "error: no .json files found in %s\n", *dirPath)
			os.Exit(1)
		}
	}

	exitCode := 0
	for _, f := range files {
		if err := validateFile(f, *kindFlag, cliPath, schemaValidator); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", f, err)
			if exitCode == 0 {
				exitCode = 3
			}
		} else {
			fmt.Printf("PASS: %s — schema + Metaschema constraints valid\n", f)
		}
	}
	os.Exit(exitCode)
}

// validateFile runs the schema pre-check (if enabled) then oscal-cli validate.
func validateFile(path, kindFlag, cliPath string, sv *schemavalidate.Validator) error {
	// #nosec G304 -- path is the explicit artifact path selected by the CLI operator.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	rootKey, err := resolveRootKey(kindFlag, data)
	if err != nil {
		return err
	}

	cliModel, ok := modelCLIName[rootKey]
	if !ok {
		return fmt.Errorf("no oscal-cli subcommand for root key: %s", rootKey)
	}

	// Phase 1: embedded JSON Schema pre-check (fast, offline, structural).
	if sv != nil {
		kind, kerr := kindFromRootKey(rootKey)
		if kerr != nil {
			return kerr
		}
		if err := sv.Validate(data, kind); err != nil {
			return fmt.Errorf("schema pre-check failed (fix this before constraint validation):\n%v", err)
		}
	}

	// Phase 2: oscal-cli validate (full Metaschema constraints).
	// #nosec G204 -- cliPath is resolved from the explicit flag or PATH by resolveOSCALCLI.
	cmd := exec.Command(cliPath, cliModel, "validate", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("oscal-cli constraint validation failed:\n%s", string(output))
		}
		return fmt.Errorf("oscal-cli execution error: %w\n%s", err, string(output))
	}
	return nil
}

// resolveRootKey determines the OSCAL root key from the -kind flag or by
// auto-detecting the root key in the JSON document.
func resolveRootKey(kindFlag string, data []byte) (string, error) {
	var expectedRoot string
	if kindFlag != "" {
		rootKey, ok := kindToRootKey(kindFlag)
		if !ok {
			return "", fmt.Errorf("unknown kind: %s", kindFlag)
		}
		expectedRoot = rootKey
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("unmarshal JSON: %w", err)
	}

	if expectedRoot != "" {
		if _, ok := doc[expectedRoot]; !ok {
			return "", fmt.Errorf("kind %q expects root key %q", kindFlag, expectedRoot)
		}
		return expectedRoot, nil
	}

	var found []string
	for _, rootKey := range rootKeyOrder {
		if _, ok := doc[rootKey]; ok {
			found = append(found, rootKey)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("could not auto-detect artifact kind; specify -kind explicitly")
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("ambiguous OSCAL document: multiple root keys: %s", strings.Join(found, ", "))
	}
}

func kindFromRootKey(rootKey string) (schemavalidate.ArtifactKind, error) {
	switch rootKey {
	case "catalog":
		return schemavalidate.KindCatalog, nil
	case "profile":
		return schemavalidate.KindProfile, nil
	case "system-security-plan":
		return schemavalidate.KindSSP, nil
	case "component-definition":
		return schemavalidate.KindComponentDefinition, nil
	case "assessment-plan":
		return schemavalidate.KindAssessmentPlan, nil
	case "assessment-results":
		return schemavalidate.KindAssessmentResults, nil
	case "plan-of-action-and-milestones":
		return schemavalidate.KindPOAM, nil
	default:
		return schemavalidate.ArtifactKind{}, fmt.Errorf("no embedded schema for root key: %s", rootKey)
	}
}

func kindToRootKey(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "catalog":
		return "catalog", true
	case "profile":
		return "profile", true
	case "ssp", "system-security-plan":
		return "system-security-plan", true
	case "component-definition", "component":
		return "component-definition", true
	case "assessment-plan", "ap":
		return "assessment-plan", true
	case "assessment-results", "ar":
		return "assessment-results", true
	case "poam", "plan-of-action-and-milestones":
		return "plan-of-action-and-milestones", true
	}
	return "", false
}

// resolveOSCALCLI finds the oscal-cli binary: explicit flag > PATH lookup.
func resolveOSCALCLI(explicit string) (string, error) {
	if explicit != "" {
		if _, err := exec.LookPath(explicit); err != nil {
			return "", fmt.Errorf("oscal-cli not found at %s: %w", explicit, err)
		}
		return explicit, nil
	}
	path, err := exec.LookPath("oscal-cli")
	if err != nil {
		return "", fmt.Errorf("oscal-cli not found in PATH (install from https://github.com/usnistgov/oscal-cli or pass -oscal-cli <path>): %w", err)
	}
	return path, nil
}
