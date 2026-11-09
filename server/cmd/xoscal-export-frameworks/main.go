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

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
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
	// #nosec G304 -- manifestPath is the explicit operator-selected input manifest.
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
	reg := frameworks.DefaultRegistry()

	var fetchFilters []string
	filenameToManifest := make(map[string]portal.Framework)
	yamlRefIDs := make(map[string]bool)
	for _, fw := range m.Frameworks {
		if !strings.HasSuffix(fw.Filename, ".yaml") && reg.Has(fw.RefID) {
			continue
		}
		yamlRefIDs[fw.RefID] = true
		fetchFilters = append(fetchFilters, fw.RefID)
		if fw.Filename != "" && strings.HasSuffix(fw.Filename, ".yaml") {
			fetchFilters = append(fetchFilters, fw.Filename, strings.TrimSuffix(fw.Filename, ".yaml"))
			filenameToManifest[fw.Filename] = fw
		}
	}

	if len(fetchFilters) > 0 {
		gh := fetcher.NewGitHubFetcher(owner, repo, path)
		rec := reconciler.NewReconciler(store)
		bulk := ingestion.NewBulkIngestor(gh, rec, store)

		files, err := gh.Fetch(ctx, strings.Join(fetchFilters, ","))
		if err != nil {
			return fmt.Errorf("fetch frameworks: %w", err)
		}
		for i := range files {
			if mf, ok := filenameToManifest[files[i].Filename]; ok {
				files[i].RefID = mf.RefID
				files[i].Jurisdiction = mf.Jurisdiction
			}
		}
		if _, err := bulk.IngestFiles(ctx, files); err != nil {
			return fmt.Errorf("bulk ingest: %w", err)
		}
	}
	if _, err := store.CreateSnapshot(ctx, "portal"); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	gen := oscal.NewGenerator(store)
	for _, refID := range m.RefIDs() {
		if !yamlRefIDs[refID] && reg.Has(refID) {
			if err := reg.ExportAuthoritative(outDir, refID); err != nil {
				return fmt.Errorf("export %s: %w", refID, err)
			}
			continue
		}

		res, gErr := gen.GenerateAllArtifacts(ctx, "portal", refID, nil)
		if gErr != nil {
			return fmt.Errorf("generate %s: %w", refID, gErr)
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
		// #nosec G703 -- outDir is an explicit local destination selected by the CLI operator; refID comes from the trusted manifest.
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		catPath := filepath.Join(dir, "catalog.json")
		// #nosec G703,G304,G306 -- catPath is beneath the explicit operator-selected output directory.
		if err := os.WriteFile(catPath, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", catPath, err)
		}
		sum := sha256.Sum256(data)
		// #nosec G703,G304,G306 -- the sidecar path is derived from the same bounded output path.
		if err := os.WriteFile(catPath+".sha256", []byte("sha256:"+hex.EncodeToString(sum[:])+"\n"), 0o600); err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}
	}
	return nil
}
