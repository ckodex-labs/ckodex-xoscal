package graph

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists graph nodes and edges for the transparency graph.
type Store interface {
	CreateNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	ListNodes(ctx context.Context, kind, labelFilter string, createdAfter time.Time, limit int) ([]*Node, error)

	CreateEdge(ctx context.Context, edge *Edge) error
	GetEdge(ctx context.Context, id string) (*Edge, error)
	ListEdges(ctx context.Context, fromNode, toNode, relation, trustState string, validAfter time.Time, limit int) ([]*Edge, error)
	DeleteEdge(ctx context.Context, id string) error

	ListEdgesFrom(ctx context.Context, nodeID string, relations []string, minTrustState string, limit int) ([]*Edge, error)
	ListEdgesTo(ctx context.Context, nodeID string, relations []string, minTrustState string, limit int) ([]*Edge, error)

	Close() error
}

// Node is a graph node.
type Node struct {
	ID         string
	Kind       string
	URN        string
	LabelsJSON string
	CreatedAt  time.Time
}

// Edge is a graph edge projected from a claim.
type Edge struct {
	ID             string
	FromNode       string
	ToNode         string
	Relation       string
	Qualifier      string
	ClaimID        string
	EvidenceDigest string
	ProofStateJSON string
	ValidFrom      time.Time
	ValidTo        *time.Time
	TrustState     string
	Weight         float64
	ExtensionsJSON string
}

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens and migrates the graph tables.
func NewSQLiteStore(dsn string) (Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS kg_graph_nodes (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	urn TEXT NOT NULL,
	labels_json TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_nodes_kind ON kg_graph_nodes(kind);
CREATE INDEX IF NOT EXISTS idx_nodes_urn ON kg_graph_nodes(urn);

CREATE TABLE IF NOT EXISTS kg_graph_edges (
	id TEXT PRIMARY KEY,
	from_node TEXT NOT NULL REFERENCES kg_graph_nodes(id) ON DELETE CASCADE,
	to_node TEXT NOT NULL REFERENCES kg_graph_nodes(id) ON DELETE CASCADE,
	relation TEXT NOT NULL,
	qualifier TEXT,
	claim_id TEXT NOT NULL,
	evidence_digest TEXT NOT NULL,
	proof_state_json TEXT,
	valid_from DATETIME NOT NULL,
	valid_to DATETIME,
	trust_state TEXT NOT NULL DEFAULT 'candidate',
	weight REAL DEFAULT 1.0,
	extensions_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_edges_from ON kg_graph_edges(from_node);
CREATE INDEX IF NOT EXISTS idx_edges_to ON kg_graph_edges(to_node);
CREATE INDEX IF NOT EXISTS idx_edges_relation ON kg_graph_edges(relation);
CREATE INDEX IF NOT EXISTS idx_edges_trust ON kg_graph_edges(trust_state);
CREATE INDEX IF NOT EXISTS idx_edges_valid ON kg_graph_edges(valid_from, valid_to);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) CreateNode(ctx context.Context, node *Node) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_graph_nodes(id, kind, urn, labels_json, created_at) VALUES (?, ?, ?, ?, ?)`,
		node.ID, node.Kind, node.URN, node.LabelsJSON, node.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert node: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetNode(ctx context.Context, id string) (*Node, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, kind, urn, labels_json, created_at FROM kg_graph_nodes WHERE id = ?`, id)
	var n Node
	var labels sql.NullString
	if err := row.Scan(&n.ID, &n.Kind, &n.URN, &labels, &n.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan node: %w", err)
	}
	if labels.Valid {
		n.LabelsJSON = labels.String
	}
	return &n, nil
}

func (s *SQLiteStore) ListNodes(ctx context.Context, kind, labelFilter string, createdAfter time.Time, limit int) ([]*Node, error) {
	query := `SELECT id, kind, urn, labels_json, created_at FROM kg_graph_nodes WHERE 1=1`
	args := []interface{}{}
	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if !createdAfter.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, createdAfter)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()
	var out []*Node
	for rows.Next() {
		var n Node
		var labels sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &n.URN, &labels, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		if labels.Valid {
			n.LabelsJSON = labels.String
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateEdge(ctx context.Context, edge *Edge) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_graph_edges(id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		edge.ID, edge.FromNode, edge.ToNode, edge.Relation, edge.Qualifier,
		edge.ClaimID, edge.EvidenceDigest, edge.ProofStateJSON, edge.ValidFrom, edge.ValidTo,
		edge.TrustState, edge.Weight, edge.ExtensionsJSON,
	)
	if err != nil {
		return fmt.Errorf("insert edge: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetEdge(ctx context.Context, id string) (*Edge, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE id = ?`, id)
	return scanEdge(row)
}

func (s *SQLiteStore) ListEdges(ctx context.Context, fromNode, toNode, relation, trustState string, validAfter time.Time, limit int) ([]*Edge, error) {
	query := `SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE 1=1`
	args := []interface{}{}
	if fromNode != "" {
		query += ` AND from_node = ?`
		args = append(args, fromNode)
	}
	if toNode != "" {
		query += ` AND to_node = ?`
		args = append(args, toNode)
	}
	if relation != "" {
		query += ` AND relation = ?`
		args = append(args, relation)
	}
	if trustState != "" {
		query += ` AND trust_state = ?`
		args = append(args, trustState)
	}
	if !validAfter.IsZero() {
		query += ` AND valid_from >= ?`
		args = append(args, validAfter)
	}
	query += ` ORDER BY valid_from DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}
	defer rows.Close()
	var out []*Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) DeleteEdge(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM kg_graph_edges WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) ListEdgesFrom(ctx context.Context, nodeID string, relations []string, minTrustState string, limit int) ([]*Edge, error) {
	query := `SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE from_node = ?`
	args := []interface{}{nodeID}
	if len(relations) > 0 {
		query += ` AND relation IN (` + placeholders(len(relations)) + `)`
		for _, r := range relations {
			args = append(args, r)
		}
	}
	query += ` ORDER BY weight ASC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list edges from: %w", err)
	}
	defer rows.Close()
	var out []*Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListEdgesTo(ctx context.Context, nodeID string, relations []string, minTrustState string, limit int) ([]*Edge, error) {
	query := `SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE to_node = ?`
	args := []interface{}{nodeID}
	if len(relations) > 0 {
		query += ` AND relation IN (` + placeholders(len(relations)) + `)`
		for _, r := range relations {
			args = append(args, r)
		}
	}
	query += ` ORDER BY weight ASC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list edges to: %w", err)
	}
	defer rows.Close()
	var out []*Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanEdge(row scanner) (*Edge, error) {
	var e Edge
	var validTo sql.NullTime
	var qualifier, proofState, extensions sql.NullString
	err := row.Scan(&e.ID, &e.FromNode, &e.ToNode, &e.Relation, &qualifier, &e.ClaimID,
		&e.EvidenceDigest, &proofState, &e.ValidFrom, &validTo, &e.TrustState, &e.Weight, &extensions)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan edge: %w", err)
	}
	if validTo.Valid {
		e.ValidTo = &validTo.Time
	}
	if qualifier.Valid {
		e.Qualifier = qualifier.String
	}
	if proofState.Valid {
		e.ProofStateJSON = proofState.String
	}
	if extensions.Valid {
		e.ExtensionsJSON = extensions.String
	}
	return &e, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	out := "?"
	for i := 1; i < n; i++ {
		out += ",?"
	}
	return out
}
