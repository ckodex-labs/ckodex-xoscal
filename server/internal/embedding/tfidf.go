package embedding

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
)

// TFIDFClient implements a pure-Go TF-IDF vectorizer for semantic search.
type TFIDFClient struct {
	mu         sync.RWMutex
	docs       []Document
	vectors    map[string][]float64 // doc UUID -> tfidf vector
	vocabulary map[string]int       // term -> index
	idf        []float64
}

// NewTFIDFClient creates an empty TF-IDF client.
func NewTFIDFClient() *TFIDFClient {
	return &TFIDFClient{
		vectors:    make(map[string][]float64),
		vocabulary: make(map[string]int),
	}
}

// Index rebuilds the TF-IDF corpus from the provided documents.
func (c *TFIDFClient) Index(ctx context.Context, docs []Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.docs = append(c.docs, docs...)
	return c.rebuild()
}

func (c *TFIDFClient) rebuild() error {
	// Build vocabulary.
	termFreqs := make(map[string]int) // term -> number of docs containing it
	docTerms := make([]map[string]int, len(c.docs))

	for i, doc := range c.docs {
		terms := tokenize(doc.Content)
		seen := make(map[string]bool)
		freq := make(map[string]int)
		for _, t := range terms {
			freq[t]++
			seen[t] = true
		}
		for t := range seen {
			termFreqs[t]++
		}
		docTerms[i] = freq
	}

	// Assign indices.
	c.vocabulary = make(map[string]int)
	c.idf = make([]float64, 0, len(termFreqs))
	idx := 0
	for term, df := range termFreqs {
		c.vocabulary[term] = idx
		idf := math.Log1p(float64(len(c.docs)) / float64(df))
		c.idf = append(c.idf, idf)
		idx++
	}

	// Build vectors.
	c.vectors = make(map[string][]float64)
	for i, doc := range c.docs {
		vec := make([]float64, len(c.vocabulary))
		maxFreq := 0
		for _, f := range docTerms[i] {
			if f > maxFreq {
				maxFreq = f
			}
		}
		for term, freq := range docTerms[i] {
			j := c.vocabulary[term]
			tf := float64(freq) / float64(maxFreq)
			vec[j] = tf * c.idf[j]
		}
		normalize(vec)
		c.vectors[doc.UUID] = vec
	}

	return nil
}

// Search ranks documents by cosine similarity to the query.
func (c *TFIDFClient) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.docs) == 0 {
		return nil, nil
	}

	qTerms := tokenize(query)
	qFreq := make(map[string]int)
	maxFreq := 0
	for _, t := range qTerms {
		qFreq[t]++
		if qFreq[t] > maxFreq {
			maxFreq = qFreq[t]
		}
	}

	qVec := make([]float64, len(c.vocabulary))
	for term, freq := range qFreq {
		if j, ok := c.vocabulary[term]; ok {
			tf := float64(freq) / float64(maxFreq)
			qVec[j] = tf * c.idf[j]
		}
	}
	normalize(qVec)

	var results []SearchResult
	for _, doc := range c.docs {
		vec, ok := c.vectors[doc.UUID]
		if !ok {
			continue
		}
		score := dot(qVec, vec)
		if score > 0 {
			results = append(results, SearchResult{
				UUID:      doc.UUID,
				ModelType: doc.ModelType,
				Score:     score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func tokenize(s string) []string {
	var terms []string
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			terms = append(terms, string(r))
		} else if len(terms) > 0 && terms[len(terms)-1] != "" {
			terms = append(terms, "")
		}
	}
	// Compact runes into words.
	var words []string
	var buf strings.Builder
	for _, term := range terms {
		if term == "" {
			if buf.Len() > 0 {
				words = append(words, buf.String())
				buf.Reset()
			}
		} else {
			buf.WriteString(term)
		}
	}
	if buf.Len() > 0 {
		words = append(words, buf.String())
	}
	return words
}

func normalize(v []float64) {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	if sum == 0 {
		return
	}
	n := math.Sqrt(sum)
	for i := range v {
		v[i] /= n
	}
}

func dot(a, b []float64) float64 {
	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}
