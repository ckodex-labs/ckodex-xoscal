package embedding

import (
	"context"
	"fmt"
	"sort"
)

// HybridSearcher merges vector similarity results with text search results
// using Reciprocal Rank Fusion (RRF). It is designed to work with any
// VectorStore implementation and a pluggable Generator.
type HybridSearcher struct {
	store     VectorStore
	generator Generator
}

// NewHybridSearcher creates a HybridSearcher that delegates text search to
// the provided VectorStore and uses generator for query embeddings.
func NewHybridSearcher(store VectorStore, generator Generator) *HybridSearcher {
	return &HybridSearcher{store: store, generator: generator}
}

// Search performs hybrid search: vector similarity + text search merged via RRF.
// It embeds the query, runs vector search against the store, runs text search,
// and fuses the two result lists. The k parameter is the number of results
// requested from each sub-search before fusion.
func (h *HybridSearcher) Search(ctx context.Context, query string, framework string, k int) ([]SearchResult, error) {
	if h.generator == nil {
		return nil, fmt.Errorf("hybrid search requires an embedding generator")
	}

	// Embed the query
	vectors, err := h.generator.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}
	queryVec := vectors[0]

	// Sub-search 1: vector similarity via store (if store implements vector search)
	vecResults, _ := h.vectorSearch(ctx, queryVec, framework, k)

	// Sub-search 2: text search via store
	textResults, err := h.store.Search(ctx, query, framework, k)
	if err != nil {
		return nil, fmt.Errorf("text search: %w", err)
	}

	// Merge with RRF (k=60 is the standard RRF constant)
	merged := reciprocalRankFusion(vecResults, textResults, 60)
	return merged, nil
}

// vectorSearch attempts vector similarity search. It first tries the store's
// native vector search if available; otherwise falls back to loading
// embeddings from the store and computing cosine similarity in memory.
// This is a placeholder until LanceDB Go SDK implements VectorSearch.
func (h *HybridSearcher) vectorSearch(ctx context.Context, queryVec []float32, framework string, k int) ([]SearchResult, error) {
	// TODO: call store.VectorSearch when LanceDB Go SDK exposes it.
	// For now, return empty so RRF degrades gracefully to text-only.
	return nil, nil
}

// reciprocalRankFusion merges two ranked lists using Reciprocal Rank Fusion.
// Score = sum(1 / (k + rank)) for each list where the item appears.
func reciprocalRankFusion(a, b []SearchResult, k int) []SearchResult {
	type scoreEntry struct {
		uuid      string
		modelType string
		score     float64
	}

	scores := make(map[string]scoreEntry)

	rankMap := func(results []SearchResult) map[string]int {
		m := make(map[string]int)
		for i, r := range results {
			if _, exists := m[r.UUID]; !exists {
				m[r.UUID] = i + 1 // 1-based rank
			}
		}
		return m
	}

	aRanks := rankMap(a)
	bRanks := rankMap(b)

	// Collect all unique UUIDs
	allUUIDs := make(map[string]struct{})
	for _, r := range a {
		allUUIDs[r.UUID] = struct{}{}
	}
	for _, r := range b {
		allUUIDs[r.UUID] = struct{}{}
	}

	for uuid := range allUUIDs {
		var total float64
		var modelType string
		if r, ok := aRanks[uuid]; ok {
			total += 1.0 / (float64(k) + float64(r))
			if modelType == "" {
				modelType = findModelType(a, uuid)
			}
		}
		if r, ok := bRanks[uuid]; ok {
			total += 1.0 / (float64(k) + float64(r))
			if modelType == "" {
				modelType = findModelType(b, uuid)
			}
		}
		scores[uuid] = scoreEntry{uuid: uuid, modelType: modelType, score: total}
	}

	// Sort by score descending
	var out []scoreEntry
	for _, e := range scores {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].score > out[j].score
	})

	var results []SearchResult
	for _, e := range out {
		results = append(results, SearchResult{
			UUID:      e.uuid,
			ModelType: e.modelType,
			Score:     e.score,
		})
	}
	return results
}

func findModelType(results []SearchResult, uuid string) string {
	for _, r := range results {
		if r.UUID == uuid {
			return r.ModelType
		}
	}
	return ""
}
