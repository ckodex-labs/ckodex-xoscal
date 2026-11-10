package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ExportJSON serializes a protobuf message to canonical JSON.
// Deprecated: Use the model-specific Export*JSON functions instead, which
// produce OSCAL-compliant JSON with proper kebab-case, unwrapped values,
// and root model wrappers.
func ExportJSON(msg proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{Multiline: true}.Marshal(msg)
}

// ExportCatalogJSON serializes a catalog to OSCAL-compliant JSON.
func ExportCatalogJSON(c *catalogv1.Catalog) ([]byte, error) {
	return marshalOSCALJSON(c, "catalog")
}

// ExportProfileJSON serializes a profile to OSCAL-compliant JSON.
func ExportProfileJSON(p *profilev1.Profile) ([]byte, error) {
	return marshalOSCALJSON(p, "profile")
}

// ExportSSPJSON serializes an SSP to OSCAL-compliant JSON.
func ExportSSPJSON(s *sspv1.SystemSecurityPlan) ([]byte, error) {
	return marshalOSCALJSON(s, "system-security-plan")
}

// ExportComponentDefinitionJSON serializes a component definition to OSCAL-compliant JSON.
func ExportComponentDefinitionJSON(c *componentv1.ComponentDefinition) ([]byte, error) {
	return marshalOSCALJSON(c, "component-definition")
}

// ExportAssessmentPlanJSON serializes an assessment plan to OSCAL-compliant JSON.
func ExportAssessmentPlanJSON(a *assessment_planv1.AssessmentPlan) ([]byte, error) {
	return marshalOSCALJSON(a, "assessment-plan")
}

// ExportAssessmentResultsJSON serializes assessment results to OSCAL-compliant JSON.
func ExportAssessmentResultsJSON(a *assessment_resultsv1.AssessmentResults) ([]byte, error) {
	return marshalOSCALJSON(a, "assessment-results")
}

// ExportPOAMJSON serializes a plan of action and milestones to OSCAL-compliant JSON.
func ExportPOAMJSON(p *poamv1.PlanOfActionAndMilestones) ([]byte, error) {
	if p != nil && len(p.Risks) > 0 { //nolint:staticcheck // SA1019: strip deprecated compatibility field before protojson marshal
		shallow := &poamv1.PlanOfActionAndMilestones{
			Uuid:             p.Uuid,
			Metadata:         p.Metadata,
			ImportSsp:        p.ImportSsp,
			LocalDefinitions: p.LocalDefinitions,
			BackMatter:       p.BackMatter,
			PoamItems:        p.PoamItems,
		}
		return marshalOSCALJSON(shallow, "plan-of-action-and-milestones")
	}
	return marshalOSCALJSON(p, "plan-of-action-and-milestones")
}

// ExportMappingCollectionJSON serializes an OSCAL 1.2 mapping collection.
func ExportMappingCollectionJSON(m *mappingv1.MappingCollection) ([]byte, error) {
	return marshalOSCALJSON(m, "mapping-collection")
}

// ExportMappingsJSON serializes mappings to a JSON wrapper.
// Deprecated: this is the pre-release mapping prototype shape. Use
// ExportMappingCollectionJSON for an OSCAL-conformant document.
func ExportMappingsJSON(maps []*mappingv1.Map) ([]byte, error) {
	wrapper := struct {
		Maps []*mappingv1.Map `json:"maps"`
	}{Maps: maps}
	return json.MarshalIndent(wrapper, "", "  ")
}

// ArtifactType identifies the kind of OSCAL artifact.
type ArtifactType string

const (
	ArtifactCatalog             ArtifactType = "catalog"
	ArtifactProfile             ArtifactType = "profile"
	ArtifactSSP                 ArtifactType = "ssp"
	ArtifactComponentDefinition ArtifactType = "component-definition"
	ArtifactAssessmentPlan      ArtifactType = "assessment-plan"
	ArtifactAssessmentResults   ArtifactType = "assessment-results"
	ArtifactPOAM                ArtifactType = "poam"
	ArtifactMapping             ArtifactType = "mapping"
	ArtifactMappings            ArtifactType = "mappings"
)

// GenerateAll produces all supported artifact types from a snapshot.
type GenerateAllResult struct {
	Catalog             *catalogv1.Catalog
	Profile             *profilev1.Profile
	SSP                 *sspv1.SystemSecurityPlan
	ComponentDefinition *componentv1.ComponentDefinition
	AssessmentPlan      *assessment_planv1.AssessmentPlan
	AssessmentResults   *assessment_resultsv1.AssessmentResults
	POAM                *poamv1.PlanOfActionAndMilestones
	Mappings            []*mappingv1.Map
}

// GenerateAllArtifacts generates every supported artifact from a snapshot
// using a single snapshot read and single-pass entity decode.
func (g *Generator) GenerateAllArtifacts(ctx context.Context, snapshotName, framework string, selected []string) (*GenerateAllResult, error) {
	res := &GenerateAllResult{}
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, true)
	if err != nil {
		return res, fmt.Errorf("generate all: %w", err)
	}

	res.Catalog = g.generateCatalogFromSnapshot(snap, snapshotName, framework)
	res.Profile = g.generateProfileFromSnapshot(snap, snapshotName, framework, selected)
	res.SSP = g.generateSSPFromSnapshot(snap, snapshotName, framework)
	res.ComponentDefinition = g.generateComponentDefinitionFromSnapshot(snap, snapshotName, framework)
	res.AssessmentPlan = g.generateAssessmentPlanFromSnapshot(snap, snapshotName, framework)
	res.AssessmentResults = g.generateAssessmentResultsFromSnapshot(snap, snapshotName, framework)
	res.POAM = g.generatePOAMFromSnapshot(snap, snapshotName, framework)
	res.Mappings = g.generateMappingsFromSnapshot(snap)
	return res, nil
}
