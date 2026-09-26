package embedding

import (
	"context"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

type mockGenerator struct {
	vectorMap map[string][]float32
}

func (m *mockGenerator) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		if v, ok := m.vectorMap[t]; ok {
			out[i] = v
		} else {
			out[i] = []float32{0.5, 0.5, 0.0}
		}
	}
	return out, nil
}

func TestHybridSearcher_Search(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open sqlite vector store: %v", err)
	}
	defer store.Close()

	gen := &mockGenerator{
		vectorMap: map[string][]float32{
			"quantum cryptography": {1.0, 0.0, 0.0},
		},
	}

	searcher := NewHybridSearcher(store, gen)

	// Doc 1: Vector match [1.0, 0.0, 0.0] but text mentions "post-quantum algorithms"
	doc1 := Document{
		UUID:      "urn:xoscal:entity:1",
		ModelType: "control",
		Framework: "nist-800-53",
		Content:   "Implementation of post-quantum algorithms and security controls",
		Embedding: []float32{1.0, 0.0, 0.0},
	}
	// Doc 2: Text match contains "cryptography" but vector is orthogonal
	doc2 := Document{
		UUID:      "urn:xoscal:entity:2",
		ModelType: "control",
		Framework: "nist-800-53",
		Content:   "General cryptography guidelines and key management",
		Embedding: []float32{0.0, 1.0, 0.0},
	}

	for _, doc := range []Document{doc1, doc2} {
		if err := store.Index(ctx, doc); err != nil {
			t.Fatalf("index %s: %v", doc.UUID, err)
		}
	}

	results, err := searcher.Search(ctx, "cryptography", "nist-800-53", 5)
	if err != nil {
		t.Fatalf("hybrid search: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected hybrid search results, got none")
	}

	// Verify both UUIDs are present in fused results
	found := make(map[string]bool)
	for _, r := range results {
		found[r.UUID] = true
		if r.Score <= 0 {
			t.Errorf("expected positive fused score, got %f", r.Score)
		}
	}
	if !found["urn:xoscal:entity:1"] || !found["urn:xoscal:entity:2"] {
		t.Errorf("expected both doc1 and doc2 in hybrid search results, got: %+v", results)
	}
}
