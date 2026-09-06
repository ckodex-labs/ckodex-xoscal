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

	var maps []*mappingv1.Map
	for _, e := range entities {
		if e.Type != "reg:Mapping" {
			continue
		}
		var m kg.Mapping
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			continue
		}
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
	return maps, nil
}

// GenerateAssessmentPlan builds an OSCAL Assessment Plan from a snapshot.
