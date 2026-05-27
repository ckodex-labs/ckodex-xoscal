package embedding

import (
	"context"
	"fmt"
)

// Document represents an OSCAL artifact for embedding.
type Document struct {
	UUID      string
	ModelType string // e.g. "catalog", "profile", "control"
	Framework string // e.g. "eu-ai-act", "iso-42001"
	Title     string
	Content   string    // normalized text content
	Embedding []float32 // optional dense vector embedding (Path B)
}

// SearchResult from a vector similarity query.
type SearchResult struct {
	UUID      string
	ModelType string
	Score     float64
}

// Client is the interface for vector database operations.
type Client interface {
	Index(ctx context.Context, docs []Document) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
}

// NoopClient is a placeholder implementation.
type NoopClient struct{}

func (c *NoopClient) Index(ctx context.Context, docs []Document) error { return nil }
func (c *NoopClient) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	return nil, fmt.Errorf("NoopClient.Search not yet implemented")
}
