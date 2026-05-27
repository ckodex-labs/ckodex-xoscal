package ingestion

import (
	"encoding/json"
	"fmt"

	"github.com/mchorfa/xoscal/server/internal/kg"
)

// Normalizer converts parsed Requirements into KG entities.
type Normalizer struct {
	Framework string
}

func NewNormalizer(framework string) *Normalizer {
	return &Normalizer{Framework: framework}
}

func (n *Normalizer) Normalize(reqs []Requirement) ([]*kg.Entity, error) {
	var entities []*kg.Entity
	for _, req := range reqs {
		payload := kg.Requirement{
			URN:                  n.toURN(req),
			Type:                 "reg:Requirement",
			Framework:            n.Framework,
			Citation:             req.Citation,
			Role:                 req.Role,
			RiskLevel:            req.RiskLevel,
			Lifecycle:            req.Lifecycle,
			Text:                 req.Text,
			Title:                req.Title,
			Section:              req.Section,
			Subsection:           req.Subsection,
			Depth:                req.Depth,
			ParentURN:            req.ParentURN,
			Assessable:           req.Assessable,
			ImplementationGroups: req.ImplementationGroups,
			NodeRefID:            req.NodeRefID,
			NodeName:             req.NodeName,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal requirement %s: %w", req.ID, err)
		}
		entities = append(entities, &kg.Entity{
			URN:     payload.URN,
			Type:    payload.Type,
			Payload: raw,
		})
	}
	return entities, nil
}

func (n *Normalizer) toURN(req Requirement) string {
	if req.NodeRefID != "" {
		return fmt.Sprintf("urn:%s:req:%s", n.Framework, req.NodeRefID)
	}
	if req.ID != "" {
		return fmt.Sprintf("urn:%s:req:%s", n.Framework, req.ID)
	}
	return fmt.Sprintf("urn:%s:req:%s:%s", n.Framework, req.Section, req.Subsection)
}
