package oscal

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/kg"
)

// mockStore implements kg.Store for testing.
type mockStore struct {
	entities []*kg.Entity
}

func (m *mockStore) CreateEntity(ctx context.Context, entity *kg.Entity) error     { return nil }
func (m *mockStore) GetEntity(ctx context.Context, urn string) (*kg.Entity, error) { return nil, nil }
func (m *mockStore) UpdateEntity(ctx context.Context, entity *kg.Entity) error     { return nil }
func (m *mockStore) ListEntities(ctx context.Context, entityType string, status kg.EntityStatus) ([]*kg.Entity, error) {
	return nil, nil
}
func (m *mockStore) Snapshot(ctx context.Context, t time.Time) ([]*kg.Entity, error) { return nil, nil }
func (m *mockStore) CreateSnapshot(ctx context.Context, name string) (*kg.Snapshot, error) {
	return nil, nil
}
func (m *mockStore) GetSnapshot(ctx context.Context, name string) ([]*kg.Entity, error) {
	return m.entities, nil
}
func (m *mockStore) ListSnapshots(ctx context.Context) ([]*kg.Snapshot, error) { return nil, nil }
func (m *mockStore) CreateRelease(ctx context.Context, name, snapshotName string) (*kg.Release, error) {
	return nil, nil
}
func (m *mockStore) GetRelease(ctx context.Context, name string) (*kg.Release, error) {
	return nil, nil
}
func (m *mockStore) ListReleases(ctx context.Context) ([]*kg.Release, error) { return nil, nil }
func (m *mockStore) Close() error                                            { return nil }

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestGenerateCatalog_EmptySnapshot(t *testing.T) {
	g := NewGenerator(&mockStore{})
	cat, err := g.GenerateCatalog(context.Background(), "empty", "ISO42001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cat == nil {
		t.Fatal("expected non-nil catalog")
	}
	if len(cat.Controls) != 0 {
		t.Fatalf("expected 0 controls, got %d", len(cat.Controls))
	}
	if cat.BackMatter == nil || len(cat.BackMatter.Resources) == 0 {
		t.Error("expected BackMatter with resources")
	}
}

func TestGenerateCatalog_CrossFrameworkFiltering(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1", Text: "text",
			Framework: "ISO42001", Assessable: true,
		})},
		{URN: "req-2", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-2", Citation: "ctrl-2", Title: "Control 2", Text: "text",
			Framework: "NIST", Assessable: true,
		})},
	}}
	g := NewGenerator(store)
	cat, err := g.GenerateCatalog(context.Background(), "snap", "ISO42001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cat.Controls) != 1 {
		t.Fatalf("expected 1 control, got %d", len(cat.Controls))
	}
	if cat.Controls[0].Id.Value != "ctrl-1" {
		t.Fatalf("expected ctrl-1, got %s", cat.Controls[0].Id.Value)
	}
}

func TestGenerateCatalog_ParametersForSection(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1", Text: "text",
			Framework: "ISO42001", Assessable: true, Section: "4.1",
		})},
	}}
	g := NewGenerator(store)
	cat, err := g.GenerateCatalog(context.Background(), "snap", "ISO42001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cat.Controls) != 1 {
		t.Fatalf("expected 1 control, got %d", len(cat.Controls))
	}
	if len(cat.Controls[0].Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(cat.Controls[0].Params))
	}
	if cat.Controls[0].Params[0].Id.Value != "ctrl-1-param" {
		t.Fatalf("expected ctrl-1-param, got %s", cat.Controls[0].Params[0].Id.Value)
	}
}

func TestGeneratePOAM_HighRiskRemediation(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "High Risk Control", Text: "critical",
			Framework: "ISO42001", Assessable: true, RiskLevel: "high", Role: "security-officer",
		})},
		{URN: "req-2", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-2", Citation: "ctrl-2", Title: "Low Risk Control", Text: "minor",
			Framework: "ISO42001", Assessable: true, RiskLevel: "low",
		})},
	}}
	g := NewGenerator(store)
	poam, err := g.GeneratePOAM(context.Background(), "snap", "ISO42001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(poam.Risks) != 2 {
		t.Fatalf("expected 2 risks, got %d", len(poam.Risks))
	}
	// High risk should have remediation.
	highRisk := poam.Risks[0]
	if len(highRisk.Remediations) == 0 {
		t.Fatal("expected high-risk risk to have remediations")
	}
	resp := highRisk.Remediations[0]
	if resp.Status.State != "open" {
		t.Fatalf("expected open status, got %s", resp.Status.State)
	}
	if len(resp.ResponsibleRoles) != 1 {
		t.Fatalf("expected 1 responsible role, got %d", len(resp.ResponsibleRoles))
	}
	if resp.ResponsibleRoles[0].RoleId.Value != "security-officer" {
		t.Fatalf("expected security-officer, got %s", resp.ResponsibleRoles[0].RoleId.Value)
	}
	// Low risk should have no remediation.
	lowRisk := poam.Risks[1]
	if len(lowRisk.Remediations) != 0 {
		t.Fatalf("expected 0 remediations for low risk, got %d", len(lowRisk.Remediations))
	}
}

func TestGenerateAllArtifacts_BackMatterPresent(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1", Text: "text",
			Framework: "ISO42001", Assessable: true,
		})},
	}}
	g := NewGenerator(store)
	res, err := g.GenerateAllArtifacts(context.Background(), "snap", "ISO42001", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Catalog.BackMatter == nil {
		t.Error("expected catalog BackMatter")
	}
	if res.Profile.BackMatter == nil {
		t.Error("expected profile BackMatter")
	}
	if res.SSP.BackMatter == nil {
		t.Error("expected ssp BackMatter")
	}
	if res.ComponentDefinition.BackMatter == nil {
		t.Error("expected component definition BackMatter")
	}
	if res.AssessmentPlan.BackMatter == nil {
		t.Error("expected assessment plan BackMatter")
	}
	if res.AssessmentResults.BackMatter == nil {
		t.Error("expected assessment results BackMatter")
	}
	if res.POAM.BackMatter == nil {
		t.Error("expected poam BackMatter")
	}
}

func TestGenerateSSP_ResponsibleParties(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1", Text: "text",
			Framework: "ISO42001", Assessable: true, Role: "owner",
		})},
		{URN: "req-2", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-2", Citation: "ctrl-2", Title: "Control 2", Text: "text",
			Framework: "ISO42001", Assessable: true, Role: "owner",
		})},
		{URN: "req-3", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-3", Citation: "ctrl-3", Title: "Control 3", Text: "text",
			Framework: "ISO42001", Assessable: true, Role: "admin",
		})},
	}}
	g := NewGenerator(store)
	ssp, err := g.GenerateSSP(context.Background(), "snap", "ISO42001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ssp.Metadata.ResponsibleParties) != 2 {
		t.Fatalf("expected 2 responsible parties, got %d", len(ssp.Metadata.ResponsibleParties))
	}
	roles := make(map[string]bool)
	for _, rp := range ssp.Metadata.ResponsibleParties {
		roles[rp.RoleId.Value] = true
	}
	if !roles["owner"] {
		t.Error("expected owner role")
	}
	if !roles["admin"] {
		t.Error("expected admin role")
	}
}
