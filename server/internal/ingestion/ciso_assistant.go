package ingestion

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
)

// CISOLibrary represents the top-level structure of a CISO Assistant library YAML file.
type CISOLibrary struct {
	URN             string      `yaml:"urn"`
	Locale          string      `yaml:"locale"`
	RefID           string      `yaml:"ref_id"`
	Name            string      `yaml:"name"`
	Description     string      `yaml:"description"`
	Copyright       string      `yaml:"copyright"`
	Version         string      `yaml:"version"`
	PublicationDate string      `yaml:"publication_date"`
	Provider        string      `yaml:"provider"`
	Packager        string      `yaml:"packager"`
	Objects         CISOObjects `yaml:"objects"`
}

// CISOObjects holds the framework and optionally mapping sets inside a library.
type CISOObjects struct {
	Framework CISOLibraryFramework `yaml:"framework"`
}

// CISOLibraryFramework represents the framework object within a CISO library.
type CISOLibraryFramework struct {
	URN                     string                       `yaml:"urn"`
	RefID                   string                       `yaml:"ref_id"`
	Name                    string                       `yaml:"name"`
	Description             string                       `yaml:"description"`
	ImplementationGroupsDef []CISOImplementationGroupDef `yaml:"implementation_groups_definition"`
	RequirementNodes        []CISORequirementNode        `yaml:"requirement_nodes"`
}

// CISOImplementationGroupDef represents an implementation group definition.
type CISOImplementationGroupDef struct {
	RefID       string `yaml:"ref_id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// CISORequirementNode represents a single requirement node in the hierarchy.
type CISORequirementNode struct {
	URN                  string   `yaml:"urn"`
	Assessable           bool     `yaml:"assessable"`
	Depth                int      `yaml:"depth"`
	ParentURN            string   `yaml:"parent_urn"`
	RefID                string   `yaml:"ref_id"`
	Name                 string   `yaml:"name"`
	Description          string   `yaml:"description"`
	ImplementationGroups []string `yaml:"implementation_groups"`
}

// CISOAssistantParser reads CISO Assistant library YAML files.
type CISOAssistantParser struct{}

// Parse converts raw CISO Assistant YAML into structured Requirements.
func (p *CISOAssistantParser) Parse(ctx context.Context, raw []byte) ([]Requirement, error) {
	var lib CISOLibrary
	if err := yaml.Unmarshal(raw, &lib); err != nil {
		return nil, fmt.Errorf("unmarshal ciso library: %w", err)
	}

	frameworkRefID := lib.RefID
	if frameworkRefID == "" {
		frameworkRefID = lib.Objects.Framework.RefID
	}

	var reqs []Requirement
	for _, node := range lib.Objects.Framework.RequirementNodes {
		req := Requirement{
			ID:                   normalizeURN(node.URN),
			Citation:             node.RefID,
			Title:                node.Name,
			Text:                 node.Description,
			Framework:            frameworkRefID,
			Depth:                node.Depth,
			ParentURN:            node.ParentURN,
			Assessable:           node.Assessable,
			ImplementationGroups: node.ImplementationGroups,
			NodeRefID:            node.RefID,
			NodeName:             node.Name,
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// normalizeURN strips the CISO-specific prefix to produce a stable ID.
func normalizeURN(urn string) string {
	// CISO URNs look like: urn:intuitem:risk:req_node:<framework>:<id>
	// We return the last segment after the framework prefix.
	if urn == "" {
		return ""
	}
	// Simple heuristic: find the last colon and take everything after.
	// For nested URNs this may need more sophistication.
	for i := len(urn) - 1; i >= 0; i-- {
		if urn[i] == ':' {
			return urn[i+1:]
		}
	}
	return urn
}
