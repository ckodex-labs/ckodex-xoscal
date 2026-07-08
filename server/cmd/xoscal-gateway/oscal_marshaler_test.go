package main

import (
	"testing"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testUUID = "123e4567-e89b-42d3-a456-426614174000"

// TestOSCALMarshalerCatalog verifies the gateway marshaler produces
// schema-valid JSON for a Catalog message.
func TestOSCALMarshalerCatalog(t *testing.T) {
	m := newOSCALMarshaler()
	cat := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Catalog",
			Version: "1.0",
		},
		Controls: []*catalogv1.Control{{
			Id:    &commonv1.Token{Value: "ctrl-1"},
			Title: &commonv1.MarkupLine{Value: "Control 1"},
		}},
	}
	b, err := m.Marshal(cat)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindCatalog); err != nil {
		t.Errorf("catalog failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerProfile verifies the gateway marshaler produces
// schema-valid JSON for a Profile message.
func TestOSCALMarshalerProfile(t *testing.T) {
	m := newOSCALMarshaler()
	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Profile",
			Version: "1.0",
		},
		Imports: []*profilev1.Import{{
			Href: &commonv1.URIReference{Value: "https://example.gov/catalog.json"},
		}},
	}
	b, err := m.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindProfile); err != nil {
		t.Errorf("profile failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerComponentDefinition verifies the gateway marshaler
// produces schema-valid JSON for a ComponentDefinition message.
func TestOSCALMarshalerComponentDefinition(t *testing.T) {
	m := newOSCALMarshaler()
	c := &componentv1.ComponentDefinition{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Component Definition",
			Version: "1.0",
		},
	}
	b, err := m.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindComponentDefinition); err != nil {
		t.Errorf("component-definition failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerAssessmentPlan verifies the gateway marshaler
// produces schema-valid JSON for an AssessmentPlan message.
func TestOSCALMarshalerAssessmentPlan(t *testing.T) {
	m := newOSCALMarshaler()
	ap := &assessment_planv1.AssessmentPlan{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test AP",
			Version: "1.0",
		},
		ImportSsp: &assessment_planv1.ImportSsp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ssp.json"},
		},
		ReviewedControls: &assessment_planv1.ReviewedControls{
			ControlSelections: []*assessment_planv1.ControlSelection{{
				Description: &commonv1.MarkupMultiline{Value: "All controls"},
			}},
		},
	}
	b, err := m.Marshal(ap)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerAssessmentResults verifies the gateway marshaler
// produces schema-valid JSON for an AssessmentResults message.
func TestOSCALMarshalerAssessmentResults(t *testing.T) {
	m := newOSCALMarshaler()
	ar := &assessment_resultsv1.AssessmentResults{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test AR",
			Version: "1.0",
		},
		ImportAp: &assessment_resultsv1.ImportAp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ap.json"},
		},
		Results: []*assessment_resultsv1.Result{{
			Uuid:        &commonv1.UUID{Value: testUUID},
			Title:       &commonv1.MarkupLine{Value: "Result 1"},
			Description: &commonv1.MarkupMultiline{Value: "Test result"},
			Start:       &commonv1.DateTime{Value: timestamppb.Now()},
			ReviewedControls: &assessment_resultsv1.ReviewedControls{
				ControlSelections: []*assessment_resultsv1.ControlSelection{{
					Description: &commonv1.MarkupMultiline{Value: "All controls"},
				}},
			},
		}},
	}
	b, err := m.Marshal(ar)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerPOAM verifies the gateway marshaler
// produces schema-valid JSON for a POAM message.
func TestOSCALMarshalerPOAM(t *testing.T) {
	m := newOSCALMarshaler()
	poam := &poamv1.PlanOfActionAndMilestones{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test POAM",
			Version: "1.0",
		},
		PoamItems: []*poamv1.PoamItem{{
			Uuid:        &commonv1.UUID{Value: testUUID},
			Title:       &commonv1.MarkupLine{Value: "POAM Item 1"},
			Description: &commonv1.MarkupMultiline{Value: "A POAM item"},
		}},
	}
	b, err := m.Marshal(poam)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindPOAM); err != nil {
		t.Errorf("poam failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerSSP verifies the gateway marshaler
// produces schema-valid JSON for an SSP message.
func TestOSCALMarshalerSSP(t *testing.T) {
	m := newOSCALMarshaler()
	ssp := &sspv1.SystemSecurityPlan{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test SSP",
			Version: "1.0",
		},
		ImportProfile: &sspv1.ImportProfile{
			Href: &commonv1.URIReference{Value: "https://example.gov/profile.json"},
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds:     []*sspv1.SystemId{{Id: "test-system"}},
			SystemName:    "Test System",
			Description:   &commonv1.MarkupMultiline{Value: "A test system."},
			SystemInformation: &sspv1.SystemInformation{
				InformationTypes: []*sspv1.InformationType{{
					Uuid:        &commonv1.UUID{Value: testUUID},
					Title:       &commonv1.MarkupLine{Value: "Test info type"},
					Description: &commonv1.MarkupMultiline{Value: "Test information type"},
				}},
			},
			Status:                &sspv1.Status{State: "operational"},
			AuthorizationBoundary: &sspv1.AuthorizationBoundary{Description: &commonv1.MarkupMultiline{Value: "Boundary"}},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Users: []*sspv1.SystemUser{{
				Uuid:  &commonv1.UUID{Value: testUUID},
				Title: "Admin",
			}},
			Components: []*sspv1.SystemComponent{{
				Uuid:        &commonv1.UUID{Value: testUUID},
				Type:        "software",
				Title:       "App",
				Description: "App component",
				Status:      &sspv1.Status{State: "operational"},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			Description: &commonv1.MarkupMultiline{Value: "Implementation"},
			ImplementedRequirements: []*sspv1.ImplementedRequirement{{
				Uuid:      &commonv1.UUID{Value: testUUID},
				ControlId: &commonv1.Token{Value: "ac-1"},
			}},
		},
	}
	b, err := m.Marshal(ssp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindSSP); err != nil {
		t.Errorf("ssp failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerNonOSCALFallback verifies the marshaler delegates
// non-OSCAL types to the default protojson marshaler.
func TestOSCALMarshalerNonOSCALFallback(t *testing.T) {
	m := newOSCALMarshaler()
	// A plain map is not a proto.Message, so it should fall through to
	// the embedded JSONPb which handles non-proto values via json.Marshal.
	b, err := m.Marshal(map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("Marshal non-OSCAL: %v", err)
	}
	if string(b) == "" {
		t.Error("expected non-empty JSON output for non-OSCAL type")
	}
}

// TestOSCALMarshalerName verifies the marshaler Name method.
func TestOSCALMarshalerName(t *testing.T) {
	m := newOSCALMarshaler()
	if m.Name() != "oscal-json" {
		t.Errorf("expected name 'oscal-json', got %q", m.Name())
	}
}

// TestOSCALMarshalerContentType verifies the marshaler ContentType method.
func TestOSCALMarshalerContentType(t *testing.T) {
	m := newOSCALMarshaler()
	if ct := m.ContentType(nil); ct != "application/json" {
		t.Errorf("expected 'application/json', got %q", ct)
	}
}

// mustValidator creates a Validator or fails the test.
func mustValidator(t *testing.T) *schemavalidate.Validator {
	t.Helper()
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return v
}
