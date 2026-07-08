package schemavalidate

import (
	"strings"
	"testing"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestProtoToSchemaConformance validates that OSCAL artifacts generated from
// proto definitions and serialized via the OSCAL JSON exporters pass JSON
// schema validation. This is the proto-to-schema conformance test.
//
// If any of these tests fail, it means the proto definitions, the generator,
// or the serializer produce JSON that does not conform to the OSCAL 1.1.2
// JSON schema.

func TestProtoConformanceCatalog(t *testing.T) {
	v := mustValidator(t)
	cat := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance Catalog",
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

func TestProtoConformanceProfile(t *testing.T) {
	v := mustValidator(t)
	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance Profile",
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

func TestProtoConformanceComponentDefinition(t *testing.T) {
	v := mustValidator(t)
	c := &componentv1.ComponentDefinition{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance Component Definition",
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

func TestProtoConformanceAssessmentPlan(t *testing.T) {
	v := mustValidator(t)
	ap := &assessment_planv1.AssessmentPlan{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance Assessment Plan",
			Version: "1.0",
		},
		ImportSsp: &assessment_planv1.ImportSsp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ssp.json"},
		},
		ReviewedControls: &assessment_planv1.ReviewedControls{
			ControlSelections: []*assessment_planv1.ControlSelection{{
				Description: &commonv1.MarkupMultiline{Value: "All controls reviewed"},
			}},
		},
	}
	b, err := oscal.ExportAssessmentPlanJSON(ap)
	if err != nil {
		t.Fatalf("ExportAssessmentPlanJSON: %v", err)
	}
	if err := v.Validate(b, KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}
}

func TestProtoConformanceAssessmentResults(t *testing.T) {
	v := mustValidator(t)
	ar := &assessment_resultsv1.AssessmentResults{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance Assessment Results",
			Version: "1.0",
		},
		ImportAp: &assessment_resultsv1.ImportAp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ap.json"},
		},
		Results: []*assessment_resultsv1.Result{{
			Uuid:        &commonv1.UUID{Value: validUUID},
			Title:       &commonv1.MarkupLine{Value: "Result 1"},
			Description: &commonv1.MarkupMultiline{Value: "Test result"},
			Start:       &commonv1.DateTime{Value: timestamppb.Now()},
			ReviewedControls: &assessment_resultsv1.ReviewedControls{
				ControlSelections: []*assessment_resultsv1.ControlSelection{{
					Description: &commonv1.MarkupMultiline{Value: "All controls reviewed"},
				}},
			},
		}},
	}
	b, err := oscal.ExportAssessmentResultsJSON(ar)
	if err != nil {
		t.Fatalf("ExportAssessmentResultsJSON: %v", err)
	}
	if err := v.Validate(b, KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}
}

func TestProtoConformancePOAM(t *testing.T) {
	v := mustValidator(t)
	poam := &poamv1.PlanOfActionAndMilestones{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance POAM",
			Version: "1.0",
		},
		PoamItems: []*poamv1.PoamItem{{
			Uuid:        &commonv1.UUID{Value: validUUID},
			Title:       &commonv1.MarkupLine{Value: "POAM Item 1"},
			Description: &commonv1.MarkupMultiline{Value: "A POAM item"},
		}},
	}
	b, err := oscal.ExportPOAMJSON(poam)
	if err != nil {
		t.Fatalf("ExportPOAMJSON: %v", err)
	}
	if err := v.Validate(b, KindPOAM); err != nil {
		t.Errorf("poam failed schema validation: %v", err)
	}
}

