package embedding

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mchorfa/xoscal/server/internal/config"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	_ "modernc.org/sqlite"
)

// VectorStore persists embeddings and searchable text for KG entities.
type VectorStore interface {
	Index(ctx context.Context, doc Document) error
	Search(ctx context.Context, query string, framework string, topK int) ([]SearchResult, error)
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
		db.Close()
		return nil, err
	}
	vs := &SQLiteVectorStore{db: db}
	if err := vs.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return vs, nil
}

func (s *SQLiteVectorStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS vector_index (
	 entity_urn TEXT PRIMARY KEY,
	 entity_type TEXT NOT NULL,
	 framework TEXT NOT NULL,
	 content TEXT NOT NULL
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
	return nil
}

func (s *SQLiteVectorStore) Close() error { return s.db.Close() }

func (s *SQLiteVectorStore) Index(ctx context.Context, doc Document) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vector_index(entity_urn, entity_type, framework, content) VALUES (?, ?, ?, ?)
		 ON CONFLICT(entity_urn) DO UPDATE SET content=excluded.content`,
		doc.UUID, doc.ModelType, doc.Framework, doc.Content,
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
