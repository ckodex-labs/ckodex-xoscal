package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	assessmentplanv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessmentresultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	componentdefinitionv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
)

func extractTitleVersion(msg proto.Message) (string, string) {
	switch m := msg.(type) {
	case *catalogv1.Catalog:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *profilev1.Profile:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *componentdefinitionv1.ComponentDefinition:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *sspv1.SystemSecurityPlan:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *assessmentplanv1.AssessmentPlan:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *assessmentresultsv1.AssessmentResults:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *poamv1.PlanOfActionAndMilestones:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	case *mappingv1.MappingCollection:
		if m.Metadata != nil {
			return m.Metadata.Title, m.Metadata.Version
		}
	}
	return "", ""
}

func create[T proto.Message](ctx context.Context, db *sql.DB, table string, uuid string, msg T) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	title, version := extractTitleVersion(msg)
	_, err = db.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (uuid, title, version, data, updated_at) VALUES (?, ?, ?, ?, ?)", table),
		uuid, title, version, b, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}
	return nil
}

func get[T proto.Message](ctx context.Context, db *sql.DB, table string, uuid string, msg T) (T, error) {
	var data []byte
	row := db.QueryRowContext(ctx, fmt.Sprintf("SELECT data FROM %s WHERE uuid = ?", table), uuid)
	if err := row.Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return msg, fmt.Errorf("%s not found: %w", table, err)
		}
		return msg, fmt.Errorf("scan %s: %w", table, err)
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return msg, fmt.Errorf("unmarshal %s: %w", table, err)
	}
	return msg, nil
}

func list[T proto.Message](ctx context.Context, db *sql.DB, table string, filter string, pageSize int, pageToken string, factory func() T) ([]T, string, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	var offset int
	if pageToken != "" {
		if _, err := fmt.Sscanf(pageToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
	}

	var rows *sql.Rows
	var err error
	if filter != "" {
		like := "%" + filter + "%"
		rows, err = db.QueryContext(ctx,
			fmt.Sprintf("SELECT data FROM %s WHERE title LIKE ? OR version LIKE ? ORDER BY uuid LIMIT ? OFFSET ?", table),
			like, like, pageSize+1, offset,
		)
	} else {
		rows, err = db.QueryContext(ctx,
			fmt.Sprintf("SELECT data FROM %s ORDER BY uuid LIMIT ? OFFSET ?", table),
			pageSize+1, offset,
		)
	}
	if err != nil {
		return nil, "", fmt.Errorf("query %s: %w", table, err)
	}
	defer rows.Close()

	var results []T
	count := 0
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, "", fmt.Errorf("scan %s: %w", table, err)
		}
		msg := factory()
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, "", fmt.Errorf("unmarshal %s: %w", table, err)
		}
		results = append(results, msg)
		count++
		if count == pageSize {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows %s: %w", table, err)
	}

	nextToken := ""
	if count == pageSize {
		// Peek next row to see if there's more
		if rows.Next() {
			nextToken = fmt.Sprintf("%d", offset+pageSize)
		}
	}
	return results, nextToken, nil
}

func update[T proto.Message](ctx context.Context, db *sql.DB, table string, uuid string, msg T) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	title, version := extractTitleVersion(msg)
	res, err := db.ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET title = ?, version = ?, data = ?, updated_at = ? WHERE uuid = ?", table),
		title, version, b, time.Now().UTC(), uuid,
	)
	if err != nil {
		return fmt.Errorf("update %s: %w", table, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func del(ctx context.Context, db *sql.DB, table string, uuid string) error {
	res, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE uuid = ?", table), uuid)
	if err != nil {
		return fmt.Errorf("delete %s: %w", table, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- Catalog ----
