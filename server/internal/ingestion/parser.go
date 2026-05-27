package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
)

// Requirement represents a parsed regulatory requirement.
type Requirement struct {
	ID                   string   `json:"id"`
	Citation             string   `json:"citation"`
	Title                string   `json:"title"`
	Text                 string   `json:"text"`
	Role                 string   `json:"role"`
	RiskLevel            string   `json:"risk_level"`
	Lifecycle            string   `json:"lifecycle"`
	Framework            string   `json:"framework"`
	Section              string   `json:"section"`
	Subsection           string   `json:"subsection"`
	Depth                int      `json:"depth"`
	ParentURN            string   `json:"parent_urn,omitempty"`
	Assessable           bool     `json:"assessable"`
	ImplementationGroups []string `json:"implementation_groups,omitempty"`
	NodeRefID            string   `json:"ref_id,omitempty"`
	NodeName             string   `json:"node_name,omitempty"`
}

// Parser converts raw regulatory text into structured Requirements.
type Parser interface {
	Parse(ctx context.Context, raw []byte) ([]Requirement, error)
}

// EUAIActParser reads structured JSON representing EU AI Act requirements.
type EUAIActParser struct{}

func (p *EUAIActParser) Parse(ctx context.Context, raw []byte) ([]Requirement, error) {
	var reqs []Requirement
	if err := json.Unmarshal(raw, &reqs); err != nil {
		// Try single object
		var single Requirement
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("unmarshal requirements: %w (also tried single: %v)", err, err2)
		}
		reqs = append(reqs, single)
	}
	return reqs, nil
}
