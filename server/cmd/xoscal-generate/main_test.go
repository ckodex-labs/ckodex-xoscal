package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func TestGenerateAllArtifactsAndSchemaValidate(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "gen.db")

	store, err := kg.NewSQLiteStore(dsn, dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	raw := []byte(`[
  {
    "id": "art-9-1",
    "article": "Article 9(1)",
    "title": "Risk management system",
    "description": "Establish and maintain a continuous risk management system.",
    "category": "Risk Management",
    "role": "provider",
    "risk_level": "high"
  }
]`)
	rec := reconciler.NewReconciler(store)
	pipe := ingestion.NewPipeline(&ingestion.EUAIActParser{}, ingestion.NewNormalizer("eu-ai-act"), rec)
	if _, err := pipe.Run(context.Background(), raw); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if _, err := store.CreateSnapshot(context.Background(), "v1"); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	_ = store.Close()

	arDir := filepath.Join(dir, "out-ar")
	var out bytes.Buffer
	if err := run([]string{"-dsn", dsn, "-snapshot", "v1", "-framework", "eu-ai-act", "-out", arDir, "-assessment-results-only", "-validate"}, &out); err != nil {
		t.Fatalf("ar-only generate failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(arDir, "assessment-results.json.sha256")); err != nil {
		t.Fatalf("missing assessment-results sidecar: %v", err)
	}

	fullDir := filepath.Join(dir, "out-full")
	out.Reset()
	if err := run([]string{"-db", dsn, "-snapshot", "v1", "-framework", "eu-ai-act", "-out", fullDir, "-validate"}, &out); err != nil {
		t.Fatalf("full generate failed: %v", err)
	}
	for _, expected := range []string{
		"catalog.json",
		"profile.json",
		"ssp.json",
		"component-definition.json",
		"assessment-plan.json",
		"assessment-results.json",
		"poam.json",
	} {
		if _, err := os.Stat(filepath.Join(fullDir, expected)); err != nil {
			t.Fatalf("missing generated artifact %s: %v", expected, err)
		}
		if _, err := os.Stat(filepath.Join(fullDir, expected+".sha256")); err != nil {
			t.Fatalf("missing sidecar for %s: %v", expected, err)
		}
	}

	if err := run([]string{"-dsn", dsn}, &out); err == nil {
		t.Fatal("expected error when -snapshot is missing")
	}
}
