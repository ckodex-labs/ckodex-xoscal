package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
)

// ParamDef represents an Organization-Defined Parameter (ODP) extracted from a requirement.
type ParamDef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Requirement represents a parsed regulatory requirement.
type Requirement struct {
	ID                   string     `json:"id"`
	Citation             string     `json:"citation"`
	Title                string     `json:"title"`
	Text                 string     `json:"text"`
	Guidance             string     `json:"guidance,omitempty"`
	Params               []ParamDef `json:"params,omitempty"`
	Role                 string     `json:"role"`
	RiskLevel            string     `json:"risk_level"`
	Lifecycle            string     `json:"lifecycle"`
	Framework            string     `json:"framework"`
	Section              string     `json:"section"`
	Subsection           string     `json:"subsection"`
	Depth                int        `json:"depth"`
	ParentURN            string     `json:"parent_urn,omitempty"`
	Assessable           bool       `json:"assessable"`
	ImplementationGroups []string   `json:"implementation_groups,omitempty"`
	NodeRefID            string     `json:"ref_id,omitempty"`
	NodeName             string     `json:"node_name,omitempty"`
}

// Parser converts raw regulatory text into structured Requirements.
type Parser interface {
	Parse(ctx context.Context, raw []byte) ([]Requirement, error)
}

// EUAIActParser reads structured JSON representing EU AI Act requirements.
type EUAIActParser struct{}

type rawRequirementJSON struct {
	Requirement
	Description string `json:"description"`
	Article     string `json:"article"`
	Category    string `json:"category"`
	Assessable  *bool  `json:"assessable"`
}

func (r rawRequirementJSON) toRequirement() Requirement {
	req := r.Requirement
	if req.Text == "" && r.Description != "" {
		req.Text = r.Description
	}
	if req.Citation == "" && r.Article != "" {
		req.Citation = r.Article
	}
	if req.Citation == "" && req.ID != "" {
		req.Citation = req.ID
	}
	if req.Section == "" && r.Category != "" {
		req.Section = r.Category
	}
	if r.Assessable != nil {
		req.Assessable = *r.Assessable
	} else {
		req.Assessable = true
	}
	return req
}

func (p *EUAIActParser) Parse(ctx context.Context, raw []byte) ([]Requirement, error) {
	var rawReqs []rawRequirementJSON
	if err := json.Unmarshal(raw, &rawReqs); err != nil {
		// Try single object
		var single rawRequirementJSON
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("unmarshal requirements: %w (also tried single: %v)", err, err2)
		}
		rawReqs = append(rawReqs, single)
	}
	reqs := make([]Requirement, 0, len(rawReqs))
	for _, r := range rawReqs {
		reqs = append(reqs, r.toRequirement())
	}
	return reqs, nil
}
