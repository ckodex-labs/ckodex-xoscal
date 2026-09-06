package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists graph nodes and edges for the transparency graph.
type Store interface {
	CreateNode(ctx context.Context, node *Node) error
	CreateProjection(ctx context.Context, from, to *Node, edge *Edge) (*ProjectionEvent, error)
	ListProjectionEvents(ctx context.Context, claimID, edgeID string) ([]*ProjectionEvent, error)
	VerifyProjectionChain(ctx context.Context) error
	GetNode(ctx context.Context, id string) (*Node, error)
	ListNodes(ctx context.Context, kind, labelFilter string, createdAfter time.Time, limit int, pageToken string) ([]*Node, string, error)

	CreateEdge(ctx context.Context, edge *Edge) error
	GetEdge(ctx context.Context, id string) (*Edge, error)
	ListEdges(ctx context.Context, fromNode, toNode, relation, trustState string, validAfter time.Time, limit int, pageToken string) ([]*Edge, string, error)
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

CREATE TABLE IF NOT EXISTS graph_projection_events (
	sequence INTEGER PRIMARY KEY,
	event_id TEXT NOT NULL UNIQUE,
	edge_id TEXT NOT NULL,
	claim_id TEXT NOT NULL,
	from_node TEXT NOT NULL,
	to_node TEXT NOT NULL,
	relation TEXT NOT NULL,
	evidence_digest TEXT NOT NULL,
	trust_state TEXT NOT NULL,
	previous_hash TEXT NOT NULL,
	event_hash TEXT NOT NULL,
	projected_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_graph_projection_events_claim ON graph_projection_events(claim_id, sequence);
CREATE INDEX IF NOT EXISTS idx_graph_projection_events_edge ON graph_projection_events(edge_id, sequence);
CREATE TRIGGER IF NOT EXISTS graph_projection_events_no_update
BEFORE UPDATE ON graph_projection_events
BEGIN
	SELECT RAISE(ABORT, 'graph_projection_events are append-only');
END;
CREATE TRIGGER IF NOT EXISTS graph_projection_events_no_delete
BEFORE DELETE ON graph_projection_events
BEGIN
	SELECT RAISE(ABORT, 'graph_projection_events are append-only');
END;
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

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

func (s *SQLiteStore) ListEdges(ctx context.Context, fromNode, toNode, relation, trustState string, validAfter time.Time, limit int, pageToken string) ([]*Edge, string, error) {
	if limit <= 0 {
		limit = 50
	}
	offset, err := parsePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
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
	query += ` ORDER BY valid_from DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list edges: %w", err)
	}
	defer rows.Close()
	var out []*Edge
	hasMore := false
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, "", err
		}
		if len(out) == limit {
			hasMore = true
			break
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(offset + limit)
	}
	return out, nextToken, nil
}

func (s *SQLiteStore) DeleteEdge(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM kg_graph_edges WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) ListEdgesFrom(ctx context.Context, nodeID string, relations []string, minTrustState string, limit int) ([]*Edge, error) {
	trustRank, err := minTrustRank(minTrustState)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE from_node = ?`
	args := []interface{}{nodeID}
	if minTrustState != "" {
		query += ` AND CASE trust_state WHEN 'rejected' THEN -1 WHEN 'candidate' THEN 0 WHEN 'incomplete' THEN 1 WHEN 'verified' THEN 2 ELSE -1 END >= ?`
		args = append(args, trustRank)
	}
	if len(relations) > 0 {
		// #nosec G202 -- only the numeric placeholder count is concatenated; relation values remain bound parameters.
		query += ` AND relation IN (` + placeholders(len(relations)) + `)`
		for _, r := range relations {
			args = append(args, r)
		}
	}
	query += ` ORDER BY weight ASC, id ASC`
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
	trustRank, err := minTrustRank(minTrustState)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json
		FROM kg_graph_edges WHERE to_node = ?`
	args := []interface{}{nodeID}
	if minTrustState != "" {
		query += ` AND CASE trust_state WHEN 'rejected' THEN -1 WHEN 'candidate' THEN 0 WHEN 'incomplete' THEN 1 WHEN 'verified' THEN 2 ELSE -1 END >= ?`
		args = append(args, trustRank)
	}
	if len(relations) > 0 {
		// #nosec G202 -- only the numeric placeholder count is concatenated; relation values remain bound parameters.
		query += ` AND relation IN (` + placeholders(len(relations)) + `)`
		for _, r := range relations {
			args = append(args, r)
		}
	}
	query += ` ORDER BY weight ASC, id ASC`
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

func minTrustRank(state string) (int, error) {
	switch strings.ToLower(state) {
	case "":
		return 0, nil
	case "candidate":
		return 0, nil
	case "incomplete":
		return 1, nil
	case "verified":
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid minimum trust state %q", state)
	}
}

func parsePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}
