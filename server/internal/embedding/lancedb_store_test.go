package embedding

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/config"
)

func skipIfNoLanceDB(t *testing.T) {
	// When LanceDB is not compiled in, NewLanceDBVectorStore returns an error immediately.
	// We detect this by attempting to create a temp store.
	cfg := config.Vector{Backend: "lancedb", URI: t.TempDir()}
	_, err := NewLanceDBVectorStore(cfg)
	if err != nil && err.Error() == "lancedb support not compiled in; build with -tags lancedb (requires CGO + Rust toolchain)" {
		t.Skip("lancedb support not compiled in; skipping")
	}
	// Clean up the temp dir if LanceDB is compiled in.
	_ = os.RemoveAll(cfg.URI)
}

func TestLanceDBVectorStore_IndexAndSearch(t *testing.T) {
	skipIfNoLanceDB(t)

	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "lancedb_test")
	cfg := config.Vector{Backend: "lancedb", URI: dir}

	store, err := NewLanceDBVectorStore(cfg)
	if err != nil {
		t.Fatalf("new lancedb store: %v", err)
	}
	defer func() {
		_ = store.Close()
		_ = os.RemoveAll(dir)
	}()

	docs := []Document{
		{UUID: "urn:1", ModelType: "control", Framework: "eu-ai-act", Title: "Risk Management", Content: "Implement a risk management system for AI."},
		{UUID: "urn:2", ModelType: "control", Framework: "eu-ai-act", Title: "Data Governance", Content: "Ensure training data quality and governance."},
		{UUID: "urn:3", ModelType: "control", Framework: "iso-42001", Title: "AI Management", Content: "Establish AI management system."},
	}

	for _, d := range docs {
		if err := store.Index(ctx, d); err != nil {
			t.Fatalf("index %s: %v", d.UUID, err)
		}
	}

	// Search within eu-ai-act framework
	res, err := store.Search(ctx, "risk management", "eu-ai-act", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected results, got none")
	}
	foundURN1 := false
	for _, r := range res {
		if r.UUID == "urn:1" {
			foundURN1 = true
		}
		if r.ModelType != "control" {
			t.Errorf("expected model_type control, got %s", r.ModelType)
		}
	}
	if !foundURN1 {
		t.Errorf("expected urn:1 in results")
	}

	// Search across all frameworks
	resAll, err := store.Search(ctx, "AI", "", 10)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(resAll) == 0 {
		t.Fatalf("expected results for cross-framework search, got none")
	}

	// Search with no-match framework
	resNone, err := store.Search(ctx, "risk", "nist-800-53", 10)
	if err != nil {
		t.Fatalf("search none: %v", err)
	}
	if len(resNone) != 0 {
		t.Errorf("expected no results for nonexistent framework, got %d", len(resNone))
	}
}

func TestLanceDBVectorStore_CloseIdempotent(t *testing.T) {
	skipIfNoLanceDB(t)

	dir := filepath.Join(t.TempDir(), "lancedb_close")
	cfg := config.Vector{Backend: "lancedb", URI: dir}

	store, err := NewLanceDBVectorStore(cfg)
	if err != nil {
		t.Fatalf("new lancedb store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	_ = os.RemoveAll(dir)
}
