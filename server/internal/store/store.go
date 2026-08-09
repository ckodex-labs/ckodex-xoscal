package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	ModelType string
	UUID      string
	Title     string
	Score     float64
}

// Store defines the persistence interface for all OSCAL models.
type Store interface {
	CreateCatalog(ctx context.Context, c *catalogv1.Catalog) error
	GetCatalog(ctx context.Context, uuid string) (*catalogv1.Catalog, error)
	ListCatalogs(ctx context.Context, filter string, pageSize int, pageToken string) ([]*catalogv1.Catalog, string, error)
	UpdateCatalog(ctx context.Context, uuid string, c *catalogv1.Catalog) error
	DeleteCatalog(ctx context.Context, uuid string) error

	CreateProfile(ctx context.Context, p *profilev1.Profile) error
	GetProfile(ctx context.Context, uuid string) (*profilev1.Profile, error)
	ListProfiles(ctx context.Context, filter string, pageSize int, pageToken string) ([]*profilev1.Profile, string, error)
	UpdateProfile(ctx context.Context, uuid string, p *profilev1.Profile) error
	DeleteProfile(ctx context.Context, uuid string) error

	CreateComponentDefinition(ctx context.Context, cd *componentdefinitionv1.ComponentDefinition) error
	GetComponentDefinition(ctx context.Context, uuid string) (*componentdefinitionv1.ComponentDefinition, error)
	ListComponentDefinitions(ctx context.Context, filter string, pageSize int, pageToken string) ([]*componentdefinitionv1.ComponentDefinition, string, error)
	UpdateComponentDefinition(ctx context.Context, uuid string, cd *componentdefinitionv1.ComponentDefinition) error
	DeleteComponentDefinition(ctx context.Context, uuid string) error

	CreateSsp(ctx context.Context, s *sspv1.SystemSecurityPlan) error
	GetSsp(ctx context.Context, uuid string) (*sspv1.SystemSecurityPlan, error)
	ListSsps(ctx context.Context, filter string, pageSize int, pageToken string) ([]*sspv1.SystemSecurityPlan, string, error)
	UpdateSsp(ctx context.Context, uuid string, s *sspv1.SystemSecurityPlan) error
	DeleteSsp(ctx context.Context, uuid string) error

	CreateAssessmentPlan(ctx context.Context, ap *assessmentplanv1.AssessmentPlan) error
	GetAssessmentPlan(ctx context.Context, uuid string) (*assessmentplanv1.AssessmentPlan, error)
	ListAssessmentPlans(ctx context.Context, filter string, pageSize int, pageToken string) ([]*assessmentplanv1.AssessmentPlan, string, error)
	UpdateAssessmentPlan(ctx context.Context, uuid string, ap *assessmentplanv1.AssessmentPlan) error
	DeleteAssessmentPlan(ctx context.Context, uuid string) error

	CreateAssessmentResults(ctx context.Context, ar *assessmentresultsv1.AssessmentResults) error
	GetAssessmentResults(ctx context.Context, uuid string) (*assessmentresultsv1.AssessmentResults, error)
	ListAssessmentResults(ctx context.Context, filter string, pageSize int, pageToken string) ([]*assessmentresultsv1.AssessmentResults, string, error)
	UpdateAssessmentResults(ctx context.Context, uuid string, ar *assessmentresultsv1.AssessmentResults) error
	DeleteAssessmentResults(ctx context.Context, uuid string) error

	CreatePoam(ctx context.Context, p *poamv1.PlanOfActionAndMilestones) error
	GetPoam(ctx context.Context, uuid string) (*poamv1.PlanOfActionAndMilestones, error)
	ListPoams(ctx context.Context, filter string, pageSize int, pageToken string) ([]*poamv1.PlanOfActionAndMilestones, string, error)
	UpdatePoam(ctx context.Context, uuid string, p *poamv1.PlanOfActionAndMilestones) error
	DeletePoam(ctx context.Context, uuid string) error

	CreateMapping(ctx context.Context, m *mappingv1.MappingCollection) error
	GetMapping(ctx context.Context, uuid string) (*mappingv1.MappingCollection, error)
	ListMappings(ctx context.Context, filter string, pageSize int, pageToken string) ([]*mappingv1.MappingCollection, string, error)
	UpdateMapping(ctx context.Context, uuid string, m *mappingv1.MappingCollection) error
	DeleteMapping(ctx context.Context, uuid string) error

	Search(ctx context.Context, query string, modelTypes []string, pageSize int, pageToken string) ([]SearchResult, string, error)
	Close() error
}

