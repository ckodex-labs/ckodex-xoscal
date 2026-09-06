package oscal

import (
	"fmt"
	"strings"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func extractGuidance(text string) string {
	if text == "" {
		return ""
	}
	// Simple heuristic: look for sentences starting with should/may/can.
	// A real implementation would use NLP; this is a lightweight data-driven fallback.
	var guidance []string
	// Naive sentence split on ". "
	sentences := strings.Split(text, ". ")
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "should ") || strings.HasPrefix(lower, "may ") || strings.HasPrefix(lower, "can ") || strings.HasPrefix(lower, "it is recommended") {
			guidance = append(guidance, trimmed)
		}
	}
	return strings.Join(guidance, ". ")
}

// requirementToProps extracts data-driven OSCAL properties from a requirement.
// Custom prop names use a namespace to avoid violating Metaschema allowed-values
// constraints on control/prop @name (which restricts to alt-identifier, label,
// marking, sort-id, status in the default namespace).
const xoscalPropNamespace = "https://ckodex.io/ns/xoscal/props"

func requirementToProps(req kg.Requirement) []*commonv1.Property {
	var props []*commonv1.Property
	if req.Role != "" {
		props = append(props, &commonv1.Property{Name: "role", Value: req.Role, Ns: xoscalPropNamespace})
	}
	if req.RiskLevel != "" {
		props = append(props, &commonv1.Property{Name: "risk-level", Value: req.RiskLevel, Ns: xoscalPropNamespace})
	}
	if req.Lifecycle != "" {
		props = append(props, &commonv1.Property{Name: "lifecycle", Value: req.Lifecycle, Ns: xoscalPropNamespace})
	}
	if len(req.ImplementationGroups) > 0 {
		props = append(props, &commonv1.Property{
			Name:  "implementation-groups",
			Value: fmt.Sprintf("%v", req.ImplementationGroups),
			Ns:    xoscalPropNamespace,
		})
	}
	return props
}

// buildBackMatter creates a BackMatter with a resource linking to the framework.
// The resource includes an rlink (required by the Metaschema cardinality constraint
// for rlink|base64) and uses only prop names from the allowed-values set
// (marking, published, type, version) per the pinned OSCAL Metaschema constraints.
const frameworkSourceBaseURL = "https://raw.githubusercontent.com/intuitem/ciso-assistant-community/main/backend/library/libraries"

func buildBackMatter(framework, snapshotName string) *commonv1.BackMatter {
	return &commonv1.BackMatter{
		Resources: []*commonv1.Resource{{
			Uuid:  newUUID(uuidNamespace, fmt.Sprintf("resource-%s-%s", framework, snapshotName)),
			Title: fmt.Sprintf("%s Framework Reference", framework),
			Description: &commonv1.MarkupMultiline{
				Value: fmt.Sprintf("Source framework metadata for %s snapshot %s", framework, snapshotName),
			},
			Props: []*commonv1.Property{
				{Name: "type", Value: "standard"},
				{Name: "version", Value: snapshotName},
			},
			Rlinks: []*commonv1.Rlink{{
				Href: &commonv1.URI{Value: fmt.Sprintf("%s/%s.yaml", frameworkSourceBaseURL, framework)},
			}},
		}},
	}
}

// GenerateCrossFrameworkMappings builds OSCAL Maps from cross-framework requirement mappings.
