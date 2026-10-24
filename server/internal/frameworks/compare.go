package frameworks

import (
	"fmt"
	"sort"
	"strings"
)

// FrameworkComparison holds the comparative delta between two compliance frameworks.
type FrameworkComparison struct {
	FromFramework string        `json:"from_framework"`
	ToFramework   string        `json:"to_framework"`
	FromTitle     string        `json:"from_title"`
	ToTitle       string        `json:"to_title"`
	FromTotal     int           `json:"from_total"`
	ToTotal       int           `json:"to_total"`
	NetDelta      int           `json:"net_delta"`
	AddedCount    int           `json:"added_count"`
	RetiredCount  int           `json:"retired_count"`
	ModifiedCount int           `json:"modified_count"`
	SharedCount   int           `json:"shared_count"`
	Added         []ControlInfo `json:"added"`
	Retired       []ControlInfo `json:"retired"`
	Modified      []DeltaInfo   `json:"modified"`
	Families      []FamilyDelta `json:"families"`
}

// DeltaInfo captures change details for a control present in both frameworks.
type DeltaInfo struct {
	ID                    string   `json:"id"`
	TitleChange           bool     `json:"title_change"`
	FromTitle             string   `json:"from_title"`
	ToTitle               string   `json:"to_title"`
	AddedEnhancements     []string `json:"added_enhancements,omitempty"`
	RemovedEnhancements   []string `json:"removed_enhancements,omitempty"`
	StatementsChanged     bool     `json:"statements_changed"`
	CanadianGuidanceAdded bool     `json:"canadian_guidance_added"`
}

// FamilyDelta captures control count evolution per security family.
type FamilyDelta struct {
	FamilyID   string `json:"family_id"`
	FamilyName string `json:"family_name"`
	FromCount  int    `json:"from_count"`
	ToCount    int    `json:"to_count"`
	NetDelta   int    `json:"net_delta"`
}

// CompareFrameworks evaluates two frameworks and computes the detailed transition delta.
func CompareFrameworks(fromFw, toFw Framework) (*FrameworkComparison, error) {
	fromControls, err := LoadControls(fromFw)
	if err != nil {
		return nil, fmt.Errorf("load controls for from-framework: %w", err)
	}

	toControls, err := LoadControls(toFw)
	if err != nil {
		return nil, fmt.Errorf("load controls for to-framework: %w", err)
	}

	fromMap := make(map[string]ControlInfo, len(fromControls))
	for _, c := range fromControls {
		fromMap[c.ID] = c
	}

	toMap := make(map[string]ControlInfo, len(toControls))
	for _, c := range toControls {
		toMap[c.ID] = c
	}

	comp := &FrameworkComparison{
		FromFramework: fromFw.ID(),
		ToFramework:   toFw.ID(),
		FromTitle:     fromFw.Title(),
		ToTitle:       toFw.Title(),
		FromTotal:     len(fromControls),
		ToTotal:       len(toControls),
		NetDelta:      len(toControls) - len(fromControls),
	}

	// 1. Added controls (in toMap, not in fromMap)
	for _, toC := range toControls {
		if _, exists := fromMap[toC.ID]; !exists {
			comp.Added = append(comp.Added, toC)
		}
	}
	comp.AddedCount = len(comp.Added)

	// 2. Retired controls (in fromMap, not in toMap)
	for _, fromC := range fromControls {
		if _, exists := toMap[fromC.ID]; !exists {
			comp.Retired = append(comp.Retired, fromC)
		}
	}
	comp.RetiredCount = len(comp.Retired)

	// 3. Shared & Modified controls
	for _, toC := range toControls {
		if fromC, exists := fromMap[toC.ID]; exists {
			comp.SharedCount++
			delta := DeltaInfo{
				ID:        toC.ID,
				FromTitle: fromC.Title,
				ToTitle:   toC.Title,
			}

			if strings.TrimSpace(fromC.Title) != strings.TrimSpace(toC.Title) {
				delta.TitleChange = true
			}

			if len(fromC.Statements) != len(toC.Statements) {
				delta.StatementsChanged = true
			}

			if fromC.CanadianGuidance == "" && toC.CanadianGuidance != "" {
				delta.CanadianGuidanceAdded = true
			}

			// Enhancements diff
			fromEnhSet := make(map[string]bool)
			for _, e := range fromC.Enhancements {
				fromEnhSet[e] = true
			}
			toEnhSet := make(map[string]bool)
			for _, e := range toC.Enhancements {
				toEnhSet[e] = true
			}

			for _, e := range toC.Enhancements {
				if !fromEnhSet[e] {
					delta.AddedEnhancements = append(delta.AddedEnhancements, e)
				}
			}
			for _, e := range fromC.Enhancements {
				if !toEnhSet[e] {
					delta.RemovedEnhancements = append(delta.RemovedEnhancements, e)
				}
			}

			if delta.TitleChange || delta.StatementsChanged || delta.CanadianGuidanceAdded ||
				len(delta.AddedEnhancements) > 0 || len(delta.RemovedEnhancements) > 0 {
				comp.Modified = append(comp.Modified, delta)
			}
		}
	}
	comp.ModifiedCount = len(comp.Modified)

	// 4. Family Transition Matrix
	familyMap := make(map[string]*FamilyDelta)
	for _, c := range fromControls {
		fid := c.FamilyID
		if fid == "" {
			fid = "OTHER"
		}
		if _, ok := familyMap[fid]; !ok {
			familyMap[fid] = &FamilyDelta{
				FamilyID:   fid,
				FamilyName: c.FamilyTitle,
			}
		}
		familyMap[fid].FromCount++
	}

	for _, c := range toControls {
		fid := c.FamilyID
		if fid == "" {
			fid = "OTHER"
		}
		if _, ok := familyMap[fid]; !ok {
			familyMap[fid] = &FamilyDelta{
				FamilyID:   fid,
				FamilyName: c.FamilyTitle,
			}
		}
		familyMap[fid].ToCount++
	}

	for _, fd := range familyMap {
		fd.NetDelta = fd.ToCount - fd.FromCount
		comp.Families = append(comp.Families, *fd)
	}

	sort.Slice(comp.Families, func(i, j int) bool {
		return comp.Families[i].FamilyID < comp.Families[j].FamilyID
	})

	return comp, nil
}

