package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"
)

// Generator produces dense vector embeddings from text.
type Generator interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// MockGenerator returns deterministic pseudo-embeddings for testing.
type MockGenerator struct{}

func (m *MockGenerator) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = deterministicVector(t, 384)
	}
	return out, nil
}

func deterministicVector(text string, dim int) []float32 {
	v := make([]float32, dim)
	var seed uint32 = 0
	for _, c := range text {
		seed = seed*31 + uint32(c)
	}
	for i := 0; i < dim; i++ {
		seed = seed*1103515245 + 12345
		value := float64(seed >> 16)
		if seed&0x80000000 != 0 {
			value -= 65536
		}
		v[i] = float32(value / 32768.0)
	}
	return v
}

// OpenAIGenerator calls the OpenAI Embeddings API.
type OpenAIGenerator struct {
	APIKey  string
	Model   string // e.g. "text-embedding-3-small"
	BaseURL string // defaults to https://api.openai.com/v1
	client  *http.Client
}

// NewOpenAIGenerator creates an OpenAIGenerator from configuration.
// OPENAI_API_KEY env var takes precedence over cfg.OpenAIKey.
func NewOpenAIGenerator(apiKey, model, baseURL string) *OpenAIGenerator {
	key := apiKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIGenerator{
		APIKey:  key,
		Model:   model,
		BaseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type openaiEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openaiEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (o *OpenAIGenerator) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if o.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured (set OPENAI_API_KEY or config vector.openai_api_key)")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := openaiEmbeddingRequest{
		Model: o.Model,
		Input: texts,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embeddings API returned %d", resp.StatusCode)
	}

	var result openaiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	out := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < 0 || d.Index >= len(out) {
			continue
		}
		vec := make([]float32, len(d.Embedding))
		for i, v := range d.Embedding {
			vec[i] = float32(v)
		}
		out[d.Index] = vec
	}
	// Check for missing embeddings
	for i, v := range out {
		if v == nil {
			return nil, fmt.Errorf("missing embedding for input %d", i)
		}
	}
	return out, nil
}
