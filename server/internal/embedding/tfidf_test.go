package embedding

import (
	"context"
	"testing"
)

func TestTFIDFClient_Search(t *testing.T) {
	ctx := context.Background()
	client := NewTFIDFClient()

	docs := []Document{
		{UUID: "doc1", ModelType: "requirement", Content: "high risk artificial intelligence system must have logging"},
		{UUID: "doc2", ModelType: "requirement", Content: "low risk ai system needs documentation"},
		{UUID: "doc3", ModelType: "control", Content: "logging and monitoring controls for systems"},
	}

	if err := client.Index(ctx, docs); err != nil {
		t.Fatalf("index: %v", err)
	}

	results, err := client.Search(ctx, "logging risk system", 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected search results, got none")
	}

	// doc1 should rank highest because it matches "logging", "risk", and "system".
	if results[0].UUID != "doc1" {
		t.Logf("top result was %s (score %.4f), expected doc1", results[0].UUID, results[0].Score)
	}

	for _, r := range results {
		if r.Score <= 0 {
			t.Fatalf("expected positive score for %s, got %f", r.UUID, r.Score)
		}
	}
}

func TestTFIDFClient_EmptyCorpus(t *testing.T) {
	ctx := context.Background()
	client := NewTFIDFClient()
	results, err := client.Search(ctx, "test", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty corpus, got %d", len(results))
	}
}
