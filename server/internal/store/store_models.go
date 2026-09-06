package store

import (
	"context"

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
