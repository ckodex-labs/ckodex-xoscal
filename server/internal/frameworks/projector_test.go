package frameworks_test

import (
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestProjectAllLayers_SchemaAndIntegrityConformance(t *testing.T) {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	reg := frameworks.DefaultRegistry()
	testFWs := []string{"cybersecure-canada", "cccs-itsp-10-033"}

	for _, fwID := range testFWs {
		fw, ok := reg.Get(fwID)
		if !ok {
			t.Fatalf("framework %s not found", fwID)
		}

		t.Run(fwID, func(t *testing.T) {
			layers, err := frameworks.ProjectAllLayers(fw)
			if err != nil {
				t.Fatalf("ProjectAllLayers failed: %v", err)
			}

			// 1. Catalog
			t.Run("Catalog", func(t *testing.T) {
				if err := v.Validate(layers.Catalog, schemavalidate.KindCatalog); err != nil {
					t.Errorf("Catalog schema error: %v", err)
				}
				findings, err := integrity.Check(layers.Catalog, schemavalidate.KindCatalog)
				if err != nil {
					t.Errorf("Catalog integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("Catalog integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 2. Profile
			t.Run("Profile", func(t *testing.T) {
				if err := v.Validate(layers.Profile, schemavalidate.KindProfile); err != nil {
					t.Errorf("Profile schema error: %v", err)
				}
				findings, err := integrity.Check(layers.Profile, schemavalidate.KindProfile)
				if err != nil {
					t.Errorf("Profile integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("Profile integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 3. Component Definition
			t.Run("ComponentDefinition", func(t *testing.T) {
				if err := v.Validate(layers.ComponentDefinition, schemavalidate.KindComponentDefinition); err != nil {
					t.Errorf("ComponentDefinition schema error: %v", err)
				}
				findings, err := integrity.Check(layers.ComponentDefinition, schemavalidate.KindComponentDefinition)
				if err != nil {
					t.Errorf("ComponentDefinition integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("ComponentDefinition integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 4. System Security Plan
			t.Run("SystemSecurityPlan", func(t *testing.T) {
				if err := v.Validate(layers.SystemSecurityPlan, schemavalidate.KindSSP); err != nil {
					t.Errorf("SystemSecurityPlan schema error: %v", err)
				}
				findings, err := integrity.Check(layers.SystemSecurityPlan, schemavalidate.KindSSP)
				if err != nil {
					t.Errorf("SystemSecurityPlan integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("SystemSecurityPlan integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 5. Assessment Plan
			t.Run("AssessmentPlan", func(t *testing.T) {
				if err := v.Validate(layers.AssessmentPlan, schemavalidate.KindAssessmentPlan); err != nil {
					t.Errorf("AssessmentPlan schema error: %v", err)
				}
				findings, err := integrity.Check(layers.AssessmentPlan, schemavalidate.KindAssessmentPlan)
				if err != nil {
					t.Errorf("AssessmentPlan integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("AssessmentPlan integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 6. Assessment Results
			t.Run("AssessmentResults", func(t *testing.T) {
				if err := v.Validate(layers.AssessmentResults, schemavalidate.KindAssessmentResults); err != nil {
					t.Errorf("AssessmentResults schema error: %v", err)
				}
				findings, err := integrity.Check(layers.AssessmentResults, schemavalidate.KindAssessmentResults)
				if err != nil {
					t.Errorf("AssessmentResults integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("AssessmentResults integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})

			// 7. POAM
			t.Run("POAM", func(t *testing.T) {
				if err := v.Validate(layers.POAM, schemavalidate.KindPOAM); err != nil {
					t.Errorf("POAM schema error: %v", err)
				}
				findings, err := integrity.Check(layers.POAM, schemavalidate.KindPOAM)
				if err != nil {
					t.Errorf("POAM integrity error: %v", err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("POAM integrity error: %s: %s (%s)", f.Rule, f.Message, f.Path)
					}
				}
			})
		})
	}
}
