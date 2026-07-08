package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func main() {
	var (
		dsn       = flag.String("dsn", "oscal.db", "SQLite DSN")
		snapshot  = flag.String("snapshot", "", "Snapshot name")
		framework = flag.String("framework", "eu-ai-act", "Framework identifier")
		outDir    = flag.String("out", "./oscal-out", "Output directory")
		arOnly    = flag.Bool("assessment-results-only", false, "Emit only assessment-results (useful for CI)")
		validate  = flag.Bool("validate", false, "Schema-validate each artifact after generation; fail on non-compliance")
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

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	// Optional schema validator
	var validator *schemavalidate.Validator
	if *validate {
		v, err := schemavalidate.NewValidator()
		if err != nil {
			log.Fatalf("init schema validator: %v", err)
		}
		validator = v
	}

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, *snapshot, *framework)
		if err != nil {
			log.Fatalf("generate assessment-results: %v", err)
		}
		writeOSCAL(fmt.Sprintf("%s/assessment-results.json", *outDir), func() ([]byte, error) {
			return oscal.ExportAssessmentResultsJSON(ar)
		}, validator, schemavalidate.KindAssessmentResults)
		fmt.Printf("Assessment results written to %s/assessment-results.json\n", *outDir)
		return
	}

	res, err := gen.GenerateAllArtifacts(ctx, *snapshot, *framework, nil)
	if err != nil {
		log.Fatalf("generate all: %v", err)
	}

	if res.Catalog != nil {
		c := res.Catalog
		writeOSCAL(fmt.Sprintf("%s/catalog.json", *outDir), func() ([]byte, error) {
			return oscal.ExportCatalogJSON(c)
		}, validator, schemavalidate.KindCatalog)
	}
	if res.Profile != nil {
		p := res.Profile
		writeOSCAL(fmt.Sprintf("%s/profile.json", *outDir), func() ([]byte, error) {
			return oscal.ExportProfileJSON(p)
		}, validator, schemavalidate.KindProfile)
	}
	if res.SSP != nil {
		s := res.SSP
		writeOSCAL(fmt.Sprintf("%s/ssp.json", *outDir), func() ([]byte, error) {
			return oscal.ExportSSPJSON(s)
		}, validator, schemavalidate.KindSSP)
	}
	if res.ComponentDefinition != nil {
		c := res.ComponentDefinition
		writeOSCAL(fmt.Sprintf("%s/component-definition.json", *outDir), func() ([]byte, error) {
			return oscal.ExportComponentDefinitionJSON(c)
		}, validator, schemavalidate.KindComponentDefinition)
	}
	if res.AssessmentPlan != nil {
		a := res.AssessmentPlan
		writeOSCAL(fmt.Sprintf("%s/assessment-plan.json", *outDir), func() ([]byte, error) {
			return oscal.ExportAssessmentPlanJSON(a)
		}, validator, schemavalidate.KindAssessmentPlan)
	}
	if res.AssessmentResults != nil {
		a := res.AssessmentResults
		writeOSCAL(fmt.Sprintf("%s/assessment-results.json", *outDir), func() ([]byte, error) {
			return oscal.ExportAssessmentResultsJSON(a)
		}, validator, schemavalidate.KindAssessmentResults)
	}
	if res.POAM != nil {
		p := res.POAM
		writeOSCAL(fmt.Sprintf("%s/poam.json", *outDir), func() ([]byte, error) {
			return oscal.ExportPOAMJSON(p)
		}, validator, schemavalidate.KindPOAM)
	}
	if len(res.Mappings) > 0 {
		mappingsData, err := json.MarshalIndent(struct{ Maps []*mappingv1.Map }{Maps: res.Mappings}, "", "  ")
		if err != nil {
			log.Fatalf("marshal mappings: %v", err)
		}
		if err := os.WriteFile(fmt.Sprintf("%s/mappings.json", *outDir), mappingsData, 0644); err != nil {
			log.Fatalf("write mappings: %v", err)
		}
	}

	fmt.Printf("OSCAL artifacts written to %s\n", *outDir)
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
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}
