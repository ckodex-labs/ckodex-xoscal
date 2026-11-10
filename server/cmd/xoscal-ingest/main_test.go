package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func TestIngestRequirementsIntoSQLite(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "ingest.db")
	inPath := filepath.Join(dir, "requirements.json")

	payload := `[
  {
    "id": "art-9-1",
    "article": "Article 9(1)",
    "title": "Risk management system",
    "description": "A risk management system shall be established, implemented, documented and maintained.",
    "category": "Risk Management",
    "role": "provider",
    "risk_level": "high"
  },
  {
    "id": "art-10-1",
    "article": "Article 10(1)",
    "title": "Data and data governance",
    "description": "High-risk AI systems shall be developed on the basis of training, validation and testing data sets.",
    "category": "Data Governance",
    "role": "provider",
    "risk_level": "high"
  }
]`
	if err := os.WriteFile(inPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"-dsn", dsn, "-input", inPath, "-framework", "eu-ai-act"}, &out); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(out.String(), "Ingested 2 requirements") {
		t.Fatalf("unexpected output: %s", out.String())
	}

	store, err := kg.NewSQLiteStore(dsn, dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer func() { _ = store.Close() }()

	reqs, err := store.ListEntities(context.Background(), "reg:Requirement", kg.EntityStatusActive)
	if err != nil {
		t.Fatalf("list requirements: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 persisted requirements, got %d", len(reqs))
	}

	if err := run([]string{"-dsn", dsn}, &out); err == nil {
		t.Fatal("expected error when -input is omitted")
	}
}
