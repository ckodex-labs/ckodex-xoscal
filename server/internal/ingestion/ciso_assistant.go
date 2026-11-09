package ingestion

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CISOLibrary represents the top-level structure of a CISO Assistant library YAML file.
type CISOLibrary struct {
	URN             string      `yaml:"urn"`
	Locale          string      `yaml:"locale"`
	RefID           string      `yaml:"ref_id"`
	Name            string      `yaml:"name"`
	Description     string      `yaml:"description"`
	Copyright       string      `yaml:"copyright"`
	Version         string      `yaml:"version"`
	PublicationDate string      `yaml:"publication_date"`
	Provider        string      `yaml:"provider"`
	Packager        string      `yaml:"packager"`
	Objects         CISOObjects `yaml:"objects"`
}

// CISOObjects holds the framework and optionally mapping sets inside a library.
type CISOObjects struct {
	Framework CISOLibraryFramework `yaml:"framework"`
}

// CISOLibraryFramework represents the framework object within a CISO library.
type CISOLibraryFramework struct {
	URN                     string                       `yaml:"urn"`
	RefID                   string                       `yaml:"ref_id"`
	Name                    string                       `yaml:"name"`
	Description             string                       `yaml:"description"`
	ImplementationGroupsDef []CISOImplementationGroupDef `yaml:"implementation_groups_definition"`
	RequirementNodes        []CISORequirementNode        `yaml:"requirement_nodes"`
}

