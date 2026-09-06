package graph

import (
	"context"
	"fmt"
	"time"
)

// CreateProjection persists both endpoint nodes and their edge atomically.
// Projection is a single graph fact; a failed edge write must not leave
// nodes that imply a partially accepted claim.
func (s *SQLiteStore) CreateProjection(ctx context.Context, from, to *Node, edge *Edge) (*ProjectionEvent, error) {
	if from == nil || to == nil || edge == nil {
		return nil, fmt.Errorf("projection nodes and edge are required")
	}
	if edge.FromNode != from.ID || edge.ToNode != to.ID {
		return nil, fmt.Errorf("projection edge endpoints do not match nodes")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin projection: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	for _, node := range []*Node{from, to} {
		if node.CreatedAt.IsZero() {
			node.CreatedAt = time.Now().UTC()
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kg_graph_nodes(id, kind, urn, labels_json, created_at) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(id) DO NOTHING`,
			node.ID, node.Kind, node.URN, node.LabelsJSON, node.CreatedAt); err != nil {
			return nil, rollback(fmt.Errorf("insert projection node %q: %w", node.ID, err))
		}
		var kind, urn string
		if err := tx.QueryRowContext(ctx, `SELECT kind, urn FROM kg_graph_nodes WHERE id = ?`, node.ID).Scan(&kind, &urn); err != nil {
			return nil, rollback(fmt.Errorf("read projection node %q: %w", node.ID, err))
		}
		if kind != node.Kind || urn != node.URN {
			return nil, rollback(fmt.Errorf("node %q already exists with different identity", node.ID))
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kg_graph_edges(id, from_node, to_node, relation, qualifier, claim_id, evidence_digest,
		 proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		edge.ID, edge.FromNode, edge.ToNode, edge.Relation, edge.Qualifier, edge.ClaimID,
		edge.EvidenceDigest, edge.ProofStateJSON, edge.ValidFrom, edge.ValidTo, edge.TrustState,
		edge.Weight, edge.ExtensionsJSON); err != nil {
		return nil, rollback(fmt.Errorf("insert projection edge %q: %w", edge.ID, err))
	}
	event, err := appendProjectionEvent(ctx, tx, edge)
	if err != nil {
		return nil, rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit projection: %w", err)
	}
	return event, nil
}
