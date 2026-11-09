package ingestion

import (
	"encoding/json"
	"fmt"
	"strings"

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
		var kgParams []kg.ParamDef
		for _, p := range req.Params {
			kgParams = append(kgParams, kg.ParamDef{
				ID:    p.ID,
				Label: p.Label,
			})
		}
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
			Guidance:             req.Guidance,
			Params:               kgParams,
			Section:              req.Section,
			Subsection:           req.Subsection,
			Depth:                req.Depth,
			ParentURN:            n.toParentURN(req.ParentURN),
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
	if req.ID != "" {
		return fmt.Sprintf("urn:%s:req:%s", n.Framework, req.ID)
	}
	if req.NodeRefID != "" {
		return fmt.Sprintf("urn:%s:req:%s", n.Framework, req.NodeRefID)
	}
	return fmt.Sprintf("urn:%s:req:%s:%s", n.Framework, req.Section, req.Subsection)
}

func (n *Normalizer) toParentURN(parentURN string) string {
	if parentURN == "" {
		return ""
	}
	prefix := fmt.Sprintf("urn:%s:req:", n.Framework)
	if strings.HasPrefix(parentURN, prefix) {
		return parentURN
	}
	norm := normalizeURN(parentURN)
	if norm == "" {
		return ""
	}
	return prefix + norm
}
