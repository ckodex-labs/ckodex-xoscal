package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateMappings(ctx context.Context, snapshotName string) ([]*mappingv1.Map, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var raw []kg.Mapping
	for _, e := range entities {
		if e == nil || e.Type != "reg:Mapping" {
			continue
		}
		var m kg.Mapping
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			continue
		}
		raw = append(raw, m)
	}
	return buildMappingProtos(raw), nil
}

func (g *Generator) generateMappingsFromSnapshot(snap *parsedSnapshot) []*mappingv1.Map {
	return buildMappingProtos(snap.mappings)
}

func buildMappingProtos(mappings []kg.Mapping) []*mappingv1.Map {
	maps := make([]*mappingv1.Map, 0, len(mappings))
	for _, m := range mappings {
		maps = append(maps, &mappingv1.Map{
			Uuid:         &commonv1.UUID{Value: m.URN},
			Relationship: &commonv1.Token{Value: m.Relationship},
			Sources: []*mappingv1.MappingItem{{
				IdRef: m.From,
			}},
			Targets: []*mappingv1.MappingItem{{
				IdRef: m.To,
			}},
		})
	}
	return maps
}

// GenerateAssessmentPlan builds an OSCAL Assessment Plan from a snapshot.
