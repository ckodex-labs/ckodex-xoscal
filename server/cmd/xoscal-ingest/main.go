package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("xoscal-ingest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dsn := fs.String("dsn", "oscal.db", "SQLite DSN")
	input := fs.String("input", "", "Input JSON file path")
	framework := fs.String("framework", "eu-ai-act", "Framework identifier")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *input == "" {
		return errors.New("-input is required")
	}

	// #nosec G304 -- input is the explicit requirements JSON path selected by the CLI operator.
	raw, err := os.ReadFile(*input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	store, err := kg.NewSQLiteStore(*dsn, dbutil.PoolConfig{})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = store.Close() }()

	rec := reconciler.NewReconciler(store)
	pipeline := ingestion.NewPipeline(
		&ingestion.EUAIActParser{},
		ingestion.NewNormalizer(*framework),
		rec,
	)

	res, err := pipeline.Run(context.Background(), raw)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Ingested %d requirements, %d entities, %d conflicts\n",
		len(res.Requirements), len(res.Entities), len(res.Conflicts))
	for _, c := range res.Conflicts {
		_, _ = fmt.Fprintf(stdout, "  Conflict: %s (%s)\n", c.Description, c.Type)
	}
	return nil
}
