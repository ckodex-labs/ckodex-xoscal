package oscal

import (
	"context"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// TestEndToEndSchemaConformance generates all OSCAL artifacts from mock
// requirements, exports each to JSON via the OSCAL exporters, and validates
// every generated artifact against the pinned official OSCAL JSON schema.
//
// This is the end-to-end conformance test: generator → exporter → schema.
// If any artifact fails validation, the proto definitions, generator logic,
// or serializer have a conformance bug.
func TestEndToEndSchemaConformance(t *testing.T) {
	store := &mockStore{entities: []*kg.Entity{
		{URN: "group-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "group-1", NodeRefID: "sec-4", NodeName: "Section 4", Citation: "sec-4",
			Framework: "ISO42001", Assessable: false,
		})},
		{URN: "req-1", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-1", Citation: "ctrl-1", Title: "Control 1\nMulti-line Title", Text: "Statement line 1\nStatement line 2",
			Guidance:         "Guidance line 1\nGuidance line 2",
			CanadianGuidance: "GC Discussion line 1\nGC Discussion line 2",
			Framework:        "ISO42001", Assessable: true, Section: "sec-4", Subsection: "4.1", ParentURN: "group-1",
			ImplementationGroups: []string{"profile-selected"},
			Params:               []kg.ParamDef{{ID: "ctrl-1_prm_1", Label: "frequency\nof review", Values: []string{"annually\nor on change"}}},
			Props:                []kg.PropDef{{Name: "jurisdiction", Value: "ca", Ns: "https://ised-isde.canada.ca/ns/oscal"}},
		})},
		{URN: "req-2", Type: "reg:Requirement", Payload: mustMarshal(kg.Requirement{
			URN: "req-2", Citation: "ctrl-2", Title: "Control 2", Text: "text",
			Framework: "ISO42001", Assessable: true, RiskLevel: "high", Role: "security-officer", Section: "sec-4", ParentURN: "group-1",
		})},
	}}
	g := NewGenerator(store)
	ctx := context.Background()

	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	// Catalog (also verify case-insensitive framework matching)
	cat, err := g.GenerateCatalog(ctx, "snap", "iso42001")
	if err != nil {
		t.Fatalf("GenerateCatalog: %v", err)
	}
	if b, err := ExportCatalogJSON(cat); err != nil {
		t.Fatalf("ExportCatalogJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindCatalog); err != nil {
		t.Errorf("catalog failed schema validation: %v", err)
	}

	// Profile
	prof, err := g.GenerateProfile(ctx, "snap", "iso42001", nil)
	if err != nil {
		t.Fatalf("GenerateProfile: %v", err)
	}
	if b, err := ExportProfileJSON(prof); err != nil {
		t.Fatalf("ExportProfileJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindProfile); err != nil {
		t.Errorf("profile failed schema validation: %v", err)
	}

	// Component Definition
	cd, err := g.GenerateComponentDefinition(ctx, "snap", "iso42001")
	if err != nil {
		t.Fatalf("GenerateComponentDefinition: %v", err)
	}
	if b, err := ExportComponentDefinitionJSON(cd); err != nil {
		t.Fatalf("ExportComponentDefinitionJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindComponentDefinition); err != nil {
		t.Errorf("component-definition failed schema validation: %v", err)
	}

	// System Security Plan
	ssp, err := g.GenerateSSP(ctx, "snap", "iso42001")
	if err != nil {
		t.Fatalf("GenerateSSP: %v", err)
	}
	if b, err := ExportSSPJSON(ssp); err != nil {
		t.Fatalf("ExportSSPJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindSSP); err != nil {
		t.Errorf("ssp failed schema validation: %v", err)
	}

	// Assessment Plan
	ap, err := g.GenerateAssessmentPlan(ctx, "snap", "iso42001")
	if err != nil {
		t.Fatalf("GenerateAssessmentPlan: %v", err)
	}
	if b, err := ExportAssessmentPlanJSON(ap); err != nil {
		t.Fatalf("ExportAssessmentPlanJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}

	// Assessment Results
	ar, err := g.GenerateAssessmentResults(ctx, "snap", "iso42001")
	if err != nil {
		t.Fatalf("GenerateAssessmentResults: %v", err)
	}
	if b, err := ExportAssessmentResultsJSON(ar); err != nil {
		t.Fatalf("ExportAssessmentResultsJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}

	// POAM
	poam, err := g.GeneratePOAM(ctx, "snap", "iso42001")
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

	// System Security Plan
	ssp, err := g.GenerateSSP(ctx, "empty", "ISO42001")
	if err != nil {
		t.Fatalf("GenerateSSP: %v", err)
	}
	if b, err := ExportSSPJSON(ssp); err != nil {
		t.Fatalf("ExportSSPJSON: %v", err)
	} else if err := v.Validate(b, schemavalidate.KindSSP); err != nil {
		t.Errorf("empty ssp failed schema validation: %v", err)
	}
}
