package kg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	_ "modernc.org/sqlite"
)

// Store defines CRUD and snapshot operations for the knowledge graph.
type Store interface {
	CreateEntity(ctx context.Context, entity *Entity) error
	GetEntity(ctx context.Context, urn string) (*Entity, error)
	UpdateEntity(ctx context.Context, entity *Entity) error
	ListEntities(ctx context.Context, entityType string, status EntityStatus) ([]*Entity, error)
	Snapshot(ctx context.Context, t time.Time) ([]*Entity, error)
	CreateSnapshot(ctx context.Context, name string) (*Snapshot, error)
	GetSnapshot(ctx context.Context, name string) ([]*Entity, error)
	ListSnapshots(ctx context.Context) ([]*Snapshot, error)
	CreateRelease(ctx context.Context, name, snapshotName string) (*Release, error)
	GetRelease(ctx context.Context, name string) (*Release, error)
	ListReleases(ctx context.Context) ([]*Release, error)
	Close() error
}

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string, pool dbutil.PoolConfig) (Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := dbutil.Configure(db, dsn, pool); err != nil {
		return nil, closeDatabase(err, db)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, closeDatabase(fmt.Errorf("enable foreign keys: %w", err), db)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, closeDatabase(fmt.Errorf("migrate: %w", err), db)
	}
	return s, nil
}

func closeDatabase(primary error, db *sql.DB) error {
	if closeErr := db.Close(); closeErr != nil {
		return fmt.Errorf("%w (close database: %v)", primary, closeErr)
	}
	return primary
}