// SQLiteStore implements Store using modernc.org/sqlite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) an SQLite database, applies connection pool limits, and migrates schema.
func NewSQLiteStore(dsn string, pool dbutil.PoolConfig) (Store, error) {
	if dsn == "" {
		dsn = ":memory:"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := dbutil.Configure(db, dsn, pool); err != nil {
		return nil, closeDatabase(err, db)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, closeDatabase(fmt.Errorf("migrate: %w", err), db)
	}
	return s, nil
}

// Close closes the underlying database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	models := []string{
		"catalogs", "profiles", "component_definitions", "ssps",
		"assessment_plans", "assessment_results", "poams", "mappings",
	}
	for _, m := range models {
		_, err := s.db.Exec(fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				uuid TEXT PRIMARY KEY,
				title TEXT NOT NULL DEFAULT '',
				version TEXT NOT NULL DEFAULT '',
				data BLOB NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_%s_title ON %s(title);
			CREATE INDEX IF NOT EXISTS idx_%s_version ON %s(version);
		`, m, m, m, m, m))
		if err != nil {
			return fmt.Errorf("create table %s: %w", m, err)
		}
	}
	return nil
}

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

func (s *SQLiteStore) CreateCatalog(ctx context.Context, c *catalogv1.Catalog) error {
	return create(ctx, s.db, "catalogs", c.Uuid.Value, c)
}

func (s *SQLiteStore) GetCatalog(ctx context.Context, uuid string) (*catalogv1.Catalog, error) {
	return get(ctx, s.db, "catalogs", uuid, &catalogv1.Catalog{})
}

func (s *SQLiteStore) ListCatalogs(ctx context.Context, filter string, pageSize int, pageToken string) ([]*catalogv1.Catalog, string, error) {
	return list(ctx, s.db, "catalogs", filter, pageSize, pageToken, func() *catalogv1.Catalog { return &catalogv1.Catalog{} })
}

func (s *SQLiteStore) UpdateCatalog(ctx context.Context, uuid string, c *catalogv1.Catalog) error {
	return update(ctx, s.db, "catalogs", uuid, c)
}

func (s *SQLiteStore) DeleteCatalog(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "catalogs", uuid)
}

// ---- Profile ----

func (s *SQLiteStore) CreateProfile(ctx context.Context, p *profilev1.Profile) error {
	return create(ctx, s.db, "profiles", p.Uuid.Value, p)
}

func (s *SQLiteStore) GetProfile(ctx context.Context, uuid string) (*profilev1.Profile, error) {
	return get(ctx, s.db, "profiles", uuid, &profilev1.Profile{})
}

func (s *SQLiteStore) ListProfiles(ctx context.Context, filter string, pageSize int, pageToken string) ([]*profilev1.Profile, string, error) {
	return list(ctx, s.db, "profiles", filter, pageSize, pageToken, func() *profilev1.Profile { return &profilev1.Profile{} })
}

func (s *SQLiteStore) UpdateProfile(ctx context.Context, uuid string, p *profilev1.Profile) error {
	return update(ctx, s.db, "profiles", uuid, p)
}

func (s *SQLiteStore) DeleteProfile(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "profiles", uuid)
}

// ---- Component Definition ----

func (s *SQLiteStore) CreateComponentDefinition(ctx context.Context, cd *componentdefinitionv1.ComponentDefinition) error {
	return create(ctx, s.db, "component_definitions", cd.Uuid.Value, cd)
}

func (s *SQLiteStore) GetComponentDefinition(ctx context.Context, uuid string) (*componentdefinitionv1.ComponentDefinition, error) {
	return get(ctx, s.db, "component_definitions", uuid, &componentdefinitionv1.ComponentDefinition{})
}

func (s *SQLiteStore) ListComponentDefinitions(ctx context.Context, filter string, pageSize int, pageToken string) ([]*componentdefinitionv1.ComponentDefinition, string, error) {
	return list(ctx, s.db, "component_definitions", filter, pageSize, pageToken, func() *componentdefinitionv1.ComponentDefinition {
		return &componentdefinitionv1.ComponentDefinition{}
	})
}

func (s *SQLiteStore) UpdateComponentDefinition(ctx context.Context, uuid string, cd *componentdefinitionv1.ComponentDefinition) error {
	return update(ctx, s.db, "component_definitions", uuid, cd)
}

func (s *SQLiteStore) DeleteComponentDefinition(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "component_definitions", uuid)
}

// ---- SSP ----

func (s *SQLiteStore) CreateSsp(ctx context.Context, ss *sspv1.SystemSecurityPlan) error {
	return create(ctx, s.db, "ssps", ss.Uuid.Value, ss)
}

func (s *SQLiteStore) GetSsp(ctx context.Context, uuid string) (*sspv1.SystemSecurityPlan, error) {
	return get(ctx, s.db, "ssps", uuid, &sspv1.SystemSecurityPlan{})
}

func (s *SQLiteStore) ListSsps(ctx context.Context, filter string, pageSize int, pageToken string) ([]*sspv1.SystemSecurityPlan, string, error) {
	return list(ctx, s.db, "ssps", filter, pageSize, pageToken, func() *sspv1.SystemSecurityPlan { return &sspv1.SystemSecurityPlan{} })
}

func (s *SQLiteStore) UpdateSsp(ctx context.Context, uuid string, ss *sspv1.SystemSecurityPlan) error {
	return update(ctx, s.db, "ssps", uuid, ss)
}

func (s *SQLiteStore) DeleteSsp(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "ssps", uuid)
}

// ---- Assessment Plan ----

func (s *SQLiteStore) CreateAssessmentPlan(ctx context.Context, ap *assessmentplanv1.AssessmentPlan) error {
	return create(ctx, s.db, "assessment_plans", ap.Uuid.Value, ap)
}

func (s *SQLiteStore) GetAssessmentPlan(ctx context.Context, uuid string) (*assessmentplanv1.AssessmentPlan, error) {
	return get(ctx, s.db, "assessment_plans", uuid, &assessmentplanv1.AssessmentPlan{})
}

func (s *SQLiteStore) ListAssessmentPlans(ctx context.Context, filter string, pageSize int, pageToken string) ([]*assessmentplanv1.AssessmentPlan, string, error) {
	return list(ctx, s.db, "assessment_plans", filter, pageSize, pageToken, func() *assessmentplanv1.AssessmentPlan {
		return &assessmentplanv1.AssessmentPlan{}
	})
}

func (s *SQLiteStore) UpdateAssessmentPlan(ctx context.Context, uuid string, ap *assessmentplanv1.AssessmentPlan) error {
	return update(ctx, s.db, "assessment_plans", uuid, ap)
}

func (s *SQLiteStore) DeleteAssessmentPlan(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "assessment_plans", uuid)
}

// ---- Assessment Results ----

func (s *SQLiteStore) CreateAssessmentResults(ctx context.Context, ar *assessmentresultsv1.AssessmentResults) error {
	return create(ctx, s.db, "assessment_results", ar.Uuid.Value, ar)
}

func (s *SQLiteStore) GetAssessmentResults(ctx context.Context, uuid string) (*assessmentresultsv1.AssessmentResults, error) {
	return get(ctx, s.db, "assessment_results", uuid, &assessmentresultsv1.AssessmentResults{})
}

func (s *SQLiteStore) ListAssessmentResults(ctx context.Context, filter string, pageSize int, pageToken string) ([]*assessmentresultsv1.AssessmentResults, string, error) {
	return list(ctx, s.db, "assessment_results", filter, pageSize, pageToken, func() *assessmentresultsv1.AssessmentResults {
		return &assessmentresultsv1.AssessmentResults{}
	})
}

func (s *SQLiteStore) UpdateAssessmentResults(ctx context.Context, uuid string, ar *assessmentresultsv1.AssessmentResults) error {
	return update(ctx, s.db, "assessment_results", uuid, ar)
}

func (s *SQLiteStore) DeleteAssessmentResults(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "assessment_results", uuid)
}

// ---- POAM ----

func (s *SQLiteStore) CreatePoam(ctx context.Context, p *poamv1.PlanOfActionAndMilestones) error {
	return create(ctx, s.db, "poams", p.Uuid.Value, p)
}

func (s *SQLiteStore) GetPoam(ctx context.Context, uuid string) (*poamv1.PlanOfActionAndMilestones, error) {
	return get(ctx, s.db, "poams", uuid, &poamv1.PlanOfActionAndMilestones{})
}

func (s *SQLiteStore) ListPoams(ctx context.Context, filter string, pageSize int, pageToken string) ([]*poamv1.PlanOfActionAndMilestones, string, error) {
	return list(ctx, s.db, "poams", filter, pageSize, pageToken, func() *poamv1.PlanOfActionAndMilestones {
		return &poamv1.PlanOfActionAndMilestones{}
	})
}

func (s *SQLiteStore) UpdatePoam(ctx context.Context, uuid string, p *poamv1.PlanOfActionAndMilestones) error {
	return update(ctx, s.db, "poams", uuid, p)
}

func (s *SQLiteStore) DeletePoam(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "poams", uuid)
}

// ---- Mapping ----

func (s *SQLiteStore) CreateMapping(ctx context.Context, m *mappingv1.MappingCollection) error {
	return create(ctx, s.db, "mappings", m.Uuid.Value, m)
}

func (s *SQLiteStore) GetMapping(ctx context.Context, uuid string) (*mappingv1.MappingCollection, error) {
	return get(ctx, s.db, "mappings", uuid, &mappingv1.MappingCollection{})
}

func (s *SQLiteStore) ListMappings(ctx context.Context, filter string, pageSize int, pageToken string) ([]*mappingv1.MappingCollection, string, error) {
	return list(ctx, s.db, "mappings", filter, pageSize, pageToken, func() *mappingv1.MappingCollection {
		return &mappingv1.MappingCollection{}
	})
}

func (s *SQLiteStore) UpdateMapping(ctx context.Context, uuid string, m *mappingv1.MappingCollection) error {
	return update(ctx, s.db, "mappings", uuid, m)
}

func (s *SQLiteStore) DeleteMapping(ctx context.Context, uuid string) error {
	return del(ctx, s.db, "mappings", uuid)
}

// ---- Search ----

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
