package schemavalidate

import (
	"strings"
	"testing"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/oscal"
)

// validUUID is a valid v4 UUID that passes the OSCAL UUID pattern.
const validUUID = "123e4567-e89b-42d3-a456-426614174000"

func TestValidatorLoadsAllSchemas(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	expectedKeys := []string{
		"catalog", "profile", "system-security-plan",
		"component-definition", "assessment-plan",
		"assessment-results", "plan-of-action-and-milestones",
	}
	for _, key := range expectedKeys {
		if _, ok := v.schemas[key]; !ok {
			t.Errorf("schema not loaded for root key: %s", key)
		}
	}
}

func TestValidateCatalog(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	cat := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Test Catalog",
			Version: "1.0",
		},
		Controls: []*catalogv1.Control{{
			Id:    &commonv1.Token{Value: "ctrl-1"},
			Title: &commonv1.MarkupLine{Value: "Control 1"},
			Parts: []*catalogv1.Part{{
				Id:    &commonv1.Token{Value: "ctrl-1-stmt"},
				Name:  "statement",
				Prose: []*commonv1.MarkupMultiline{{Value: "This is the control statement."}},
			}},
		}},
	}
	b, err := oscal.ExportCatalogJSON(cat)
	if err != nil {
		t.Fatalf("ExportCatalogJSON: %v", err)
	}
	if err := v.Validate(b, KindCatalog); err != nil {
		t.Errorf("catalog failed schema validation: %v", err)
	}
}

func TestValidateCatalogRejectsInvalid(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	// Missing root "catalog" wrapper — raw protobuf JSON
	invalid := `{"uuid":"123e4567-e89b-42d3-a456-426614174000","metadata":{"title":"x","version":"1.0"}}`
	err = v.Validate([]byte(invalid), KindCatalog)
	if err == nil {
		t.Fatal("expected validation error for invalid catalog, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation failure message, got: %v", err)
	}
}

func TestValidateProfile(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Test Profile",
			Version: "1.0",
		},
		Imports: []*profilev1.Import{{
			Href: &commonv1.URIReference{Value: "https://example.gov/catalog.json"},
		}},
	}
	b, err := oscal.ExportProfileJSON(p)
	if err != nil {
		t.Fatalf("ExportProfileJSON: %v", err)
	}
	if err := v.Validate(b, KindProfile); err != nil {
		t.Errorf("profile failed schema validation: %v", err)
	}
}

func TestValidateSSP(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	// SSP requires deeply nested required fields (system-characteristics with
	// system-ids, description, system-information, status, authorization-boundary;
	// system-implementation with users/components; control-implementation with
	// implemented-requirements). The proto doesn't fully model all of these yet.
	// Validate with hand-crafted minimal JSON that satisfies the schema.
	minimalSSP := `{
		"system-security-plan": {
			"uuid": "` + validUUID + `",
			"metadata": {
				"title": "Test SSP",
				"version": "1.0",
				"oscal-version": "1.1.2",
				"last-modified": "2026-01-01T00:00:00Z"
			},
			"import-profile": {"href": "https://example.gov/profile.json"},
			"system-characteristics": {
				"system-ids": [{"id": "test-system"}],
				"system-name": "Test System",
				"description": "A test system.",
				"system-information": {
					"information-types": [{"title": "Test info type", "description": "Test"}]
				},
				"status": {"state": "operational"},
				"authorization-boundary": {"description": "Boundary"}
			},
			"system-implementation": {
				"users": [{"uuid": "` + validUUID + `", "title": "Admin"}],
				"components": [{"uuid": "` + validUUID + `", "type": "software", "title": "App", "description": "App component", "status": {"state": "operational"}}]
			},
			"control-implementation": {
				"description": "Implementation",
				"implemented-requirements": [{"uuid": "` + validUUID + `", "control-id": "ac-1"}]
			}
		}
	}`
	if err := v.Validate([]byte(minimalSSP), KindSSP); err != nil {
		t.Errorf("ssp failed schema validation: %v", err)
	}
}

func TestValidateComponentDefinition(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	c := &componentv1.ComponentDefinition{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Test Component Definition",
			Version: "1.0",
		},
	}
	b, err := oscal.ExportComponentDefinitionJSON(c)
	if err != nil {
		t.Fatalf("ExportComponentDefinitionJSON: %v", err)
	}
	if err := v.Validate(b, KindComponentDefinition); err != nil {
		t.Errorf("component-definition failed schema validation: %v", err)
	}
}

func TestValidateAssessmentPlan(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	// Assessment plan requires reviewed-controls with control-selections.
	// The proto's ReviewedControls doesn't model control-selections yet.
	// Validate with hand-crafted minimal JSON.
	minimalAP := `{
		"assessment-plan": {
			"uuid": "` + validUUID + `",
			"metadata": {
				"title": "Test Assessment Plan",
				"version": "1.0",
				"oscal-version": "1.1.2",
				"last-modified": "2026-01-01T00:00:00Z"
			},
			"import-ssp": {"href": "https://example.gov/ssp.json"},
			"reviewed-controls": {
				"control-selections": [{"description": "All controls"}]
			}
		}
	}`
	if err := v.Validate([]byte(minimalAP), KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}
}

func TestValidateAssessmentResults(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	// Assessment results requires results with start and reviewed-controls.
	// The proto doesn't fully model these yet.
	// Validate with hand-crafted minimal JSON.
	minimalAR := `{
		"assessment-results": {
			"uuid": "` + validUUID + `",
			"metadata": {
				"title": "Test Assessment Results",
				"version": "1.0",
				"oscal-version": "1.1.2",
				"last-modified": "2026-01-01T00:00:00Z"
			},
			"import-ap": {"href": "https://example.gov/ap.json"},
			"results": [{
				"uuid": "` + validUUID + `",
				"title": "Result 1",
				"description": "Test result",
				"start": "2026-01-01T00:00:00Z",
				"reviewed-controls": {
					"control-selections": [{"description": "All controls"}]
				}
			}]
		}
	}`
	if err := v.Validate([]byte(minimalAR), KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}
}

func TestValidatePOAM(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	// POAM schema requires "poam-items" with at least one item.
	// The proto does not yet model poam-items.
	// Validate with hand-crafted minimal JSON.
	minimalPOAM := `{
		"plan-of-action-and-milestones": {
			"uuid": "` + validUUID + `",
			"metadata": {
				"title": "Test POAM",
				"version": "1.0",
				"oscal-version": "1.1.2",
				"last-modified": "2026-01-01T00:00:00Z"
			},
			"poam-items": [{
				"uuid": "` + validUUID + `",
				"title": "POAM Item 1",
				"description": "A POAM item"
			}]
		}
	}`
	if err := v.Validate([]byte(minimalPOAM), KindPOAM); err != nil {
		t.Errorf("poam failed schema validation: %v", err)
	}
}
