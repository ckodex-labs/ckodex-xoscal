package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func main() {
	var (
		dsn                 = flag.String("dsn", "oscal.db", "SQLite DSN")
		snapshot            = flag.String("snapshot", "", "Snapshot name")
		framework           = flag.String("framework", "eu-ai-act", "Framework identifier")
		outDir              = flag.String("out", "./oscal-out", "Output directory")
		arOnly              = flag.Bool("assessment-results-only", false, "Emit only assessment-results (useful for CI)")
		validate            = flag.Bool("validate", false, "Schema-validate each artifact after generation; fail on non-compliance")
		validateConstraints = flag.Bool("validate-constraints", false, "Also run oscal-cli Metaschema constraint validation after generation; requires oscal-cli on PATH")
		oscalCLIPath        = flag.String("oscal-cli", "", "path to oscal-cli binary (default: search PATH)")
	)
	flag.Parse()

	if *snapshot == "" {
		log.Fatal("-snapshot is required")
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	gen := oscal.NewGenerator(store)
	ctx := context.Background()

	if err := os.MkdirAll(*outDir, 0750); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	// Optional schema validator (tier 1)
	var validator *schemavalidate.Validator
	if *validate || *validateConstraints {
		v, err := schemavalidate.NewValidator()
		if err != nil {
			log.Fatalf("init schema validator: %v", err)
		}
		validator = v
	}

	// Optional constraint validator (tier 2, requires oscal-cli)
	var cliPath string
	if *validateConstraints {
		cp, err := resolveOSCALCLI(*oscalCLIPath)
		if err != nil {
			log.Fatalf("constraint validation requested but oscal-cli not found: %v", err)
		}
		cliPath = cp
	}

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, *snapshot, *framework)
		if err != nil {
			log.Fatalf("generate assessment-results: %v", err)
		}
		path := fmt.Sprintf("%s/assessment-results.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportAssessmentResultsJSON(ar)
		}, validator, schemavalidate.KindAssessmentResults)
		fmt.Printf("Assessment results written to %s\n", path)
		if *validateConstraints {
			runConstraintValidation(cliPath, path, schemavalidate.KindAssessmentResults)
		}
		return
	}

	res, err := gen.GenerateAllArtifacts(ctx, *snapshot, *framework, nil)
	if err != nil {
		log.Fatalf("generate all: %v", err)
	}

	// Track written files for tier-2 constraint validation.
	var writtenFiles []struct {
		path string
		kind schemavalidate.ArtifactKind
	}

	if res.Catalog != nil {
		c := res.Catalog
		path := fmt.Sprintf("%s/catalog.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportCatalogJSON(c)
		}, validator, schemavalidate.KindCatalog)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindCatalog})
	}
	if res.Profile != nil {
		p := res.Profile
		path := fmt.Sprintf("%s/profile.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportProfileJSON(p)
		}, validator, schemavalidate.KindProfile)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindProfile})
	}
	if res.SSP != nil {
		s := res.SSP
		path := fmt.Sprintf("%s/ssp.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportSSPJSON(s)
		}, validator, schemavalidate.KindSSP)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindSSP})
	}
	if res.ComponentDefinition != nil {
		c := res.ComponentDefinition
		path := fmt.Sprintf("%s/component-definition.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportComponentDefinitionJSON(c)
		}, validator, schemavalidate.KindComponentDefinition)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindComponentDefinition})
	}
	if res.AssessmentPlan != nil {
		a := res.AssessmentPlan
		path := fmt.Sprintf("%s/assessment-plan.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportAssessmentPlanJSON(a)
		}, validator, schemavalidate.KindAssessmentPlan)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindAssessmentPlan})
	}
	if res.AssessmentResults != nil {
		a := res.AssessmentResults
		path := fmt.Sprintf("%s/assessment-results.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportAssessmentResultsJSON(a)
		}, validator, schemavalidate.KindAssessmentResults)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindAssessmentResults})
	}
	if res.POAM != nil {
		p := res.POAM
		path := fmt.Sprintf("%s/poam.json", *outDir)
		writeOSCAL(path, func() ([]byte, error) {
			return oscal.ExportPOAMJSON(p)
		}, validator, schemavalidate.KindPOAM)
		writtenFiles = append(writtenFiles, struct {
			path string
			kind schemavalidate.ArtifactKind
		}{path, schemavalidate.KindPOAM})
	}
	if len(res.Mappings) > 0 {
		mappingsData, err := json.MarshalIndent(struct{ Maps []*mappingv1.Map }{Maps: res.Mappings}, "", "  ")
		if err != nil {
			log.Fatalf("marshal mappings: %v", err)
		}
		if err := os.WriteFile(fmt.Sprintf("%s/mappings.json", *outDir), mappingsData, 0600); err != nil {
			log.Fatalf("write mappings: %v", err)
		}
	}

	fmt.Printf("OSCAL artifacts written to %s\n", *outDir)

	// Tier 2: Metaschema constraint validation via oscal-cli.
	if *validateConstraints {
		for _, wf := range writtenFiles {
			runConstraintValidation(cliPath, wf.path, wf.kind)
		}
		fmt.Println("All artifacts passed Metaschema constraint validation")
	}
}

// writeOSCAL writes OSCAL-compliant JSON to the given path. If a validator
// and kind are provided, the artifact is schema-validated and the command
// fails on non-compliance.
func writeOSCAL(path string, marshal func() ([]byte, error), validator *schemavalidate.Validator, kind schemavalidate.ArtifactKind) {
	b, err := marshal()
	if err != nil {
		log.Fatalf("marshal %s: %v", path, err)
	}
	if validator != nil {
		if err := validator.Validate(b, kind); err != nil {
			log.Fatalf("schema validation failed for %s: %v", path, err)
		}
	}
	// #nosec G304,G306 -- path is the explicit output destination selected by the CLI operator.
	if err := os.WriteFile(path, b, 0600); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}

// kindToCLIModel maps an ArtifactKind to the oscal-cli subcommand name.
var kindToCLIModel = map[schemavalidate.ArtifactKind]string{
	schemavalidate.KindCatalog:             "catalog",
	schemavalidate.KindProfile:             "profile",
	schemavalidate.KindSSP:                 "ssp",
	schemavalidate.KindComponentDefinition: "component-definition",
	schemavalidate.KindAssessmentPlan:      "ap",
	schemavalidate.KindAssessmentResults:   "ar",
	schemavalidate.KindPOAM:                "poam",
}

// runConstraintValidation runs oscal-cli <model> validate on the given file.
// It fails the command (log.Fatal) on any constraint violation.
func runConstraintValidation(cliPath, filePath string, kind schemavalidate.ArtifactKind) {
	model, ok := kindToCLIModel[kind]
	if !ok {
		log.Printf("warning: no oscal-cli subcommand for kind %s, skipping constraint validation", kind.Name)
		return
	}
	// #nosec G204 -- cliPath is resolved from the explicit flag or PATH by resolveOSCALCLI.
	cmd := exec.Command(cliPath, model, "validate", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Metaschema constraint validation failed for %s:\n%s", filePath, string(output))
	}
	fmt.Printf("  constraint-valid: %s\n", filePath)
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
		return "", fmt.Errorf("oscal-cli not found in PATH: %w", err)
	}
	return path, nil
}
