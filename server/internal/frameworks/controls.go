package frameworks

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ControlInfo represents parsed, structured control details for terminal & UI display.
type ControlInfo struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Class            string            `json:"class,omitempty"`
	FamilyID         string            `json:"family_id"`
	FamilyTitle      string            `json:"family_title"`
	FrameworkID      string            `json:"framework_id"`
	Statements       []string          `json:"statements,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"`
	Guidance         string            `json:"guidance,omitempty"`
	CanadianGuidance string            `json:"canadian_guidance,omitempty"`
	IsCanadian       bool              `json:"is_canadian"`
	IsEnhancement    bool              `json:"is_enhancement"`
	ParentControlID  string            `json:"parent_control_id,omitempty"`
	Enhancements     []string          `json:"enhancements,omitempty"`
}

// LoadControls extracts all controls and enhancements from a framework's catalog.
func LoadControls(f Framework) ([]ControlInfo, error) {
	data, err := f.CatalogJSON()
	if err != nil {
		return nil, fmt.Errorf("load catalog for %s: %w", f.ID(), err)
	}

	var doc oscalCatalogDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json for %s: %w", f.ID(), err)
	}

	var result []ControlInfo

	var parseControl func(c oscalJSONControl, famID, famTitle, parentID string) ControlInfo
	parseControl = func(c oscalJSONControl, famID, famTitle, parentID string) ControlInfo {
		info := ControlInfo{
			ID:              strings.ToLower(c.ID),
			Title:           c.Title,
			Class:           c.Class,
			FamilyID:        strings.ToUpper(famID),
			FamilyTitle:     famTitle,
			FrameworkID:     f.ID(),
			Parameters:      make(map[string]string),
			ParentControlID: strings.ToLower(parentID),
			IsEnhancement:   parentID != "",
		}

		// Check props for Canadian sovereignty
		for _, prop := range c.Props {
			if prop.Name == "jurisdiction" && strings.ToLower(prop.Value) == "ca" {
				info.IsCanadian = true
			}
			if prop.Name == "sovereign-guidance" && strings.ToLower(prop.Value) == "true" {
				info.IsCanadian = true
			}
		}

		// Parameters
		for _, p := range c.Params {
			val := p.Label
			if len(p.Values) > 0 {
				val = strings.Join(p.Values, ", ")
			}
			info.Parameters[p.ID] = val
		}

		// Parts
		var stmts []string
		for _, p := range c.Parts {
			switch strings.ToLower(p.Name) {
			case "statement":
				if p.Prose != "" {
					stmts = append(stmts, p.Prose)
				}
				for _, sub := range p.Parts {
					if sub.Prose != "" {
						prefix := ""
						if sub.Title != "" {
							prefix = sub.Title + " "
						}
						stmts = append(stmts, prefix+sub.Prose)
					}
				}
			case "guidance":
				if p.Prose != "" {
					info.Guidance = p.Prose
				}
			case "gc-guidance", "canadian-guidance":
				if p.Prose != "" {
					info.CanadianGuidance = p.Prose
					info.IsCanadian = true
				}
			default:
				if p.Prose != "" && !strings.Contains(strings.ToLower(p.Name), "assessment") {
					stmts = append(stmts, p.Prose)
				}
			}
		}
		info.Statements = stmts

		// Sub-controls / Enhancements
		for _, sub := range c.Controls {
			enh := parseControl(sub, famID, famTitle, c.ID)
			result = append(result, enh)
			info.Enhancements = append(info.Enhancements, enh.ID)
		}

		return info
	}

	// Traverse Groups
	for _, g := range doc.Catalog.Groups {
		famID := g.ID
		famTitle := g.Title
		for _, c := range g.Controls {
			ctrl := parseControl(c, famID, famTitle, "")
			result = append(result, ctrl)
		}
	}

	// Traverse Top-level Controls
	for _, c := range doc.Catalog.Controls {
		ctrl := parseControl(c, "GENERAL", "General Controls", "")
		result = append(result, ctrl)
	}

	// Sort by ID
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// SearchControls searches controls within a framework matching the query.
func SearchControls(controls []ControlInfo, query string, limit int) []ControlInfo {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		if limit > 0 && len(controls) > limit {
			return controls[:limit]
		}
		return controls
	}

	var matches []ControlInfo
	for _, c := range controls {
		matched := false
		if strings.Contains(strings.ToLower(c.ID), q) ||
			strings.Contains(strings.ToLower(c.Title), q) ||
			strings.Contains(strings.ToLower(c.FamilyID), q) ||
			strings.Contains(strings.ToLower(c.FamilyTitle), q) ||
			strings.Contains(strings.ToLower(c.Guidance), q) ||
			strings.Contains(strings.ToLower(c.CanadianGuidance), q) {
			matched = true
		} else {
			for _, stmt := range c.Statements {
				if strings.Contains(strings.ToLower(stmt), q) {
					matched = true
					break
				}
			}
		}

		if matched {
			matches = append(matches, c)
			if limit > 0 && len(matches) >= limit {
				break
			}
		}
	}

	return matches
}

// GetControl retrieves a single control by ID from a loaded list of controls.
func GetControl(controls []ControlInfo, id string) (*ControlInfo, bool) {
	target := strings.ToLower(strings.TrimSpace(id))
	for i := range controls {
		if strings.ToLower(controls[i].ID) == target {
			return &controls[i], true
		}
	}
	return nil, false
}
