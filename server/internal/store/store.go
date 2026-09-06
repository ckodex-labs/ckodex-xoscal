package store

import (
	"context"
	"database/sql"
	"fmt"

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
