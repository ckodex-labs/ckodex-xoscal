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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func main() {
	var (
		dsn       = flag.String("dsn", "oscal.db", "SQLite DSN")
		snapshot  = flag.String("snapshot", "", "Snapshot name")
		framework = flag.String("framework", "eu-ai-act", "Framework identifier")
		outDir    = flag.String("out", "./oscal-out", "Output directory")
		arOnly    = flag.Bool("assessment-results-only", false, "Emit only assessment-results (useful for CI)")
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

	if *arOnly {
		ar, err := gen.GenerateAssessmentResults(ctx, *snapshot, *framework)
		if err != nil {
			log.Fatalf("generate assessment-results: %v", err)
		}
		writeJSON(fmt.Sprintf("%s/assessment-results.json", *outDir), ar)
		fmt.Printf("Assessment results written to %s/assessment-results.json\n", *outDir)
		return
	}

	res, err := gen.GenerateAllArtifacts(ctx, *snapshot, *framework, nil)
	if err != nil {
		log.Fatalf("generate all: %v", err)
	}

	if res.Catalog != nil {
		writeJSON(fmt.Sprintf("%s/catalog.json", *outDir), res.Catalog)
	}
	if res.Profile != nil {
		writeJSON(fmt.Sprintf("%s/profile.json", *outDir), res.Profile)
	}
	if res.SSP != nil {
		writeJSON(fmt.Sprintf("%s/ssp.json", *outDir), res.SSP)
	}
	if res.ComponentDefinition != nil {
		writeJSON(fmt.Sprintf("%s/component-definition.json", *outDir), res.ComponentDefinition)
	}
	if res.AssessmentPlan != nil {
		writeJSON(fmt.Sprintf("%s/assessment-plan.json", *outDir), res.AssessmentPlan)
	}
	if res.AssessmentResults != nil {
		writeJSON(fmt.Sprintf("%s/assessment-results.json", *outDir), res.AssessmentResults)
	}
	if res.POAM != nil {
		writeJSON(fmt.Sprintf("%s/poam.json", *outDir), res.POAM)
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

func writeJSON(path string, msg proto.Message) {
	b, err := protojson.MarshalOptions{Multiline: true}.Marshal(msg)
	if err != nil {
		log.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}
