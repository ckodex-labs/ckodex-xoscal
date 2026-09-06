package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

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

func (s *SQLiteStore) ListNodes(ctx context.Context, kind, labelFilter string, createdAfter time.Time, limit int, pageToken string) ([]*Node, string, error) {
	if limit <= 0 {
		limit = 50
	}
	offset, err := parsePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	query := `SELECT id, kind, urn, labels_json, created_at FROM kg_graph_nodes WHERE 1=1`
	args := []interface{}{}
	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if labelFilter != "" {
		query += ` AND labels_json = ?`
		args = append(args, labelFilter)
	}
	if !createdAfter.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, createdAfter)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()
	var out []*Node
	hasMore := false
	for rows.Next() {
		var n Node
		var labels sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &n.URN, &labels, &n.CreatedAt); err != nil {
			return nil, "", fmt.Errorf("scan node: %w", err)
		}
		if len(out) == limit {
			hasMore = true
			break
		}
		if labels.Valid {
			n.LabelsJSON = labels.String
		}
		out = append(out, &n)
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