func TestProtoConformanceSSP(t *testing.T) {
	v := mustValidator(t)
	// SSP requires deeply nested required fields. The proto models most of
	// these, but the OSCAL schema requires system-information with
	// information-types, and system-implementation with users and components
	// that have status fields. We build a minimal but schema-valid SSP.
	ssp := &sspv1.SystemSecurityPlan{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance SSP",
			Version: "1.0",
		},
		ImportProfile: &sspv1.ImportProfile{
			Href: &commonv1.URIReference{Value: "https://example.gov/profile.json"},
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds: []*sspv1.SystemId{{Id: "test-system"}},
			SystemName: "Test System",
			Description: &commonv1.MarkupMultiline{Value: "A test system."},
			SystemInformation: &sspv1.SystemInformation{
				InformationTypes: []*sspv1.InformationType{{
					Uuid:        &commonv1.UUID{Value: validUUID},
					Title:       &commonv1.MarkupLine{Value: "Test info type"},
					Description: &commonv1.MarkupMultiline{Value: "Test information type"},
				}},
			},
			Status:                 &sspv1.Status{State: "operational"},
			AuthorizationBoundary:  &sspv1.AuthorizationBoundary{Description: &commonv1.MarkupMultiline{Value: "Boundary"}},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Users: []*sspv1.SystemUser{{
				Uuid:  &commonv1.UUID{Value: validUUID},
				Title: "Admin",
			}},
			Components: []*sspv1.SystemComponent{{
				Uuid:        &commonv1.UUID{Value: validUUID},
				Type:        "software",
				Title:       "App",
				Description: "App component",
				Status:      &sspv1.Status{State: "operational"},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			Description: &commonv1.MarkupMultiline{Value: "Implementation"},
			ImplementedRequirements: []*sspv1.ImplementedRequirement{{
				Uuid:      &commonv1.UUID{Value: validUUID},
				ControlId: &commonv1.Token{Value: "ac-1"},
			}},
		},
	}
	b, err := oscal.ExportSSPJSON(ssp)
	if err != nil {
		t.Fatalf("ExportSSPJSON: %v", err)
	}
	if err := v.Validate(b, KindSSP); err != nil {
		t.Errorf("ssp failed schema validation: %v", err)
	}
}

// mustValidator creates a Validator or fails the test.
func mustValidator(t *testing.T) *Validator {
	t.Helper()
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return v
}

// TestDeprecatedFieldsSuppressed verifies that deprecated proto fields are
// stripped from the OSCAL JSON output and do not cause schema validation
// failures due to additionalProperties:false constraints.
func TestDeprecatedFieldsSuppressed(t *testing.T) {
	// POAM with both risks (deprecated) and poam_items populated.
	// The risks field must be stripped; only poam-items should appear.
	poam := &poamv1.PlanOfActionAndMilestones{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Deprecated Field Test POAM",
			Version: "1.0",
		},
		Risks: []*poamv1.Risk{{
			Uuid:        &commonv1.UUID{Value: validUUID},
			Title:       &commonv1.MarkupLine{Value: "Old Risk"},
			Description: &commonv1.MarkupMultiline{Value: "This should be stripped"},
			Statement:   &commonv1.MarkupMultiline{Value: "Statement"},
			Status:      &poamv1.RiskStatus{State: "open"},
		}},
		PoamItems: []*poamv1.PoamItem{{
			Uuid:        &commonv1.UUID{Value: validUUID},
			Title:       &commonv1.MarkupLine{Value: "POAM Item 1"},
			Description: &commonv1.MarkupMultiline{Value: "A POAM item"},
		}},
	}
	b, err := oscal.ExportPOAMJSON(poam)
	if err != nil {
		t.Fatalf("ExportPOAMJSON: %v", err)
	}

	// Verify "risks" does not appear in the JSON output
	jsonStr := string(b)
	if strings.Contains(jsonStr, "\"risks\"") {
		t.Errorf("deprecated 'risks' field should be stripped from POAM JSON, but found it in output:\n%s", jsonStr)
	}
	// Verify "poam-items" does appear
	if !strings.Contains(jsonStr, "\"poam-items\"") {
		t.Errorf("expected 'poam-items' in POAM JSON, but not found:\n%s", jsonStr)
	}

	// Validate against schema
	v := mustValidator(t)
	if err := v.Validate(b, KindPOAM); err != nil {
		t.Errorf("POAM with deprecated risks field failed schema validation: %v", err)
	}
}
