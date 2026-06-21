// Command xoscal-export-frameworks fetches the frameworks listed in the manifest,
// ingests them into a temporary KG, snapshots it, and emits one OSCAL catalog per
// framework with a sha256 sidecar.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/portal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func main() {
	manifestPath := flag.String("manifest", "data/frameworks/manifest.yaml", "Framework manifest path")
	outDir := flag.String("out", "./oscal-frameworks", "Output directory")
	dsn := flag.String("dsn", "portal-export.db", "Temp SQLite DSN")
	flag.Parse()

	if err := run(*manifestPath, *outDir, *dsn); err != nil {
		log.Fatal(err)
	}
}

func run(manifestPath, outDir, dsn string) error {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	m, err := portal.ParseManifest(raw)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}
	owner, repo, path, err := m.Source()
	if err != nil {
		return fmt.Errorf("derive source: %w", err)
	}

	store, err := kg.NewSQLiteStore(dsn, dbutil.PoolConfig{})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()
	gh := fetcher.NewGitHubFetcher(owner, repo, path)
	rec := reconciler.NewReconciler(store)
	bulk := ingestion.NewBulkIngestor(gh, rec, store)

	if _, err := bulk.Run(ctx, strings.Join(m.RefIDs(), ",")); err != nil {
		return fmt.Errorf("bulk ingest: %w", err)
	}
	if _, err := store.CreateSnapshot(ctx, "portal"); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	gen := oscal.NewGenerator(store)
	for _, refID := range m.RefIDs() {
		res, err := gen.GenerateAllArtifacts(ctx, "portal", refID, nil)
		if err != nil {
			return fmt.Errorf("generate %s: %w", refID, err)
		}
		if res.Catalog == nil {
			log.Printf("warn: %s produced no catalog, skipping", refID)
			continue
		}
		data, err := oscal.ExportCatalogJSON(res.Catalog)
		if err != nil {
			return fmt.Errorf("export %s: %w", refID, err)
		}
		dir := filepath.Join(outDir, refID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		catPath := filepath.Join(dir, "catalog.json")
		if err := os.WriteFile(catPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", catPath, err)
		}
		sum := sha256.Sum256(data)
		if err := os.WriteFile(catPath+".sha256", []byte("sha256:"+hex.EncodeToString(sum[:])+"\n"), 0o644); err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}
	}
	return nil
}
