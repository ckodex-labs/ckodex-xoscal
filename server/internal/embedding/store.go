package embedding

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"github.com/mchorfa/xoscal/server/internal/config"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	_ "modernc.org/sqlite"
)

// VectorStore persists embeddings and searchable text for KG entities.
type VectorStore interface {
	Index(ctx context.Context, doc Document) error
	Search(ctx context.Context, query string, framework string, topK int) ([]SearchResult, error)
	VectorSearch(ctx context.Context, queryVec []float32, framework string, topK int) ([]SearchResult, error)
	Close() error
}

// SQLiteVectorStore implements VectorStore with SQLite FTS5.
type SQLiteVectorStore struct {
	db *sql.DB
}

// NewVectorStore selects the vector backend based on configuration.
// "lancedb" uses LanceDB full-text search; anything else falls back to SQLite FTS5.
func NewVectorStore(cfg config.Vector, pool dbutil.PoolConfig) (VectorStore, error) {
	if cfg.Backend == "lancedb" {
		return NewLanceDBVectorStore(cfg)
	}
	// Default: SQLite with DSN from main Store config if Vector.URI is empty.
	dsn := cfg.URI
	if dsn == "" {
		dsn = ":memory:"
	}
	return NewSQLiteVectorStore(dsn, pool)
}

func NewSQLiteVectorStore(dsn string, pool dbutil.PoolConfig) (VectorStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := dbutil.Configure(db, dsn, pool); err != nil {
		return nil, closeDatabase(err, db)
	}
	vs := &SQLiteVectorStore{db: db}
	if err := vs.migrate(); err != nil {
		return nil, closeDatabase(fmt.Errorf("migrate: %w", err), db)
	}
	return vs, nil
}

func closeDatabase(primary error, db *sql.DB) error {
	if closeErr := db.Close(); closeErr != nil {
		return fmt.Errorf("%w (close database: %v)", primary, closeErr)
	}
	return primary
}

func (s *SQLiteVectorStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS vector_index (
	 entity_urn TEXT PRIMARY KEY,
	 entity_type TEXT NOT NULL,
	 framework TEXT NOT NULL,
	 content TEXT NOT NULL,
	 embedding BLOB
);
CREATE VIRTUAL TABLE IF NOT EXISTS vector_fts USING fts5(content, content='vector_index', content_rowid='rowid');
CREATE TRIGGER IF NOT EXISTS vector_index_ai AFTER INSERT ON vector_index BEGIN
	INSERT INTO vector_fts(rowid, content) VALUES (new.rowid, new.content);
END;
CREATE TRIGGER IF NOT EXISTS vector_index_ad AFTER DELETE ON vector_index BEGIN
	INSERT INTO vector_fts(vector_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
END;
CREATE TRIGGER IF NOT EXISTS vector_index_au AFTER UPDATE ON vector_index BEGIN
	INSERT INTO vector_fts(vector_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
	INSERT INTO vector_fts(rowid, content) VALUES (new.rowid, new.content);
END;
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	// Ensure embedding column exists for stores migrated prior to dense vector support.
	_, _ = s.db.Exec(`ALTER TABLE vector_index ADD COLUMN embedding BLOB`)
	return nil
}

func (s *SQLiteVectorStore) Close() error { return s.db.Close() }

func (s *SQLiteVectorStore) Index(ctx context.Context, doc Document) error {
	var embBlob []byte
	if len(doc.Embedding) > 0 {
		embBlob = encodeVector(doc.Embedding)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vector_index(entity_urn, entity_type, framework, content, embedding) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(entity_urn) DO UPDATE SET 
		 	content=excluded.content,
		 	embedding=COALESCE(excluded.embedding, vector_index.embedding)`,
		doc.UUID, doc.ModelType, doc.Framework, doc.Content, embBlob,
	)
	if err != nil {
		return fmt.Errorf("index document: %w", err)
	}
	return nil
}

func (s *SQLiteVectorStore) Search(ctx context.Context, query string, framework string, topK int) ([]SearchResult, error) {
	sqlQuery := `SELECT vi.entity_urn, vi.entity_type, bm25(vector_fts) FROM vector_index vi
		 JOIN vector_fts vfts ON vi.rowid = vfts.rowid
		 WHERE vector_fts MATCH ?`
	args := []interface{}{query}
	if framework != "" {
		sqlQuery += ` AND vi.framework = ?`
		args = append(args, framework)
	}
	sqlQuery += ` ORDER BY rank LIMIT ?`
	args = append(args, topK)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var out []SearchResult
	for rows.Next() {
		var sr SearchResult
		var score float64
		if err := rows.Scan(&sr.UUID, &sr.ModelType, &score); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		// bm25 returns negative values (lower is better); normalize to positive 0-1-ish.
		if score < 0 {
			score = -score
		}
		sr.Score = score
		out = append(out, sr)
	}
	return out, rows.Err()
}

// VectorSearch performs dense vector cosine similarity search over stored embeddings.
func (s *SQLiteVectorStore) VectorSearch(ctx context.Context, queryVec []float32, framework string, topK int) ([]SearchResult, error) {
	if len(queryVec) == 0 || topK <= 0 {
		return nil, nil
	}

	sqlQuery := `SELECT entity_urn, entity_type, embedding FROM vector_index WHERE embedding IS NOT NULL`
	var args []interface{}
	if framework != "" {
		sqlQuery += ` AND framework = ?`
		args = append(args, framework)
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	type scored struct {
		result SearchResult
		score  float64
	}
	var scoredList []scored

	for rows.Next() {
		var urn, modelType string
		var rawBlob []byte
		if err := rows.Scan(&urn, &modelType, &rawBlob); err != nil {
			return nil, fmt.Errorf("scan vector row: %w", err)
		}
		vec := decodeVector(rawBlob)
		if len(vec) == 0 {
			continue
		}
		sim := CosineSimilarity(queryVec, vec)
		if sim > 0 {
			scoredList = append(scoredList, scored{
				result: SearchResult{
					UUID:      urn,
					ModelType: modelType,
					Score:     sim,
				},
				score: sim,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	if len(scoredList) > topK {
		scoredList = scoredList[:topK]
	}

	out := make([]SearchResult, len(scoredList))
	for i, item := range scoredList {
		out[i] = item.result
	}
	return out, nil
}

func encodeVector(vec []float32) []byte {
	if len(vec) == 0 {
		return nil
	}
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeVector(buf []byte) []float32 {
	if len(buf) < 4 || len(buf)%4 != 0 {
		return nil
	}
	vec := make([]float32, len(buf)/4)
	for i := range vec {
		bits := binary.LittleEndian.Uint32(buf[i*4:])
		vec[i] = math.Float32frombits(bits)
	}
	return vec
}
