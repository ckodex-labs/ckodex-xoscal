package graph

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ProjectionEvent is the immutable audit record for a successful graph edge
// projection. It is committed in the same transaction as the edge itself.
type ProjectionEvent struct {
	Sequence       int64
	EventID        string
	EdgeID         string
	ClaimID        string
	FromNode       string
	ToNode         string
	Relation       string
	EvidenceDigest string
	TrustState     string
	PreviousHash   string
	EventHash      string
	ProjectedAt    time.Time
}

func appendProjectionEvent(ctx context.Context, tx *sql.Tx, edge *Edge) (*ProjectionEvent, error) {
	var previousHash string
	if err := tx.QueryRowContext(ctx,
		`SELECT event_hash FROM graph_projection_events ORDER BY sequence DESC LIMIT 1`).Scan(&previousHash); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("read previous projection hash: %w", err)
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence), 0) + 1 FROM graph_projection_events`).Scan(&sequence); err != nil {
		return nil, fmt.Errorf("allocate projection sequence: %w", err)
	}
	event := &ProjectionEvent{
		Sequence:       sequence,
		EventID:        fmt.Sprintf("graph-projection-%s-%d", edge.ID, sequence),
		EdgeID:         edge.ID,
		ClaimID:        edge.ClaimID,
		FromNode:       edge.FromNode,
		ToNode:         edge.ToNode,
		Relation:       edge.Relation,
		EvidenceDigest: edge.EvidenceDigest,
		TrustState:     edge.TrustState,
		PreviousHash:   previousHash,
		ProjectedAt:    time.Now().UTC(),
	}
	event.EventHash = hashProjectionEvent(event)
	_, err := tx.ExecContext(ctx,
		`INSERT INTO graph_projection_events(sequence, event_id, edge_id, claim_id, from_node,
		 to_node, relation, evidence_digest, trust_state, previous_hash, event_hash, projected_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Sequence, event.EventID, event.EdgeID, event.ClaimID, event.FromNode,
		event.ToNode, event.Relation, event.EvidenceDigest, event.TrustState,
		event.PreviousHash, event.EventHash, event.ProjectedAt)
	if err != nil {
		return nil, fmt.Errorf("append projection event: %w", err)
	}
	return event, nil
}

func (s *SQLiteStore) ListProjectionEvents(ctx context.Context, claimID, edgeID string) ([]*ProjectionEvent, error) {
	query := `SELECT sequence, event_id, edge_id, claim_id, from_node, to_node, relation,
		evidence_digest, trust_state, previous_hash, event_hash, projected_at
		FROM graph_projection_events WHERE 1=1`
	args := []interface{}{}
	if claimID != "" {
		query += ` AND claim_id = ?`
		args = append(args, claimID)
	}
	if edgeID != "" {
		query += ` AND edge_id = ?`
		args = append(args, edgeID)
	}
	query += ` ORDER BY sequence ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projection events: %w", err)
	}
	defer rows.Close()
	var events []*ProjectionEvent
	for rows.Next() {
		event, err := scanProjectionEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projection events: %w", err)
	}
	return events, nil
}

// VerifyProjectionChain validates every graph projection event in sequence.
func (s *SQLiteStore) VerifyProjectionChain(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sequence, event_id, edge_id, claim_id, from_node, to_node, relation,
		 evidence_digest, trust_state, previous_hash, event_hash, projected_at
		 FROM graph_projection_events ORDER BY sequence ASC`)
	if err != nil {
		return fmt.Errorf("read projection chain: %w", err)
	}
	defer rows.Close()
	previousHash := ""
	var expectedSequence int64 = 1
	for rows.Next() {
		event, err := scanProjectionEvent(rows)
		if err != nil {
			return err
		}
		if event.Sequence != expectedSequence {
			return fmt.Errorf("projection chain sequence %d does not match expected %d", event.Sequence, expectedSequence)
		}
		if event.PreviousHash != previousHash {
			return fmt.Errorf("projection chain link %d does not match", event.Sequence)
		}
		if event.EventHash != hashProjectionEvent(event) {
			return fmt.Errorf("projection event %q hash does not match", event.EventID)
		}
		previousHash = event.EventHash
		expectedSequence++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read projection chain: %w", err)
	}
	return nil
}

func scanProjectionEvent(row scanner) (*ProjectionEvent, error) {
	var event ProjectionEvent
	if err := row.Scan(&event.Sequence, &event.EventID, &event.EdgeID, &event.ClaimID,
		&event.FromNode, &event.ToNode, &event.Relation, &event.EvidenceDigest,
		&event.TrustState, &event.PreviousHash, &event.EventHash, &event.ProjectedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan projection event: %w", err)
	}
	return &event, nil
}

func hashProjectionEvent(event *ProjectionEvent) string {
	payload := struct {
		Version        string
		Sequence       int64
		EventID        string
		EdgeID         string
		ClaimID        string
		FromNode       string
		ToNode         string
		Relation       string
		EvidenceDigest string
		TrustState     string
		PreviousHash   string
		ProjectedAt    string
	}{
		Version: "graph-projection-v1", Sequence: event.Sequence, EventID: event.EventID,
		EdgeID: event.EdgeID, ClaimID: event.ClaimID, FromNode: event.FromNode,
		ToNode: event.ToNode, Relation: event.Relation, EvidenceDigest: event.EvidenceDigest,
		TrustState: event.TrustState, PreviousHash: event.PreviousHash,
		ProjectedAt: event.ProjectedAt.UTC().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
