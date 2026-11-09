package oscal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Join(strings.Fields(s), " ")
}

func (g *Generator) GenerateCatalog(ctx context.Context, snapshotName string, framework string) (*catalogv1.Catalog, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	type node struct {
		req      kg.Requirement
		children []string
	}
	byURN := make(map[string]*node)
	var orderedURNs []string
	var roots []string
	var fwMeta *kg.Framework

	// First pass: index all requirements and framework metadata in deterministic order.
	for _, e := range entities {
		if e.Type == "reg:Framework" {
			var fw kg.Framework
			if err := json.Unmarshal(e.Payload, &fw); err == nil && strings.EqualFold(fw.RefID, framework) {
				fwCopy := fw
				fwMeta = &fwCopy
			}
			continue
		}
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			continue
		}
		if !strings.EqualFold(req.Framework, framework) {
			continue
		}
		if _, exists := byURN[req.URN]; !exists {
			orderedURNs = append(orderedURNs, req.URN)
		}
		byURN[req.URN] = &node{req: req}
	}

	// Second pass: build parent-child relationships preserving document order.
	for _, urn := range orderedURNs {
		n := byURN[urn]
		if n.req.ParentURN != "" {
			if parent, ok := byURN[n.req.ParentURN]; ok {
				parent.children = append(parent.children, n.req.URN)
			} else {
				roots = append(roots, n.req.URN)
			}
		} else {
			roots = append(roots, n.req.URN)
		}
	}

	usedIDs := make(map[string]bool)
	uniqueToken := func(preferred, fallback string) string {
		tok := sanitizeToken(preferred)
		if tok == "" {
			tok = sanitizeToken(fallback)
		}
		if tok == "" {
			tok = "_node"
		}
		if !usedIDs[strings.ToLower(tok)] {
			usedIDs[strings.ToLower(tok)] = true
			return tok
		}
		fbTok := sanitizeToken(fallback)
		if fbTok != "" && !usedIDs[strings.ToLower(fbTok)] {
			usedIDs[strings.ToLower(fbTok)] = true
			return fbTok
		}
		for i := 2; ; i++ {
			cand := fmt.Sprintf("%s_%d", tok, i)
			if !usedIDs[strings.ToLower(cand)] {
				usedIDs[strings.ToLower(cand)] = true
				return cand
			}
		}
	}

	var buildControl func(urn string) *catalogv1.Control
	var collectSubControls func(urn string) []*catalogv1.Control

	collectSubControls = func(urn string) []*catalogv1.Control {
		n, ok := byURN[urn]
		if !ok {
			return nil
		}
		var subs []*catalogv1.Control
		for _, childURN := range n.children {
			childNode, ok := byURN[childURN]
			if !ok {
				continue
			}
			if childNode.req.Assessable {
				if childCtrl := buildControl(childURN); childCtrl != nil {
					subs = append(subs, childCtrl)
				}
			} else {
				subs = append(subs, collectSubControls(childURN)...)
			}
		}
		return subs
	}

	var kgPartsToProto func(parts []kg.PartDef) []*catalogv1.Part
	kgPartsToProto = func(parts []kg.PartDef) []*catalogv1.Part {
		var out []*catalogv1.Part
		for _, p := range parts {
			var pProps []*commonv1.Property
			for _, pr := range p.Props {
				val := singleLine(pr.Value)
				if pr.Name == "" || val == "" {
					continue
				}
				pProps = append(pProps, &commonv1.Property{
					Id:    pr.ID,
					Name:  pr.Name,
					Value: val,
					Class: pr.Class,
					Ns:    normalizePropNs(pr.Name, pr.Ns),
				})
			}
			pNs := p.Ns
			if pNs == "" {
				switch strings.ToLower(p.Name) {
				case "statement", "guidance", "item", "objective", "assessment-objective", "assessment-method", "assessment-objects", "overview":
				case "gc-guidance", "canadian-guidance":
					pNs = "https://ised-isde.canada.ca/ns/oscal"
				default:
					pNs = xoscalPropNamespace
				}
			}
			part := &catalogv1.Part{
				Name:  p.Name,
				Ns:    pNs,
				Class: p.Class,
				Props: pProps,
				Parts: kgPartsToProto(p.Parts),
			}
			if p.ID != "" {
				part.Id = &commonv1.Token{Value: sanitizeToken(p.ID)}
			}
			if p.Title != "" {
				part.Title = &commonv1.MarkupLine{Value: singleLine(p.Title)}
			}
			if p.Prose != "" {
				part.Prose = []*commonv1.MarkupMultiline{{Value: p.Prose}}
			}
			out = append(out, part)
		}
		return out
	}

	buildControl = func(urn string) *catalogv1.Control {
		n, ok := byURN[urn]
		if !ok {
			return nil
		}
		req := n.req
		citation := req.Citation
		if citation == "" {
			citation = req.NodeRefID
		}
		if citation == "" {
			return nil
		}
		ctrlID := uniqueToken(citation, req.URN)
		title := singleLine(req.Title)
		if title == "" {
			title = singleLine(citation)
		}
		ctrl := &catalogv1.Control{
			Id:    &commonv1.Token{Value: ctrlID},
			Class: req.Class,
			Title: &commonv1.MarkupLine{Value: title},
			Props: requirementToProps(req),
		}
		if len(req.Parts) > 0 {
			ctrl.Parts = kgPartsToProto(req.Parts)
		} else {
			ctrl.Parts = []*catalogv1.Part{
				{Id: &commonv1.Token{Value: ctrlID + "-stmt"}, Name: "statement", Prose: []*commonv1.MarkupMultiline{{Value: req.Text}}},
			}
			guidance := req.Guidance
			if guidance == "" && req.NodeRefID == "" {
				guidance = extractGuidance(req.Text)
			}
			if guidance != "" {
				ctrl.Parts = append(ctrl.Parts, &catalogv1.Part{
					Id:    &commonv1.Token{Value: ctrlID + "-guidance"},
					Name:  "guidance",
					Prose: []*commonv1.MarkupMultiline{{Value: guidance}},
				})
			}
			if req.CanadianGuidance != "" && req.CanadianGuidance != guidance {
				ctrl.Parts = append(ctrl.Parts, &catalogv1.Part{
					Id:    &commonv1.Token{Value: ctrlID + "-gc-guidance"},
					Name:  "gc-guidance",
					Ns:    "https://ised-isde.canada.ca/ns/oscal",
					Title: &commonv1.MarkupLine{Value: "GC Discussion"},
					Prose: []*commonv1.MarkupMultiline{{Value: req.CanadianGuidance}},
					Props: []*commonv1.Property{
						{Name: "jurisdiction", Value: "ca", Ns: "https://ised-isde.canada.ca/ns/oscal"},
						{Name: "sovereign-guidance", Value: "true", Ns: "https://ised-isde.canada.ca/ns/oscal"},
					},
				})
			}
		}
		if len(req.Params) > 0 {
			for _, p := range req.Params {
				param := &catalogv1.Parameter{
					Id:    &commonv1.Token{Value: sanitizeToken(p.ID)},
					Class: p.Class,
				}
				if p.Label != "" {
					param.Label = &commonv1.MarkupLine{Value: singleLine(p.Label)}
				}
				for _, v := range p.Values {
					if sv := singleLine(v); sv != "" {
						param.Values = append(param.Values, &commonv1.MarkupLine{Value: sv})
					}
				}
				if len(p.Choices) > 0 || p.HowMany != "" {
					sel := &catalogv1.Select{
						HowMany: p.HowMany == "one-or-more" || p.HowMany == "true",
					}
					for _, c := range p.Choices {
						if sc := singleLine(c); sc != "" {
							sel.Choices = append(sel.Choices, &commonv1.MarkupLine{Value: sc})
						}
					}
					param.Select = sel
				}
				ctrl.Params = append(ctrl.Params, param)
			}
		} else if req.NodeRefID == "" {
			// Legacy fallback for requirements with section/subsection but no explicit params.
			if req.Section != "" {
				ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
					Id: &commonv1.Token{Value: ctrlID + "-param"},
					Label: &commonv1.MarkupLine{
						Value: fmt.Sprintf("Parameter for %s", title),
					},
				})
			}
			if req.Subsection != "" {
				ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
					Id: &commonv1.Token{Value: ctrlID + "-subparam"},
					Label: &commonv1.MarkupLine{
						Value: fmt.Sprintf("Sub-parameter for %s", title),
					},
				})
			}
		}
		ctrl.Controls = collectSubControls(urn)
		return ctrl
	}

	var buildGroup func(urn string) *catalogv1.Group
	buildGroup = func(urn string) *catalogv1.Group {
		n, ok := byURN[urn]
		if !ok {
			return nil
		}
		groupRawID := n.req.NodeRefID
		if groupRawID == "" {
			groupRawID = n.req.Citation
		}
		groupID := uniqueToken(groupRawID, n.req.URN)
		groupTitle := singleLine(n.req.NodeName)
		if groupTitle == "" {
			groupTitle = singleLine(n.req.Title)
		}
		if groupTitle == "" {
			groupTitle = singleLine(groupID)
		}
		g := &catalogv1.Group{
			Id:    &commonv1.Token{Value: groupID},
			Title: &commonv1.MarkupLine{Value: groupTitle},
		}
		if n.req.Assessable {
			if c := buildControl(urn); c != nil {
				g.Controls = append(g.Controls, c)
			}
			return g
		}
		for _, childURN := range n.children {
			childNode, ok := byURN[childURN]
			if !ok {
				continue
			}
			if childNode.req.Assessable {
				if c := buildControl(childURN); c != nil {
					g.Controls = append(g.Controls, c)
				}
			} else if childGroup := buildGroup(childURN); childGroup != nil {
				g.Groups = append(g.Groups, childGroup)
			}
		}
		// Return nil if group has no controls and no subgroups (empty group).
		if len(g.Controls) == 0 && len(g.Groups) == 0 {
			return nil
		}
		// OSCAL 1.2.3 group schema (anyOf with additionalProperties: false) forbids
		// a group from having both "groups" and "controls" simultaneously.
		if len(g.Groups) > 0 && len(g.Controls) > 0 {
			directGroup := &catalogv1.Group{
				Id:       &commonv1.Token{Value: uniqueToken(groupID+"-controls", n.req.URN+"-controls")},
				Title:    &commonv1.MarkupLine{Value: groupTitle},
				Controls: g.Controls,
			}
			g.Groups = append([]*catalogv1.Group{directGroup}, g.Groups...)
			g.Controls = nil
		}
		return g
	}

	var groups []*catalogv1.Group
	var controls []*catalogv1.Control
	for _, urn := range roots {
		n, ok := byURN[urn]
		if !ok {
			continue
		}
		if n.req.Assessable {
			if c := buildControl(urn); c != nil {
				controls = append(controls, c)
			}
		} else {
			if g := buildGroup(urn); g != nil {
				groups = append(groups, g)
			}
		}
	}

	catTitle := fmt.Sprintf("%s Control Catalog", framework)
	catVersion := snapshotName
	if fwMeta != nil {
		if fwMeta.Name != "" {
			catTitle = singleLine(fwMeta.Name)
		}
		if fwMeta.Version != "" {
			catVersion = singleLine(fwMeta.Version)
		}
	}

	return &catalogv1.Catalog{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("catalog-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   catTitle,
			Version: catVersion,
		},
		Controls:   controls,
		Groups:     groups,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GenerateProfile builds an OSCAL Profile from a snapshot with selected controls.
