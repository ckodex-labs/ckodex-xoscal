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
	return LoadControlsFromBytes(data, f.ID())
}

// LoadControlsFromBytes extracts all controls and enhancements from raw OSCAL catalog JSON bytes.
func LoadControlsFromBytes(data []byte, frameworkID string) ([]ControlInfo, error) {
	var doc oscalCatalogDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal catalog json for %s: %w", frameworkID, err)
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
			FrameworkID:     frameworkID,
			Parameters:      make(map[string]string),
			ParentControlID: strings.ToLower(parentID),
			IsEnhancement:   parentID != "",
		}

		// Check props for Canadian sovereignty and optional family metadata
		for _, prop := range c.Props {
			switch strings.ToLower(prop.Name) {
			case "jurisdiction":
				if strings.EqualFold(prop.Value, "ca") {
					info.IsCanadian = true
				}
			case "sovereign-guidance", "sovereign-control", "sovereign-specification":
				if strings.EqualFold(prop.Value, "true") {
					info.IsCanadian = true
				}
			case "family-id":
				if info.FamilyID == "" || info.FamilyID == "GENERAL" {
					info.FamilyID = strings.ToUpper(prop.Value)
				}
			case "family-title":
				if info.FamilyTitle == "" || info.FamilyTitle == "General Controls" {
					info.FamilyTitle = prop.Value
				}
			}
		}

		// Parameters
		for _, p := range c.Params {
			val := p.Label
			if len(p.Values) > 0 {
				val = strings.Join(p.Values, ", ")
			}
			if val == "" && p.Select != nil && len(p.Select.Choice) > 0 {
				val = strings.Join(p.Select.Choice, "; ")
			}
			info.Parameters[p.ID] = val
		}

		// Recursive helper to extract nested statement items
		var extractPartStmts func(part oscalJSONPart)
		var stmts []string
		extractPartStmts = func(part oscalJSONPart) {
			lbl := part.Title
			for _, prop := range part.Props {
				if strings.EqualFold(prop.Name, "label") && prop.Value != "" {
					lbl = prop.Value
				}
				if (strings.EqualFold(prop.Name, "jurisdiction") && strings.EqualFold(prop.Value, "ca")) ||
					(strings.EqualFold(prop.Name, "sovereign-specification") && strings.EqualFold(prop.Value, "true")) {
					info.IsCanadian = true
				}
			}
			if part.Prose != "" {
				if strings.EqualFold(part.Name, "item") && lbl != "" {
					stmts = append(stmts, lbl+" "+part.Prose)
				} else {
					stmts = append(stmts, part.Prose)
				}
			}
			for _, sub := range part.Parts {
				extractPartStmts(sub)
			}
		}

		// Parts
		for _, p := range c.Parts {
			pName := strings.ToLower(p.Name)
			isSovereignGuidance := pName == "gc-guidance" || pName == "canadian-guidance" || strings.EqualFold(p.Title, "GC Discussion")
			for _, prop := range p.Props {
				if (strings.EqualFold(prop.Name, "sovereign-guidance") && strings.EqualFold(prop.Value, "true")) ||
					(strings.EqualFold(prop.Name, "jurisdiction") && strings.EqualFold(prop.Value, "ca")) {
					isSovereignGuidance = true
				}
			}

			switch {
			case pName == "statement":
				extractPartStmts(p)
			case isSovereignGuidance && (pName == "guidance" || pName == "gc-guidance" || pName == "canadian-guidance"):
				if p.Prose != "" {
					info.CanadianGuidance = p.Prose
					info.IsCanadian = true
				}
			case pName == "guidance":
				if p.Prose != "" {
					info.Guidance = p.Prose
				}
			default:
				if !strings.Contains(pName, "assessment") {
					extractPartStmts(p)
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

	// Traverse Groups (including nested sub-groups)
	var traverseGroup func(g oscalJSONGroup, topFamID, topFamTitle string)
	traverseGroup = func(g oscalJSONGroup, topFamID, topFamTitle string) {
		famID := topFamID
		famTitle := topFamTitle
		if famID == "" {
			famID = g.ID
			famTitle = g.Title
		}
		for _, c := range g.Controls {
			ctrl := parseControl(c, famID, famTitle, "")
			result = append(result, ctrl)
		}
		for _, sub := range g.Groups {
			traverseGroup(sub, famID, famTitle)
		}
	}
	for _, g := range doc.Catalog.Groups {
		traverseGroup(g, g.ID, g.Title)
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
