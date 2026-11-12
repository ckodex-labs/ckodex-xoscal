package embedding

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
