package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("xoscal-generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	dbAlias := fs.String("db", "", "Alias for -dsn")
	snapshot := fs.String("snapshot", "", "Snapshot name")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	outDir := fs.String("out", "./oscal-out", "Output directory")
	arOnly := fs.Bool("assessment-results-only", false, "Emit only assessment-results (useful for CI)")
	validate := fs.Bool("validate", false, "Schema-validate each artifact after generation; fail on non-compliance")
	validateConstraints := fs.Bool("validate-constraints", false, "Also run oscal-cli Metaschema constraint validation after generation; requires oscal-cli on PATH")
	oscalCLIPath := fs.String("oscal-cli", "", "path to oscal-cli binary (default: search PATH)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	isPlaceholder := func(s string) bool {
		s = strings.TrimSpace(s)
		return (strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}")) ||
			(strings.HasPrefix(s, "UTCP_ARG_") && strings.HasSuffix(s, "_UTCP_END"))
	}
	if isPlaceholder(*dbAlias) {
		*dbAlias = ""
	}
	if *dbAlias != "" {
		*dsn = *dbAlias
	}
	if isPlaceholder(*dsn) || *dsn == "" {
		*dsn = "oscal.db"
	}
	if isPlaceholder(*framework) || *framework == "" {
		*framework = "eu-ai-act"
	}
	if isPlaceholder(*outDir) || *outDir == "" {
		*outDir = "./oscal-out"
	}
	if isPlaceholder(*snapshot) {
		*snapshot = ""
	}

	if *snapshot == "" {
		return errors.New("-snapshot is required")
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = store.Close() }()

	gen := oscal.NewGenerator(store)
	ctx := context.Background()

	entities, err := store.GetSnapshot(ctx, *snapshot)
	if err != nil {
		return fmt.Errorf("get snapshot %q: %w", *snapshot, err)
	}
	reqCount := 0
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err == nil && req.Assessable && strings.EqualFold(req.Framework, *framework) {
			reqCount++
		}
	}
	if reqCount == 0 {
		return fmt.Errorf("snapshot %q contains 0 requirements for framework %q", *snapshot, *framework)
	}

	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Optional schema validator (tier 1)
	var validator *schemavalidate.Validator
	if *validate || *validateConstraints {
		v, err := schemavalidate.NewValidator()
		if err != nil {
			return fmt.Errorf("init schema validator: %w", err)
		}
		validator = v
	}

	// Optional constraint validator (tier 2, requires oscal-cli)
	var cliPath string
	if *validateConstraints {
		cp, err := resolveOSCALCLI(*oscalCLIPath)
		if err != nil {
			return fmt.Errorf("constraint validation requested but oscal-cli not found: %w", err)
		}
		cliPath = cp
	}

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, *snapshot, *framework)
		if err != nil {
			return fmt.Errorf("generate assessment-results: %w", err)
		}
		path := filepath.Join(*outDir, "assessment-results.json")
		if err := writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportAssessmentResultsJSON(ar)
		}, validator, schemavalidate.KindAssessmentResults); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "Assessment results written to %s\n", path)
		if *validateConstraints {
			if err := runConstraintValidation(cliPath, path, schemavalidate.KindAssessmentResults, stdout); err != nil {
				return err
			}
		}
		return nil
	}

	res, err := gen.GenerateAllArtifacts(ctx, *snapshot, *framework, nil)
	if err != nil {
		return fmt.Errorf("generate all: %w", err)
	}

	type writtenFile struct {
		path string
		kind schemavalidate.ArtifactKind
	}
	var writtenFiles []writtenFile

	if res.Catalog != nil {
		c := res.Catalog
		path := filepath.Join(*outDir, "catalog.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportCatalogJSON(c) }, validator, schemavalidate.KindCatalog); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindCatalog})
	}
	if res.Profile != nil {
		p := res.Profile
		path := filepath.Join(*outDir, "profile.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportProfileJSON(p) }, validator, schemavalidate.KindProfile); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindProfile})
	}
	if res.SSP != nil {
		s := res.SSP
		path := filepath.Join(*outDir, "ssp.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportSSPJSON(s) }, validator, schemavalidate.KindSSP); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindSSP})
	}
	if res.ComponentDefinition != nil {
		c := res.ComponentDefinition
		path := filepath.Join(*outDir, "component-definition.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportComponentDefinitionJSON(c) }, validator, schemavalidate.KindComponentDefinition); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindComponentDefinition})
	}
	if res.AssessmentPlan != nil {
		a := res.AssessmentPlan
		path := filepath.Join(*outDir, "assessment-plan.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportAssessmentPlanJSON(a) }, validator, schemavalidate.KindAssessmentPlan); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindAssessmentPlan})
	}
	if res.AssessmentResults != nil {
		a := res.AssessmentResults
		path := filepath.Join(*outDir, "assessment-results.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportAssessmentResultsJSON(a) }, validator, schemavalidate.KindAssessmentResults); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindAssessmentResults})
	}
	if res.POAM != nil {
		p := res.POAM
		path := filepath.Join(*outDir, "poam.json")
		if err := writeOSCAL(path, func() ([]byte, error) { return oscal.ExportPOAMJSON(p) }, validator, schemavalidate.KindPOAM); err != nil {
			return err
		}
		writtenFiles = append(writtenFiles, writtenFile{path, schemavalidate.KindPOAM})
	}
	if len(res.Mappings) > 0 {
		mappingsData, err := json.MarshalIndent(struct{ Maps []*mappingv1.Map }{Maps: res.Mappings}, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal mappings: %w", err)
		}
		mappingsPath := filepath.Join(*outDir, "mappings.json")
		// #nosec G304,G306 -- mappingsPath is beneath the explicit output directory selected by the operator.
		if err := os.WriteFile(mappingsPath, mappingsData, 0o600); err != nil {
			return fmt.Errorf("write mappings: %w", err)
		}
		if _, err := dbutil.WriteDigestSidecar(mappingsPath); err != nil {
			return fmt.Errorf("write digest sidecar for %s: %w", mappingsPath, err)
		}
	}

	_, _ = fmt.Fprintf(stdout, "OSCAL artifacts written to %s\n", *outDir)

	if *validateConstraints {
		for _, wf := range writtenFiles {
			if err := runConstraintValidation(cliPath, wf.path, wf.kind, stdout); err != nil {
				return err
			}
		}
		_, _ = fmt.Fprintln(stdout, "All artifacts passed Metaschema constraint validation")
	}
	return nil
}

func writeOSCAL(path string, marshal func() ([]byte, error), validator *schemavalidate.Validator, kind schemavalidate.ArtifactKind) error {
	b, err := marshal()
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if validator != nil {
		if err := validator.Validate(b, kind); err != nil {
			return fmt.Errorf("schema validation failed for %s: %w", path, err)
		}
	}
	// #nosec G304,G306 -- path is the explicit output destination selected by the CLI operator.
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := dbutil.WriteDigestSidecar(path); err != nil {
		return fmt.Errorf("write digest sidecar for %s: %w", path, err)
	}
	return nil
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

func runConstraintValidation(cliPath, filePath string, kind schemavalidate.ArtifactKind, stdout io.Writer) error {
	model, ok := kindToCLIModel[kind]
	if !ok {
		return nil
	}
	// #nosec G204 -- cliPath is resolved from the explicit flag or PATH by resolveOSCALCLI.
	cmd := exec.Command(cliPath, model, "validate", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("metaschema constraint validation failed for %s:\n%s", filePath, string(output))
	}
	_, _ = fmt.Fprintf(stdout, "  constraint-valid: %s\n", filePath)
	return nil
}

func resolveOSCALCLI(explicit string) (string, error) {
	if explicit != "" {
		if _, err := exec.LookPath(explicit); err != nil {
			return "", fmt.Errorf("oscal-cli not found at %s: %w", explicit, err)
		}
		return explicit, nil
	}
	path, err := exec.LookPath("oscal-cli")
	if err != nil {
		return "", fmt.Errorf("oscal-cli not found in PATH: %w", err)
	}
	return path, nil
}
