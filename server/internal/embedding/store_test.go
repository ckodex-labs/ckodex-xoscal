package embedding

import (
	"context"
	"math"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func TestVectorSerialization(t *testing.T) {
	orig := []float32{0.0, 1.5, -3.25, 42.125, -0.001}
	blob := encodeVector(orig)
	if len(blob) != len(orig)*4 {
		t.Fatalf("expected blob len %d, got %d", len(orig)*4, len(blob))
	}
	decoded := decodeVector(blob)
	if len(decoded) != len(orig) {
		t.Fatalf("expected decoded len %d, got %d", len(orig), len(decoded))
	}
	for i := range orig {
		if math.Abs(float64(orig[i]-decoded[i])) > 1e-6 {
			t.Errorf("index %d: expected %f, got %f", i, orig[i], decoded[i])
		}
	}
}

func TestSQLiteVectorStore_VectorSearch(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open sqlite vector store: %v", err)
	}
	defer store.Close()

	// Target query vector: points towards [1.0, 0.0, 0.0]
	queryVec := []float32{1.0, 0.0, 0.0}

	// Doc A: exact match [1.0, 0.0, 0.0] -> similarity 1.0
	docA := Document{
		UUID:      "urn:xoscal:entity:a",
		ModelType: "control",
		Framework: "nist-800-53",
		Content:   "Access control policy and procedures",
		Embedding: []float32{1.0, 0.0, 0.0},
	}
	// Doc B: partial match [0.7071, 0.7071, 0.0] -> similarity ~0.707
	docB := Document{
		UUID:      "urn:xoscal:entity:b",
		ModelType: "control",
		Framework: "nist-800-53",
		Content:   "Account management mechanism",
		Embedding: []float32{0.7071, 0.7071, 0.0},
	}
	// Doc C: orthogonal vector [0.0, 1.0, 0.0] -> similarity 0.0
	docC := Document{
		UUID:      "urn:xoscal:entity:c",
		ModelType: "control",
		Framework: "nist-800-53",
		Content:   "Cryptographic key establishment",
		Embedding: []float32{0.0, 1.0, 0.0},
	}
	// Doc D: identical to Doc A but different framework "iso-27001"
	docD := Document{
		UUID:      "urn:xoscal:entity:d",
		ModelType: "control",
		Framework: "iso-27001",
		Content:   "Information security policy",
		Embedding: []float32{1.0, 0.0, 0.0},
	}

	for _, doc := range []Document{docA, docB, docC, docD} {
		if err := store.Index(ctx, doc); err != nil {
			t.Fatalf("index %s: %v", doc.UUID, err)
		}
	}

	// 1. Search all frameworks, topK=5
	res, err := store.VectorSearch(ctx, queryVec, "", 5)
	if err != nil {
		t.Fatalf("vector search all: %v", err)
	}
	if len(res) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(res))
	}
	// Top results should have similarity close to 1.0
	if math.Abs(res[0].Score-1.0) > 1e-4 {
		t.Errorf("expected top result score ~1.0, got %f", res[0].Score)
	}

	// 2. Filter by framework "nist-800-53"
	resNist, err := store.VectorSearch(ctx, queryVec, "nist-800-53", 2)
	if err != nil {
		t.Fatalf("vector search nist: %v", err)
	}
	if len(resNist) != 2 {
		t.Fatalf("expected exactly 2 results, got %d", len(resNist))
	}
	if resNist[0].UUID != "urn:xoscal:entity:a" {
		t.Errorf("expected first result to be docA, got %s", resNist[0].UUID)
	}
	if resNist[1].UUID != "urn:xoscal:entity:b" {
		t.Errorf("expected second result to be docB, got %s", resNist[1].UUID)
	}
	if resNist[0].Score <= resNist[1].Score {
		t.Errorf("expected docA score (%f) > docB score (%f)", resNist[0].Score, resNist[1].Score)
	}
}