func (s *SQLiteStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS kg_entities (
	 urn TEXT NOT NULL,
	 entity_type TEXT NOT NULL,
	 version INTEGER NOT NULL DEFAULT 1,
	 status TEXT NOT NULL DEFAULT 'active',
	 valid_from DATETIME NOT NULL DEFAULT (datetime('now')),
	 valid_to DATETIME,
	 payload JSON NOT NULL,
	 PRIMARY KEY (urn, version)
);
CREATE INDEX IF NOT EXISTS idx_kg_type ON kg_entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_kg_status ON kg_entities(status);
CREATE INDEX IF NOT EXISTS idx_kg_valid ON kg_entities(valid_from, valid_to);

CREATE TABLE IF NOT EXISTS kg_snapshots (
	 name TEXT PRIMARY KEY,
	 created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS kg_snapshot_entities (
	 snapshot_name TEXT NOT NULL REFERENCES kg_snapshots(name) ON DELETE CASCADE,
	 entity_urn TEXT NOT NULL,
	 version INTEGER NOT NULL,
	 PRIMARY KEY (snapshot_name, entity_urn)
);
CREATE TABLE IF NOT EXISTS kg_releases (
	 name TEXT PRIMARY KEY,
	 snapshot_name TEXT NOT NULL REFERENCES kg_snapshots(name),
	 created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) CreateEntity(ctx context.Context, entity *Entity) error {
	if entity.Version == 0 {
		entity.Version = 1
	}
	if entity.Status == "" {
		entity.Status = EntityStatusActive
	}
	if entity.ValidFrom.IsZero() {
		entity.ValidFrom = time.Now().UTC()
	}
	payload, err := json.Marshal(entity.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO kg_entities(urn, entity_type, version, status, valid_from, valid_to, payload) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entity.URN, entity.Type, entity.Version, string(entity.Status), entity.ValidFrom, entity.ValidTo, payload,
	)
	if err != nil {
		return fmt.Errorf("insert entity: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetEntity(ctx context.Context, urn string) (*Entity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT urn, entity_type, version, status, valid_from, valid_to, payload FROM kg_entities WHERE urn = ? AND status = ? ORDER BY version DESC LIMIT 1`, urn, string(EntityStatusActive))
	var e Entity
	var statusStr string
	var validTo sql.NullTime
	if err := row.Scan(&e.URN, &e.Type, &e.Version, &statusStr, &e.ValidFrom, &validTo, &e.Payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan entity: %w", err)
	}
	e.Status = EntityStatus(statusStr)
	if validTo.Valid {
		e.ValidTo = &validTo.Time
	}
	return &e, nil
}

func (s *SQLiteStore) UpdateEntity(ctx context.Context, entity *Entity) error {
	now := time.Now().UTC()
	// Soft-delete old version
	_, err := s.db.ExecContext(ctx,
		`UPDATE kg_entities SET status = ?, valid_to = ? WHERE urn = ? AND status = ?`,
		string(EntityStatusSuperseded), now, entity.URN, string(EntityStatusActive),
	)
	if err != nil {
		return fmt.Errorf("expire old version: %w", err)
	}
	// Compute next version from DB to avoid collisions.
	var maxVersion int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM kg_entities WHERE urn = ?`, entity.URN,
	).Scan(&maxVersion)
	entity.Version = maxVersion + 1
	entity.Status = EntityStatusActive
	entity.ValidFrom = now
	entity.ValidTo = nil
	return s.CreateEntity(ctx, entity)
}

func (s *SQLiteStore) ListEntities(ctx context.Context, entityType string, status EntityStatus) ([]*Entity, error) {
	query := `SELECT urn, entity_type, version, status, valid_from, valid_to, payload FROM kg_entities WHERE 1=1`
	args := []interface{}{}
	if entityType != "" {
		query += ` AND entity_type = ?`
		args = append(args, entityType)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY urn, version`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}
	defer rows.Close()
	return scanEntities(rows)
}

func (s *SQLiteStore) Snapshot(ctx context.Context, t time.Time) ([]*Entity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT urn, entity_type, version, status, valid_from, valid_to, payload FROM kg_entities
		 WHERE status = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)
		 ORDER BY urn, version`,
		string(EntityStatusActive), t, t,
	)
	if err != nil {
		return nil, fmt.Errorf("snapshot query: %w", err)
	}
	defer rows.Close()
	return scanEntities(rows)
}

func (s *SQLiteStore) CreateSnapshot(ctx context.Context, name string) (*Snapshot, error) {
	now := time.Now().UTC()
	entities, err := s.Snapshot(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO kg_snapshots(name, created_at) VALUES (?, ?)`, name, now); err != nil {
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}
	for _, e := range entities {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO kg_snapshot_entities(snapshot_name, entity_urn, version) VALUES (?, ?, ?)`,
			name, e.URN, e.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("insert snapshot entity: %w", err)
		}
	}
	return &Snapshot{Name: name, CreatedAt: now, EntityCount: len(entities)}, nil
}

func (s *SQLiteStore) GetSnapshot(ctx context.Context, name string) ([]*Entity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT e.urn, e.entity_type, e.version, e.status, e.valid_from, e.valid_to, e.payload
		 FROM kg_snapshot_entities se
		 JOIN kg_entities e ON se.entity_urn = e.urn AND se.version = e.version
		 WHERE se.snapshot_name = ?`, name)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	defer rows.Close()
	return scanEntities(rows)
}

func (s *SQLiteStore) ListSnapshots(ctx context.Context) ([]*Snapshot, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, created_at FROM kg_snapshots ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	var out []*Snapshot
	for rows.Next() {
		var sn Snapshot
		if err := rows.Scan(&sn.Name, &sn.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		// Count entities
		var count int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM kg_snapshot_entities WHERE snapshot_name = ?`, sn.Name).Scan(&count); err == nil {
			sn.EntityCount = count
		}
		out = append(out, &sn)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateRelease(ctx context.Context, name, snapshotName string) (*Release, error) {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_releases(name, snapshot_name, created_at) VALUES (?, ?, ?)`,
		name, snapshotName, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert release: %w", err)
	}
	return &Release{Name: name, Snapshot: snapshotName, CreatedAt: now}, nil
}

func (s *SQLiteStore) GetRelease(ctx context.Context, name string) (*Release, error) {
	row := s.db.QueryRowContext(ctx, `SELECT name, snapshot_name, created_at FROM kg_releases WHERE name = ?`, name)
	var r Release
	if err := row.Scan(&r.Name, &r.Snapshot, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan release: %w", err)
	}
	return &r, nil
}

func (s *SQLiteStore) ListReleases(ctx context.Context) ([]*Release, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, snapshot_name, created_at FROM kg_releases ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	defer rows.Close()
	var out []*Release
	for rows.Next() {
		var r Release
		if err := rows.Scan(&r.Name, &r.Snapshot, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan release: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func scanEntities(rows *sql.Rows) ([]*Entity, error) {
	var out []*Entity
	for rows.Next() {
		var e Entity
		var statusStr string
		var validTo sql.NullTime
		if err := rows.Scan(&e.URN, &e.Type, &e.Version, &statusStr, &e.ValidFrom, &validTo, &e.Payload); err != nil {
			return nil, fmt.Errorf("scan entity: %w", err)
		}
		e.Status = EntityStatus(statusStr)
		if validTo.Valid {
			e.ValidTo = &validTo.Time
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
