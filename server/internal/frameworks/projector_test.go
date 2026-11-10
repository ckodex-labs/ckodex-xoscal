package frameworks_test

import (
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestProjectAllLayers_SchemaAndIntegrityConformance(t *testing.T) {
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

			checkLayer := func(t *testing.T, name string, data []byte, kind schemavalidate.ArtifactKind) {
				t.Helper()
				if err := frameworks.ValidateSchemaCached(data, kind); err != nil {
					t.Errorf("%s schema error: %v", name, err)
				}
				findings, err := frameworks.CheckIntegrityCached(data, kind)
				if err != nil {
					t.Errorf("%s integrity error: %v", name, err)
				}
				for _, f := range findings {
					if f.Severity == integrity.SeverityError {
						t.Errorf("%s integrity error: %s: %s (%s)", name, f.Rule, f.Message, f.Path)
					}
				}
			}

			t.Run("Catalog", func(t *testing.T) { checkLayer(t, "Catalog", layers.Catalog, schemavalidate.KindCatalog) })
			t.Run("Profile", func(t *testing.T) { checkLayer(t, "Profile", layers.Profile, schemavalidate.KindProfile) })
			t.Run("ComponentDefinition", func(t *testing.T) {
				checkLayer(t, "ComponentDefinition", layers.ComponentDefinition, schemavalidate.KindComponentDefinition)
			})
			t.Run("SystemSecurityPlan", func(t *testing.T) {
				checkLayer(t, "SystemSecurityPlan", layers.SystemSecurityPlan, schemavalidate.KindSSP)
			})
			t.Run("AssessmentPlan", func(t *testing.T) {
				checkLayer(t, "AssessmentPlan", layers.AssessmentPlan, schemavalidate.KindAssessmentPlan)
			})
			t.Run("AssessmentResults", func(t *testing.T) {
				checkLayer(t, "AssessmentResults", layers.AssessmentResults, schemavalidate.KindAssessmentResults)
			})
			t.Run("POAM", func(t *testing.T) { checkLayer(t, "POAM", layers.POAM, schemavalidate.KindPOAM) })
		})
	}
}

func TestProjectAllLayers_DeterministicTimestampsAndIdempotency(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	for _, fwID := range []string{"cybersecure-canada", "nist-csf-2.0", "e-its-2024"} {
		fw, ok := reg.Get(fwID)
		if !ok {
			t.Fatalf("framework %s not found", fwID)
		}
		first, err := frameworks.ProjectAllLayers(fw)
		if err != nil {
			t.Fatalf("first ProjectAllLayers(%s): %v", fwID, err)
		}
		second, err := frameworks.ProjectAllLayers(fw)
		if err != nil {
			t.Fatalf("second ProjectAllLayers(%s): %v", fwID, err)
		}

		pairs := []struct {
			name string
			a, b []byte
		}{
			{"Catalog", first.Catalog, second.Catalog},
			{"Profile", first.Profile, second.Profile},
			{"ComponentDefinition", first.ComponentDefinition, second.ComponentDefinition},
			{"SystemSecurityPlan", first.SystemSecurityPlan, second.SystemSecurityPlan},
			{"AssessmentPlan", first.AssessmentPlan, second.AssessmentPlan},
			{"AssessmentResults", first.AssessmentResults, second.AssessmentResults},
			{"POAM", first.POAM, second.POAM},
		}
		for _, p := range pairs {
			if string(p.a) != string(p.b) {
				t.Errorf("%s/%s is not byte-for-byte deterministic across runs", fwID, p.name)
			}
		}
	}
}