// FormatTerminal generates clean ASCII formatted transition report for CLI.
func (c *FrameworkComparison) FormatTerminal() string {
	var sb strings.Builder
	sb.WriteString("\n=== SOVEREIGN FRAMEWORK TRANSITION DELTA ===\n")
	sb.WriteString(fmt.Sprintf("From: %s (%s)\n", c.FromFramework, c.FromTitle))
	sb.WriteString(fmt.Sprintf("To:   %s (%s)\n\n", c.ToFramework, c.ToTitle))

	sign := "+"
	if c.NetDelta < 0 {
		sign = ""
	}
	sb.WriteString(fmt.Sprintf("Summary: %d Base Controls -> %d Target Controls (Net: %s%d)\n",
		c.FromTotal, c.ToTotal, sign, c.NetDelta))
	sb.WriteString(fmt.Sprintf("Delta:   %d Added | %d Retired | %d Modified | %d Unchanged\n\n",
		c.AddedCount, c.RetiredCount, c.ModifiedCount, c.SharedCount-c.ModifiedCount))

	sb.WriteString("--- FAMILY-LEVEL TRANSITION MATRIX ---\n")
	sb.WriteString(fmt.Sprintf("%-10s %-28s %-12s %-12s %s\n", "FAMILY", "NAME", "FROM", "TO", "NET CHANGE"))
	sb.WriteString(strings.Repeat("-", 76) + "\n")
	for _, f := range c.Families {
		chSign := "+"
		if f.NetDelta < 0 {
			chSign = ""
		}
		sb.WriteString(fmt.Sprintf("%-10s %-28s %-12d %-12d %s%d\n",
			f.FamilyID, truncateStr(f.FamilyName, 26), f.FromCount, f.ToCount, chSign, f.NetDelta))
	}
	sb.WriteString("\n")

	if len(c.Added) > 0 {
		sb.WriteString(fmt.Sprintf("--- KEY ADDITIONS IN %s (Showing Top 15 of %d) ---\n", c.ToFramework, len(c.Added)))
		limit := 15
		if len(c.Added) < limit {
			limit = len(c.Added)
		}
		for i := 0; i < limit; i++ {
			ctrl := c.Added[i]
			caTag := ""
			if ctrl.IsCanadian {
				caTag = " [CA SOVEREIGN]"
			}
			sb.WriteString(fmt.Sprintf("  %-14s %s%s\n", ctrl.ID, ctrl.Title, caTag))
		}
		if len(c.Added) > limit {
			sb.WriteString(fmt.Sprintf("  ... and %d more added controls\n", len(c.Added)-limit))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatMarkdown generates standard GitHub/GitLab markdown documentation.
func (c *FrameworkComparison) FormatMarkdown() string {
	var sb strings.Builder
	sb.WriteString("# Sovereign Framework Transition Report\n\n")
	sb.WriteString(fmt.Sprintf("**Baseline Framework:** `%s` — %s  \n", c.FromFramework, c.FromTitle))
	sb.WriteString(fmt.Sprintf("**Target Framework:** `%s` — %s  \n\n", c.ToFramework, c.ToTitle))

	sign := "+"
	if c.NetDelta < 0 {
		sign = ""
	}
	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Requirements Delta:** `%d` -> `%d` (`%s%d` net change)\n",
		c.FromTotal, c.ToTotal, sign, c.NetDelta))
	sb.WriteString(fmt.Sprintf("- **New Controls / Enhancements:** `%d`\n", c.AddedCount))
	sb.WriteString(fmt.Sprintf("- **Modified Controls:** `%d`\n", c.ModifiedCount))
	sb.WriteString(fmt.Sprintf("- **Retired Controls:** `%d`\n\n", c.RetiredCount))

	sb.WriteString("## Family Transition Breakdown\n\n")
	sb.WriteString("| Family ID | Family Name | Base Count | Target Count | Net Delta |\n")
	sb.WriteString("|---|---|---|---|---|\n")
	for _, f := range c.Families {
		chSign := "+"
		if f.NetDelta < 0 {
			chSign = ""
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | `%s%d` |\n",
			f.FamilyID, f.FamilyName, f.FromCount, f.ToCount, chSign, f.NetDelta))
	}
	sb.WriteString("\n")

	return sb.String()
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
