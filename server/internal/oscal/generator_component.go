package oscal

import (
	"context"
	"fmt"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
)

func (g *Generator) GenerateComponentDefinition(ctx context.Context, snapshotName string, framework string) (*componentv1.ComponentDefinition, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generateComponentDefinitionFromSnapshot(snap, snapshotName, framework), nil
}

func (g *Generator) generateComponentDefinitionFromSnapshot(snap *parsedSnapshot, snapshotName string, framework string) *componentv1.ComponentDefinition {
	var components []*componentv1.DefinedComponent
	if len(snap.assessableReqs) > 0 {
		implReqs := make([]*componentv1.ImplementedRequirement, 0, len(snap.assessableReqs))
		for _, req := range snap.assessableReqs {
			ctrlID := sanitizeToken(req.Citation)
			if ctrlID == "" {
				ctrlID = sanitizeToken(req.NodeRefID)
			}
			if ctrlID == "" {
				continue
			}
			title := singleLine(req.Title)
			if title == "" {
				title = singleLine(ctrlID)
			}
			implReqs = append(implReqs, &componentv1.ImplementedRequirement{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("implreq-%s", ctrlID)),
				ControlId:   ctrlID,
				Description: fmt.Sprintf("Baseline implementation of %s (%s).", ctrlID, title),
			})
		}
		if len(implReqs) > 0 {
			components = []*componentv1.DefinedComponent{{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("comp-%s-%s", framework, snapshotName)),
				Type:        "software",
				Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("%s Reference System Component", framework)},
				Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("Reference component implementing %s controls.", framework)},
				ControlImplementations: []*componentv1.ControlImplementation{{
					Uuid:                    newUUID(uuidNamespace, fmt.Sprintf("ctrlimpl-%s-%s", framework, snapshotName)),
					Source:                  &commonv1.URIReference{Value: fmt.Sprintf("catalog-%s-%s", framework, snapshotName)},
					Description:             fmt.Sprintf("Control implementations for %s.", framework),
					ImplementedRequirements: implReqs,
				}},
			}}
		}
	}

	return &componentv1.ComponentDefinition{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("compdef-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Component Definition", framework),
			Version: snapshotName,
		},
		BackMatter: buildBackMatter(framework, snapshotName),
		Components: components,
	}
}

// GenerateMappings builds OSCAL Control Mapping entries from a snapshot.