// CISOImplementationGroupDef represents an implementation group definition.
type CISOImplementationGroupDef struct {
	RefID       string `yaml:"ref_id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// CISOQuestion represents an assessment question entry on a requirement node.
type CISOQuestion struct {
	Text string `yaml:"text"`
}

// CISORequirementNode represents a single requirement node in the hierarchy.
type CISORequirementNode struct {
	URN                  string                  `yaml:"urn"`
	Assessable           bool                    `yaml:"assessable"`
	Depth                int                     `yaml:"depth"`
	ParentURN            string                  `yaml:"parent_urn"`
	RefID                string                  `yaml:"ref_id"`
	Name                 string                  `yaml:"name"`
	Description          string                  `yaml:"description"`
	Annotation           string                  `yaml:"annotation"`
	TypicalEvidence      string                  `yaml:"typical_evidence"`
	Questions            map[string]CISOQuestion `yaml:"questions"`
	ImplementationGroups []string                `yaml:"implementation_groups"`
}

var odpBracketRegex = regexp.MustCompile(`(?i)\[\s*(?:Assignment|Selection|Affectation|Sélection)\s*(?:\([^)\[\]]*\))?\s*:?\s*([^\[\]]+?)\s*\]`)

// CISOAssistantParser reads CISO Assistant library YAML files.
type CISOAssistantParser struct{}

// Parse converts raw CISO Assistant YAML into structured Requirements.
func (p *CISOAssistantParser) Parse(ctx context.Context, raw []byte) ([]Requirement, error) {
	var lib CISOLibrary
	if err := yaml.Unmarshal(raw, &lib); err != nil {
		return nil, fmt.Errorf("unmarshal ciso library: %w", err)
	}

	frameworkRefID := lib.RefID
	if frameworkRefID == "" {
		frameworkRefID = lib.Objects.Framework.RefID
	}

	nodes := lib.Objects.Framework.RequirementNodes
	byURN := make(map[string]CISORequirementNode, len(nodes))
	refIDCounts := make(map[string]int, len(nodes))
	for _, node := range nodes {
		if node.URN != "" {
			byURN[node.URN] = node
		}
		tok := strings.ToLower(sanitizeCitationToken(strings.TrimSpace(node.RefID)))
		if tok != "" {
			refIDCounts[tok]++
		}
	}

	var reqs []Requirement
	for _, node := range nodes {
		normID := normalizeURN(node.URN)
		citation := strings.TrimSpace(node.RefID)
		tok := strings.ToLower(sanitizeCitationToken(citation))
		if citation == "" || refIDCounts[tok] > 1 {
			citation = normID
		}

		text := strings.TrimSpace(node.Description)
		if text == "" && len(node.Questions) > 0 {
			var qKeys []string
			for k := range node.Questions {
				qKeys = append(qKeys, k)
			}
			sort.Strings(qKeys)
			var qTexts []string
			for _, k := range qKeys {
				if qt := strings.TrimSpace(node.Questions[k].Text); qt != "" {
					qTexts = append(qTexts, qt)
				}
			}
			text = strings.Join(qTexts, "\n")
		}
		if text == "" {
			text = strings.TrimSpace(node.Name)
		}

		var params []ParamDef
		if node.Assessable && text != "" {
			text, params = extractODPParams(citation, text)
		}

		title := strings.TrimSpace(node.Name)
		if !node.Assessable {
			if title == "" {
				title = strings.TrimSpace(node.Description)
			}
			if title == "" {
				title = citation
			}
		} else if isStubTitle(citation, title) {
			ancTitle := resolveAncestorTitle(node, byURN)
			if ancTitle != "" {
				title = fmt.Sprintf("%s — %s", ancTitle, citation)
			} else if text != "" {
				firstLine := strings.TrimSpace(strings.Split(text, "\n")[0])
				if len(firstLine) > 80 {
					firstLine = strings.TrimSpace(firstLine[:77]) + "..."
				}
				title = fmt.Sprintf("%s (%s)", firstLine, citation)
			} else {
				title = fmt.Sprintf("Requirement %s", citation)
			}
		}
		title = cleanSingleLine(title)

		var guidanceParts []string
		if ann := strings.TrimSpace(node.Annotation); ann != "" {
			guidanceParts = append(guidanceParts, ann)
		}
		if ev := strings.TrimSpace(node.TypicalEvidence); ev != "" {
			guidanceParts = append(guidanceParts, "Typical evidence:\n"+ev)
		}
		guidance := strings.Join(guidanceParts, "\n\n")

		nodeRefID := strings.TrimSpace(node.RefID)
		if nodeRefID == "" || refIDCounts[tok] > 1 {
			nodeRefID = citation
		}

		req := Requirement{
			ID:                   normID,
			Citation:             citation,
			Title:                title,
			Text:                 text,
			Guidance:             guidance,
			Params:               params,
			Framework:            frameworkRefID,
			Depth:                node.Depth,
			ParentURN:            node.ParentURN,
			Assessable:           node.Assessable,
			ImplementationGroups: node.ImplementationGroups,
			NodeRefID:            nodeRefID,
			NodeName:             title,
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func isStubTitle(citation, title string) bool {
	t := strings.TrimSpace(title)
	if t == "" {
		return true
	}
	cLower := strings.ToLower(strings.TrimSpace(citation))
	cTok := strings.TrimPrefix(strings.ToLower(sanitizeCitationToken(citation)), "_")
	tLower := strings.ToLower(t)
	if tLower == cLower || tLower == cTok || strings.TrimPrefix(tLower, "_") == cTok {
		return true
	}
	if strings.HasPrefix(tLower, "control ") && len(tLower) <= len(cTok)+10 {
		return true
	}
	return false
}

func resolveAncestorTitle(node CISORequirementNode, byURN map[string]CISORequirementNode) string {
	curr := node.ParentURN
	visited := make(map[string]bool)
	for curr != "" && !visited[curr] {
		visited[curr] = true
		pNode, ok := byURN[curr]
		if !ok {
			break
		}
		pRef := strings.TrimSpace(pNode.RefID)
		if pRef == "" {
			pRef = normalizeURN(pNode.URN)
		}
		pName := strings.TrimSpace(pNode.Name)
		if pName != "" && !isStubTitle(pRef, pName) {
			return pName
		}
		pDesc := strings.TrimSpace(pNode.Description)
		if pDesc != "" && len(pDesc) <= 120 && !isStubTitle(pRef, pDesc) {
			return pDesc
		}
		curr = pNode.ParentURN
	}
	return ""
}

func extractODPParams(citation, text string) (string, []ParamDef) {
	var params []ParamDef
	baseID := strings.ToLower(sanitizeCitationToken(citation))
	if baseID == "" {
		baseID = "_param"
	}
	counter := 0
	for {
		loc := odpBracketRegex.FindStringSubmatchIndex(text)
		if loc == nil {
			break
		}
		counter++
		paramID := fmt.Sprintf("%s_prm_%d", baseID, counter)
		label := cleanSingleLine(text[loc[2]:loc[3]])
		params = append(params, ParamDef{
			ID:    paramID,
			Label: label,
		})
		text = text[:loc[0]] + fmt.Sprintf("{{ insert: param, %s }}", paramID) + text[loc[1]:]
	}
	return text, params
}

func cleanSingleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeCitationToken(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			if i == 0 && (r == '.' || r == '-') {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	res := b.String()
	if len(res) > 0 {
		f := res[0]
		if (f >= '0' && f <= '9') || f == '.' || f == '-' {
			res = "_" + res
		}
	}
	return res
}

// normalizeURN strips the CISO-specific prefix to produce a stable, collision-free ID.
func normalizeURN(urn string) string {
	// CISO URNs look like: urn:intuitem:risk:req_node:<framework>:<id...>
	if urn == "" {
		return ""
	}
	parts := strings.Split(urn, ":")
	if len(parts) >= 6 && parts[0] == "urn" && parts[3] == "req_node" {
		return strings.Join(parts[5:], ".")
	}
	for i := len(urn) - 1; i >= 0; i-- {
		if urn[i] == ':' {
			return urn[i+1:]
		}
	}
	return urn
}
