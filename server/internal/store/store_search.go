package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func (s *SQLiteStore) Search(ctx context.Context, query string, modelTypes []string, pageSize int, pageToken string) ([]SearchResult, string, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	var offset int
	if pageToken != "" {
		if _, err := fmt.Sscanf(pageToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
	}

	allTypes := []string{"catalogs", "profiles", "component_definitions", "ssps", "assessment_plans", "assessment_results", "poams", "mappings"}
	types := allTypes
	if len(modelTypes) > 0 {
		types = make([]string, 0, len(modelTypes))
		for _, mt := range modelTypes {
			t := strings.ToLower(mt)
			switch t {
			case "catalog":
				types = append(types, "catalogs")
			case "profile":
				types = append(types, "profiles")
			case "component_definition":
				types = append(types, "component_definitions")
			case "ssp":
				types = append(types, "ssps")
			case "assessment_plan":
				types = append(types, "assessment_plans")
			case "assessment_results":
				types = append(types, "assessment_results")
			case "poam":
				types = append(types, "poams")
			case "mapping":
				types = append(types, "mappings")
			default:
				types = append(types, t)
			}
		}
	}

	like := "%" + query + "%"
	var allResults []SearchResult
	for _, table := range types {
		rows, err := s.db.QueryContext(ctx,
			fmt.Sprintf("SELECT uuid, title, version FROM %s WHERE title LIKE ? OR version LIKE ? ORDER BY uuid LIMIT ? OFFSET ?", table),
			like, like, pageSize+1, offset,
		)
		if err != nil {
			return nil, "", fmt.Errorf("search %s: %w", table, err)
		}
		for rows.Next() {
			var r SearchResult
			var version string
			if err := rows.Scan(&r.UUID, &r.Title, &version); err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					return nil, "", fmt.Errorf("scan %s: %w (close: %v)", table, err, closeErr)
				}
				return nil, "", fmt.Errorf("scan %s: %w", table, err)
			}
			r.ModelType = strings.TrimSuffix(table, "s")
			if r.ModelType == "ssps" {
				r.ModelType = "ssp"
			} else if r.ModelType == "poams" {
				r.ModelType = "poam"
			}
			r.Score = 1.0 // placeholder until semantic search is implemented
			allResults = append(allResults, r)
		}
		if err := rows.Close(); err != nil {
			return nil, "", fmt.Errorf("close search rows for %s: %w", table, err)
		}
		if err := rows.Err(); err != nil {
			return nil, "", fmt.Errorf("search rows %s: %w", table, err)
		}
	}

	nextToken := ""
	if len(allResults) > pageSize {
		nextToken = fmt.Sprintf("%d", offset+pageSize)
		allResults = allResults[:pageSize]
	}
	return allResults, nextToken, nil
}

func closeDatabase(primary error, db *sql.DB) error {
	if closeErr := db.Close(); closeErr != nil {
		return fmt.Errorf("%w (close database: %v)", primary, closeErr)
	}
	return primary
}
