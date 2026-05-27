package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func main() {
	var (
		dsn       = flag.String("dsn", "oscal.db", "SQLite DSN")
		input     = flag.String("input", "", "Input JSON file path")
		framework = flag.String("framework", "eu-ai-act", "Framework identifier")
	)
	flag.Parse()

	if *input == "" {
		log.Fatal("-input is required")
	}

	raw, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("read input: %v", err)
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	rec := reconciler.NewReconciler(store)
	pipeline := ingestion.NewPipeline(
		&ingestion.EUAIActParser{},
		ingestion.NewNormalizer(*framework),
		rec,
	)

	res, err := pipeline.Run(context.Background(), raw)
	if err != nil {
		log.Fatalf("pipeline: %v", err)
	}

	fmt.Printf("Ingested %d requirements, %d entities, %d conflicts\n",
		len(res.Requirements), len(res.Entities), len(res.Conflicts))
	for _, c := range res.Conflicts {
		fmt.Printf("  Conflict: %s (%s)\n", c.Description, c.Type)
	}
}
