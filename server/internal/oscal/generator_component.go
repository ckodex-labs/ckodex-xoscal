package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateComponentDefinition(ctx context.Context, snapshotName string, framework string) (*componentv1.ComponentDefinition, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var components []*componentv1.DefinedComponent
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			continue
		}
		if req.Framework != framework {
			continue
		}
		var controlImplementations []*componentv1.ControlImplementation
		if req.Citation != "" {
			controlImplementations = append(controlImplementations, &componentv1.ControlImplementation{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("ctrlimpl-%s", req.Citation)),
				Source:      &commonv1.URIReference{Value: fmt.Sprintf("catalog-%s-%s", framework, snapshotName)},
				Description: req.Text,
				ImplementedRequirements: []*componentv1.ImplementedRequirement{{
					Uuid:        newUUID(uuidNamespace, fmt.Sprintf("implreq-%s", req.Citation)),
					ControlId:   sanitizeToken(req.Citation),
					Description: req.Text,
				}},
			})
		}
		components = append(components, &componentv1.DefinedComponent{
			Uuid:                   newUUID(uuidNamespace, fmt.Sprintf("comp-%s", req.Citation)),
			Type:                   "software",
			Title:                  &commonv1.MarkupLine{Value: req.Title},
			Description:            &commonv1.MarkupMultiline{Value: req.Text},
			ControlImplementations: controlImplementations,
		})
	}

	return &componentv1.ComponentDefinition{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("compdef-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Component Definition", framework),
			Version: snapshotName,
		},
		BackMatter: buildBackMatter(framework, snapshotName),
		Components: components,
	}, nil
}

// GenerateMappings builds OSCAL Control Mapping entries from a snapshot.
