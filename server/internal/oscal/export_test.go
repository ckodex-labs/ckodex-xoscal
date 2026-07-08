package oscal

import (
	"bytes"
	"testing"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
)

func TestExportAssessmentResultsJSON(t *testing.T) {
	ar := &assessment_resultsv1.AssessmentResults{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Assessment Results",
			Version: "1.0",
		},
		Results: []*assessment_resultsv1.Result{{
			Uuid:        &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174001"},
			Title:       &commonv1.MarkupLine{Value: "Result 1"},
			Description: &commonv1.MarkupMultiline{Value: "Test result"},
			Findings: []*assessment_resultsv1.Finding{{
				Uuid:  &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174002"},
				Title: &commonv1.MarkupLine{Value: "Finding 1"},
				Target: &assessment_resultsv1.FindingTarget{
					Type:     "control-objective",
					TargetId: &commonv1.Token{Value: "ctrl-1"},
				},
			}},
		}},
	}

	b, err := ExportAssessmentResultsJSON(ar)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"uuid"`)) {
		t.Error("expected JSON to contain uuid field")
	}
	if !bytes.Contains(b, []byte(`"Test Assessment Results"`)) {
		t.Error("expected JSON to contain title")
	}
	if !bytes.Contains(b, []byte(`"Finding 1"`)) {
		t.Error("expected JSON to contain finding title")
	}
}

func TestExportCatalogJSON(t *testing.T) {
	cat := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Catalog",
			Version: "1.0",
		},
		Controls: []*catalogv1.Control{{
			Id:    &commonv1.Token{Value: "ctrl-1"},
			Title: &commonv1.MarkupLine{Value: "Control 1"},
		}},
	}
	b, err := ExportCatalogJSON(cat)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	// OSCAL JSON must have root "catalog" wrapper
	if !bytes.Contains(b, []byte(`"catalog"`)) {
		t.Error("expected JSON to contain root 'catalog' wrapper key")
	}
	// UUID must be a plain string, not {"value": "..."}
	if !bytes.Contains(b, []byte(`"uuid": "123e4567-e89b-12d3-a456-426614174000"`)) {
		t.Error("expected JSON to contain plain uuid string")
	}
	if bytes.Contains(b, []byte(`{"value":`)) {
		t.Error("expected JSON to NOT contain {\"value\": ...} wrapper objects")
	}
	// Must have kebab-case keys
	if !bytes.Contains(b, []byte(`"oscal-version"`)) {
		t.Error("expected JSON to contain kebab-case 'oscal-version' field")
	}
	if !bytes.Contains(b, []byte(`"last-modified"`)) {
		t.Error("expected JSON to contain kebab-case 'last-modified' field")
	}
	// Must NOT have camelCase keys
	if bytes.Contains(b, []byte(`"backMatter"`)) {
		t.Error("expected JSON to NOT contain camelCase 'backMatter'")
	}
	if bytes.Contains(b, []byte(`"lastModified"`)) {
		t.Error("expected JSON to NOT contain camelCase 'lastModified'")
	}
}

func TestExportProfileJSON(t *testing.T) {
	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Profile",
			Version: "1.0",
		},
	}
	b, err := ExportProfileJSON(p)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"Test Profile"`)) {
		t.Error("expected JSON to contain title")
	}
}

func TestExportSSPJSON(t *testing.T) {
	ssp := &sspv1.SystemSecurityPlan{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test SSP",
			Version: "1.0",
		},
	}
	b, err := ExportSSPJSON(ssp)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"Test SSP"`)) {
		t.Error("expected JSON to contain title")
	}
}

func TestExportComponentDefinitionJSON(t *testing.T) {
	cd := &componentv1.ComponentDefinition{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Component Definition",
			Version: "1.0",
		},
	}
	b, err := ExportComponentDefinitionJSON(cd)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"Test Component Definition"`)) {
		t.Error("expected JSON to contain title")
	}
}

func TestExportAssessmentPlanJSON(t *testing.T) {
	ap := &assessment_planv1.AssessmentPlan{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Assessment Plan",
			Version: "1.0",
		},
	}
	b, err := ExportAssessmentPlanJSON(ap)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"Test Assessment Plan"`)) {
		t.Error("expected JSON to contain title")
	}
}

func TestExportPOAMJSON(t *testing.T) {
	poam := &poamv1.PlanOfActionAndMilestones{
		Uuid: &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Metadata: &commonv1.Metadata{
			Title:   "Test POAM",
			Version: "1.0",
		},
	}
	b, err := ExportPOAMJSON(poam)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"Test POAM"`)) {
		t.Error("expected JSON to contain title")
	}
}

func TestExportMappingsJSON(t *testing.T) {
	maps := []*mappingv1.Map{{
		Uuid:         &commonv1.UUID{Value: "123e4567-e89b-12d3-a456-426614174000"},
		Relationship: &commonv1.Token{Value: "equivalent"},
	}}
	b, err := ExportMappingsJSON(maps)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	if !bytes.Contains(b, []byte(`"equivalent"`)) {
		t.Error("expected JSON to contain relationship")
	}
}
