//go:build lancedb

package embedding

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/lancedb/lancedb-go/pkg/contracts"
	"github.com/lancedb/lancedb-go/pkg/lancedb"
	"github.com/mchorfa/xoscal/server/internal/config"
)

// LanceDBVectorStore implements VectorStore using LanceDB vector search + metadata filtering.
type LanceDBVectorStore struct {
	conn      contracts.IConnection
	table     contracts.ITable
	generator Generator // optional; enables vector search when FTS is unavailable
}

// NewLanceDBVectorStore opens (or creates) a LanceDB table for OSCAL documents.
func NewLanceDBVectorStore(cfg config.Vector) (VectorStore, error) {
	ctx := context.Background()

	opts := &contracts.ConnectionOptions{}
	if cfg.S3Region != "" {
		opts.StorageOptions = &contracts.StorageOptions{
			S3Config: &contracts.S3Config{
				Region:          &cfg.S3Region,
				AccessKeyID:     &cfg.S3KeyID,
				SecretAccessKey: &cfg.S3Secret,
			},
		}
	}

	db, err := lancedb.Connect(ctx, cfg.URI, opts)
	if err != nil {
		return nil, fmt.Errorf("lancedb connect: %w", err)
	}

	tableName := "oscal_documents"
	table, err := db.OpenTable(ctx, tableName)
	if err != nil {
		// Table does not exist — create with schema.
		schema, schemaErr := buildSchema()
		if schemaErr != nil {
			db.Close()
			return nil, fmt.Errorf("lancedb schema: %w", schemaErr)
		}
		table, err = db.CreateTable(ctx, tableName, schema)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("lancedb create table: %w", err)
		}
		// Build FTS index on content for full-text search.
		if idxErr := table.CreateIndex(ctx, []string{"content"}, contracts.IndexTypeFts); idxErr != nil {
			table.Close()
			db.Close()
			return nil, fmt.Errorf("lancedb create fts index: %w", idxErr)
		}
		// Build BTree index on framework for fast metadata filtering.
		if idxErr := table.CreateIndex(ctx, []string{"framework"}, contracts.IndexTypeBTree); idxErr != nil {
			table.Close()
			db.Close()
			return nil, fmt.Errorf("lancedb create btree index: %w", idxErr)
		}
	}

	// Create OpenAI generator if configured; enables vector search.
	var gen Generator
	if cfg.OpenAIKey != "" || os.Getenv("OPENAI_API_KEY") != "" {
		gen = NewOpenAIGenerator(cfg.OpenAIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL)
	}

	return &LanceDBVectorStore{conn: db, table: table, generator: gen}, nil
}

func buildSchema() (contracts.ISchema, error) {
	return lancedb.NewSchemaBuilder().
		AddStringField("uuid", false).
		AddStringField("model_type", false).
		AddStringField("framework", false).
		AddStringField("title", true).
		AddStringField("content", false).
		AddVectorField("embedding", 384, contracts.VectorDataTypeFloat32, true). // optional; enables Path B vector search
		AddInt64Field("created_at", true).
		Build()
}

func (s *LanceDBVectorStore) Close() error {
	if s.table != nil {
		_ = s.table.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	return nil
}

func (s *LanceDBVectorStore) Index(ctx context.Context, doc Document) error {
	schema, err := buildSchema()
	if err != nil {
		return fmt.Errorf("schema: %w", err)
	}

	record, err := docToArrowRecord(schema, doc)
	if err != nil {
		return fmt.Errorf("convert document: %w", err)
	}
	defer record.Release()

	if err := s.table.AddRecords(ctx, []arrow.Record{record}, nil); err != nil {
		return fmt.Errorf("lancedb add: %w", err)
	}
	return nil
}

func (s *LanceDBVectorStore) Search(ctx context.Context, query string, framework string, topK int) ([]SearchResult, error) {
	if s.generator == nil {
		return nil, fmt.Errorf("LanceDB search requires an embedding generator; configure OPENAI_API_KEY or use sqlite backend")
	}

	// Generate query embedding for vector similarity search.
	embeddings, err := s.generator.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty query embedding")
	}
	queryVec := embeddings[0]

	var rows []map[string]interface{}
	if framework != "" {
		safe := strings.ReplaceAll(framework, "'", "''")
		filter := fmt.Sprintf("framework = '%s'", safe)
		rows, err = s.table.VectorSearchWithFilter(ctx, "embedding", queryVec, topK, filter)
	} else {
		rows, err = s.table.VectorSearch(ctx, "embedding", queryVec, topK)
	}
	if err != nil {
		return nil, fmt.Errorf("lancedb vector search: %w", err)
	}

	var out []SearchResult
	for _, r := range rows {
		uuidVal, _ := r["uuid"].(string)
		modelTypeVal, _ := r["model_type"].(string)
		score := 1.0
		if dist, ok := r["_distance"].(float32); ok {
			score = float64(1.0 - dist) // LanceDB returns distance; convert to similarity
		}
		out = append(out, SearchResult{
			UUID:      uuidVal,
			ModelType: modelTypeVal,
			Score:     score,
		})
	}
	return out, nil
}

func docToArrowRecord(schema contracts.ISchema, doc Document) (arrow.Record, error) {
	arrowSchema := schema.ToArrowSchema()
	pool := memory.NewGoAllocator()
	builder := array.NewRecordBuilder(pool, arrowSchema)
	defer builder.Release()

	for i, field := range arrowSchema.Fields() {
		switch field.Name {
		case "uuid":
			builder.Field(i).(*array.StringBuilder).Append(doc.UUID)
		case "model_type":
			builder.Field(i).(*array.StringBuilder).Append(doc.ModelType)
		case "framework":
			builder.Field(i).(*array.StringBuilder).Append(doc.Framework)
		case "title":
			builder.Field(i).(*array.StringBuilder).Append(doc.Title)
		case "content":
			builder.Field(i).(*array.StringBuilder).Append(doc.Content)
		case "embedding":
			if len(doc.Embedding) > 0 {
				flb := builder.Field(i).(*array.FixedSizeListBuilder)
				flb.Append(true)
				f32b := flb.ValueBuilder().(*array.Float32Builder)
				for _, v := range doc.Embedding {
					f32b.Append(v)
				}
			} else {
				builder.Field(i).AppendEmptyValue()
			}
		case "created_at":
			builder.Field(i).(*array.Int64Builder).Append(time.Now().UnixMicro())
		default:
			builder.Field(i).AppendEmptyValue()
		}
	}

	return builder.NewRecord(), nil
}

// RemoveLanceDBDir deletes a local LanceDB directory; useful for test cleanup.
func RemoveLanceDBDir(path string) error {
	return os.RemoveAll(path)
}
