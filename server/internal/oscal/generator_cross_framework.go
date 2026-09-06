package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateCrossFrameworkMappings(ctx context.Context, snapshotName, sourceFramework, targetFramework string) ([]*mappingv1.Map, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	// Build URN -> Requirement lookup for resolving mapping references.
	reqByURN := make(map[string]kg.Requirement)
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var r kg.Requirement
		if err := json.Unmarshal(e.Payload, &r); err != nil {
			continue
		}
		reqByURN[e.URN] = r
	}

	var maps []*mappingv1.Map
	for _, e := range entities {
		if e.Type != "reg:RequirementMapping" {
			continue
		}
		var m kg.RequirementMapping
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			continue
		}
		// Filter by source/target framework when specified.
		srcReq, srcOk := reqByURN[m.SourceURN]
		tgtReq, tgtOk := reqByURN[m.TargetURN]
		if sourceFramework != "" && (!srcOk || srcReq.Framework != sourceFramework) {
			continue
		}
		if targetFramework != "" && (!tgtOk || tgtReq.Framework != targetFramework) {
			continue
		}

		// Resolve IdRef from framework:citation (fall back to raw URN if lookup fails).
		srcIdRef := m.SourceURN
		if srcOk && srcReq.Citation != "" {
			srcIdRef = fmt.Sprintf("%s:%s", srcReq.Framework, srcReq.Citation)
		}
		tgtIdRef := m.TargetURN
		if tgtOk && tgtReq.Citation != "" {
			tgtIdRef = fmt.Sprintf("%s:%s", tgtReq.Framework, tgtReq.Citation)
		}

		confidence := m.Strength
		if confidence == 0 {
			confidence = 0.5
		}

		maps = append(maps, &mappingv1.Map{
			Uuid:              &commonv1.UUID{Value: m.URN},
			MatchingRationale: m.Rationale,
			Relationship:      &commonv1.Token{Value: m.Relationship},
			Sources: []*mappingv1.MappingItem{{
				IdRef: srcIdRef,
			}},
			Targets: []*mappingv1.MappingItem{{
				IdRef: tgtIdRef,
			}},
			ConfidenceScore: &mappingv1.ConfidenceScore{Value: confidence},
		})
	}
	return maps, nil
}
