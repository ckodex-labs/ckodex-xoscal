package oscal

import (
	"context"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// TestEndToEndSchemaConformance generates all OSCAL artifacts from mock
// requirements, exports each to JSON via the OSCAL exporters, and validates
// every artifact against the official OSCAL 1.1.2 JSON schema.
//
// This is the end-to-end conformance test: generator → exporter → schema.
// If any artifact fails validation, the proto definitions, generator logic,
// or serializer have a conformance bug.
func TestEndToEndSchemaConformance(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1", Text: "text",
			Framework: "ISO42001", Assessable: true,
		})},
		{URN: "req-2", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-2", Citation: "ctrl-2", Title: "Control 2", Text: "text",
			Framework: "ISO42001", Assessable: true, RiskLevel: "high", Role: "security-officer",
		})},
	}}
	g := NewGenerator(store)
	ctx := context.Background()

	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	// Catalog
	cat, err := g.GenerateCatalog(ctx, "snap", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateCatalog: %v", err)
	}
	if b, err := ExportCatalogJSON(cat); err != nil {
		t.Fatalf("ExportCatalogJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindCatalog); err != nil {
		t.Errorf("catalog failed schema validation: %v", err)
	}

	// Profile
	prof, err := g.GenerateProfile(ctx, "snap", "ISO42001", nil)
	if err != nil {
		t.Fatalf("GenerateProfile: %v", err)
	}
	if b, err := ExportProfileJSON(prof); err != nil {
		t.Fatalf("ExportProfileJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindProfile); err != nil {
		t.Errorf("profile failed schema validation: %v", err)
	}

	// Component Definition
	cd, err := g.GenerateComponentDefinition(ctx, "snap", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateComponentDefinition: %v", err)
	}
	if b, err := ExportComponentDefinitionJSON(cd); err != nil {
		t.Fatalf("ExportComponentDefinitionJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindComponentDefinition); err != nil {
		t.Errorf("component-definition failed schema validation: %v", err)
	}

	// Assessment Plan
	ap, err := g.GenerateAssessmentPlan(ctx, "snap", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateAssessmentPlan: %v", err)
	}
	if b, err := ExportAssessmentPlanJSON(ap); err != nil {
		t.Fatalf("ExportAssessmentPlanJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}

	// Assessment Results
	ar, err := g.GenerateAssessmentResults(ctx, "snap", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateAssessmentResults: %v", err)
	}
	if b, err := ExportAssessmentResultsJSON(ar); err != nil {
		t.Fatalf("ExportAssessmentResultsJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}

	// POAM
	poam, err := g.GeneratePOAM(ctx, "snap", "ISO42001")
	if err != nil {
		t.Fatalf("GeneratePOAM: %v", err)
	}
	if b, err := ExportPOAMJSON(poam); err != nil {
		t.Fatalf("ExportPOAMJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindPOAM); err != nil {
		t.Errorf("poam failed schema validation: %v", err)
	}
}

// TestEndToEndEmptySnapshotConformance validates that even an empty snapshot
// (no requirements) produces schema-valid OSCAL artifacts.
func TestEndToEndEmptySnapshotConformance(t *testing.T) {
	g := NewGenerator(&mockStore{})
	ctx := context.Background()

	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	// Catalog with no controls
	cat, err := g.GenerateCatalog(ctx, "empty", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateCatalog: %v", err)
	}
	if b, err := ExportCatalogJSON(cat); err != nil {
		t.Fatalf("ExportCatalogJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindCatalog); err != nil {
		t.Errorf("empty catalog failed schema validation: %v", err)
	}

	// Profile with no imports content
	prof, err := g.GenerateProfile(ctx, "empty", "ISO42001", nil)
	if err != nil {
		t.Fatalf("GenerateProfile: %v", err)
	}
	if b, err := ExportProfileJSON(prof); err != nil {
		t.Fatalf("ExportProfileJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindProfile); err != nil {
		t.Errorf("empty profile failed schema validation: %v", err)
	}

	// Component Definition
	cd, err := g.GenerateComponentDefinition(ctx, "empty", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateComponentDefinition: %v", err)
	}
	if b, err := ExportComponentDefinitionJSON(cd); err != nil {
		t.Fatalf("ExportComponentDefinitionJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindComponentDefinition); err != nil {
		t.Errorf("empty component-definition failed schema validation: %v", err)
	}
}
