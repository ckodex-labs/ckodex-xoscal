package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIGenerator_Embed(t *testing.T) {
	wantVectors := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("path = %s, want /v1/embeddings", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Authorization = %s, want Bearer test-key", auth)
		}
		var req openaiEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "text-embedding-3-small" {
			t.Errorf("model = %s, want text-embedding-3-small", req.Model)
		}
		if len(req.Input) != 2 {
			t.Errorf("input len = %d, want 2", len(req.Input))
		}
		resp := openaiEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float64 `json:"embedding"`
			}{
				{Object: "embedding", Index: 0, Embedding: wantVectors[0]},
				{Object: "embedding", Index: 1, Embedding: wantVectors[1]},
			},
			Model: "text-embedding-3-small",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	gen := NewOpenAIGenerator("test-key", "text-embedding-3-small", server.URL+"/v1")
	vecs, err := gen.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}
	if len(vecs) != 2 {
		t.Fatalf("len(vecs) = %d, want 2", len(vecs))
	}
	for i, v := range vecs {
		if len(v) != 3 {
			t.Fatalf("vec[%d] len = %d, want 3", i, len(v))
		}
		for j, val := range v {
			if float32(wantVectors[i][j]) != val {
				t.Errorf("vec[%d][%d] = %f, want %f", i, j, val, wantVectors[i][j])
			}
		}
	}
}

func TestOpenAIGenerator_EmbedMissingKey(t *testing.T) {
	gen := NewOpenAIGenerator("", "text-embedding-3-small", "")
	_, err := gen.Embed(context.Background(), []string{"test"})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestOpenAIGenerator_EmbedEmptyInput(t *testing.T) {
	gen := NewOpenAIGenerator("test-key", "text-embedding-3-small", "")
	vecs, err := gen.Embed(context.Background(), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vecs != nil {
		t.Fatalf("expected nil for empty input, got %v", vecs)
	}
}
