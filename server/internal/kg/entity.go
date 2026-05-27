package kg

import (
	"encoding/json"
	"time"
)

// EntityStatus represents the lifecycle state of a KG entity.
type EntityStatus string

const (
	EntityStatusActive     EntityStatus = "active"
	EntityStatusSuperseded EntityStatus = "superseded"
	EntityStatusRecalled   EntityStatus = "recalled"
)

// Entity is the base type for all JSON-LD knowledge graph nodes.
type Entity struct {
	URN       string          `json:"@id"`
	Type      string          `json:"@type"`
	Version   int             `json:"version"`
	Status    EntityStatus    `json:"status"`
	ValidFrom time.Time       `json:"valid_from"`
	ValidTo   *time.Time      `json:"valid_to,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

// Requirement represents a regulatory requirement.
type Requirement struct {
	URN                  string   `json:"@id"`
	Type                 string   `json:"@type"`
	Framework            string   `json:"framework"`
	Citation             string   `json:"citation"`
	Role                 string   `json:"role"`
	RiskLevel            string   `json:"risk_level"`
	Lifecycle            string   `json:"lifecycle"`
	Text                 string   `json:"text"`
	Title                string   `json:"title"`
	Section              string   `json:"section"`
	Subsection           string   `json:"subsection"`
	Depth                int      `json:"depth"`
	ParentURN            string   `json:"parent_urn,omitempty"`
	Assessable           bool     `json:"assessable"`
	ImplementationGroups []string `json:"implementation_groups,omitempty"`
	NodeRefID            string   `json:"ref_id,omitempty"`
	NodeName             string   `json:"node_name,omitempty"`
}

// Control represents a security/control definition.
type Control struct {
	URN         string   `json:"@id"`
	Type        string   `json:"@type"`
	Framework   string   `json:"framework"`
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters,omitempty"`
}

// Mapping represents a relationship between requirements and controls.
type Mapping struct {
	URN          string  `json:"@id"`
	Type         string  `json:"@type"`
	From         string  `json:"from"`
	To           string  `json:"to"`
	Relationship string  `json:"relationship"`
	Confidence   float64 `json:"confidence"`
}

// Framework represents a cybersecurity framework metadata.
type Framework struct {
	URN             string `json:"@id"`
	Type            string `json:"@type"`
	RefID           string `json:"ref_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Provider        string `json:"provider"`
	Version         string `json:"version"`
	PublicationDate string `json:"publication_date"`
	Locale          string `json:"locale"`
	Packager        string `json:"packager"`
}

// ImplementationGroup represents an implementation group (maturity level, tier, etc.).
type ImplementationGroup struct {
	URN         string `json:"@id"`
	Type        string `json:"@type"`
	Framework   string `json:"framework"`
	RefID       string `json:"ref_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RequirementMapping represents a cross-framework mapping between requirements.
type RequirementMapping struct {
	URN          string  `json:"@id"`
	Type         string  `json:"@type"`
	SourceURN    string  `json:"source_urn"`
	TargetURN    string  `json:"target_urn"`
	Relationship string  `json:"relationship"` // equal, subset, superset, intersect
	Rationale    string  `json:"rationale"`
	Strength     float64 `json:"strength"`
}

// Snapshot represents a frozen view of the KG at a point in time.
type Snapshot struct {
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	EntityCount int       `json:"entity_count"`
}

// Release bundles a snapshot into a named versioned artifact.
type Release struct {
	Name      string    `json:"name"`
	Snapshot  string    `json:"snapshot"`
	CreatedAt time.Time `json:"created_at"`
}
