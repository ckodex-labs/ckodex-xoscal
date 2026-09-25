package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"gopkg.in/yaml.v3"
)

// WorkspaceConfig represents the .xoscal/xoscal.yaml configuration file.
type WorkspaceConfig struct {
	Version   string   `yaml:"version"`
	Name      string   `yaml:"name"`
	Framework string   `yaml:"framework"`
	Artifacts []string `yaml:"artifacts"`
}

// GenerateComponentDefinition builds a protobuf ComponentDefinition from scan results.
func GenerateComponentDefinition(scan *ProjectScan, framework string) *componentv1.ComponentDefinition {
	if framework == "" {
		framework = "nist-sp-800-53-rev5"
	}

	var components []*componentv1.DefinedComponent
	for _, c := range scan.Components {
		var implReqs []*componentv1.ImplementedRequirement
		for _, ctrl := range c.Controls {
			implReqs = append(implReqs, &componentv1.ImplementedRequirement{
				Uuid:        UUIDFromURN(fmt.Sprintf("urn:xoscal:implreq:%s:%s", c.Name, ctrl.ControlID)),
				ControlId:   ctrl.ControlID,
				Description: ctrl.Description,
			})
		}

		ctrlImpl := &componentv1.ControlImplementation{
			Uuid:                    UUIDFromURN(fmt.Sprintf("urn:xoscal:ctrlimpl:%s", c.Name)),
			Source:                  &commonv1.URIReference{Value: "https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final"},
			Description:             fmt.Sprintf("Baseline control implementations for %s", c.Title),
			ImplementedRequirements: implReqs,
		}

		comp := &componentv1.DefinedComponent{
			Uuid:                   UUIDFromURN(fmt.Sprintf("urn:xoscal:comp:%s", c.Name)),
			Type:                   c.Type,
			Title:                  &commonv1.MarkupLine{Value: c.Title},
			Description:            &commonv1.MarkupMultiline{Value: c.Description},
			ControlImplementations: []*componentv1.ControlImplementation{ctrlImpl},
		}
		components = append(components, comp)
	}

	return &componentv1.ComponentDefinition{
		Uuid: UUIDFromURN(fmt.Sprintf("urn:xoscal:compdef:%s", scan.Name)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Component Definition", scan.Name),
			Version: scan.Version,
		},
		BackMatter: buildScaffoldBackMatter(framework),
		Components: components,
	}
}

func buildScaffoldBackMatter(framework string) *commonv1.BackMatter {
	title := fmt.Sprintf("%s Reference", framework)
	desc := fmt.Sprintf("Authoritative framework standard for %s", framework)
	href := "https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final"
	propType := "standard"
	version := "rev5"

	switch strings.ToLower(framework) {
	case "cccs-itsg-33", "itsg-33":
		title = "CCCS ITSG-33 IT Security Risk Management: A Lifecycle Approach - Annex 3"
		desc = "Authoritative Government of Canada security control catalogue published by CCCS / CSE"
		href = "https://www.cyber.gc.ca/en/guidance/it-security-risk-management-lifecycle-approach-itsg-33"
		propType = "standard"
		version = "Annex 3"
	case "cccs-medium-cloud-pbmm", "pbmm":
		title = "Government of Canada Cloud Security Control Profile: Protected B / Medium Integrity / Medium Availability (PBMM)"
		desc = "Mandatory baseline security control profile for CSPs and GC cloud services per ITSP.50.103 and ITSG-33 Annex 4A Profile 1"
		href = "https://www.cyber.gc.ca/en/guidance/government-canada-cloud-security-control-profile-protect-b-medium-integrity-medium-availability"
		propType = "profile"
		version = "ITSP.50.103"
	case "cybersecure-canada", "csc":
		title = "CyberSecure Canada - Baseline Cyber Security Controls for Small and Medium Organizations"
		desc = "National cyber security certification standard developed by ISED and CCCS"
		href = "https://ised-isde.canada.ca/site/cybersecure-canada/en"
		propType = "standard"
		version = "1.2"
	case "cccs-itsp-10-171", "itsp-10-171":
		title = "CCCS ITSP.10.171 - Protecting Specified Information in Non-Government of Canada Systems and Organizations"
		desc = "Canadian cybersecurity baseline for defense suppliers and commercial organizations handling Controlled Goods"
		href = "https://www.cyber.gc.ca/en/guidance/protecting-specified-information-non-government-canada-systems-and-organizations-itsp10171"
		propType = "standard"
		version = "1.0"
	case "iso-27001", "iso27001":
		title = "ISO/IEC 27001:2022 Information Security Management Systems"
		desc = "International standard for information security management systems"
		href = "https://www.iso.org/standard/27001"
		propType = "standard"
		version = "2022"
	}

	return &commonv1.BackMatter{
		Resources: []*commonv1.Resource{{
			Uuid:  UUIDFromURN("urn:xoscal:resource:" + framework),
			Title: title,
			Description: &commonv1.MarkupMultiline{
				Value: desc,
			},
			Props: []*commonv1.Property{
				{Name: "type", Value: propType},
				{Name: "version", Value: version},
			},
			Rlinks: []*commonv1.Rlink{{
				Href: &commonv1.URI{Value: href},
			}},
		}},
	}
}

// ScaffoldWorkspace initializes the .xoscal workspace and writes component-definition.json.
// It verifies that the generated JSON passes OSCAL 1.2.3 schema validation before writing to disk.
func ScaffoldWorkspace(dir string, framework string) (*ProjectScan, []byte, error) {
	scan, err := ScanRepository(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("scan repository: %w", err)
	}

	compDef := GenerateComponentDefinition(scan, framework)
	jsonBytes, err := oscal.ExportComponentDefinitionJSON(compDef)
	if err != nil {
		return nil, nil, fmt.Errorf("export component definition JSON: %w", err)
	}

	// Immediate Tier 1 Schema Preflight
	validator, err := schemavalidate.NewValidator()
	if err != nil {
		return nil, nil, fmt.Errorf("init schema validator: %w", err)
	}
	if err := validator.Validate(jsonBytes, schemavalidate.KindComponentDefinition); err != nil {
		return nil, nil, fmt.Errorf("scaffolded component definition failed schema validation: %w", err)
	}

	// Create .xoscal directory
	xoscalDir := filepath.Join(dir, ".xoscal")
	if err := os.MkdirAll(xoscalDir, 0750); err != nil {
		return nil, nil, fmt.Errorf("create .xoscal dir: %w", err)
	}

	// Write workspace config
	cfg := WorkspaceConfig{
		Version:   "1.0.0",
		Name:      scan.Name,
		Framework: framework,
		Artifacts: []string{"component-definition.json"},
	}
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal workspace config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(xoscalDir, "xoscal.yaml"), cfgData, 0600); err != nil {
		return nil, nil, fmt.Errorf("write xoscal.yaml: %w", err)
	}

	// Write component-definition.json
	compDefPath := filepath.Join(dir, "component-definition.json")
	if err := os.WriteFile(compDefPath, jsonBytes, 0600); err != nil {
		return nil, nil, fmt.Errorf("write component-definition.json: %w", err)
	}

	return scan, jsonBytes, nil
}
