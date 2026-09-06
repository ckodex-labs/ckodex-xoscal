package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

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
	var roots []string

	// First pass: index all requirements.
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			continue
		}
		if req.Framework != framework {
			continue
		}
		byURN[req.URN] = &node{req: req}
	}

	// Second pass: build parent-child relationships.
	for _, n := range byURN {
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

	buildControl := func(req kg.Requirement) *catalogv1.Control {
		// Skip controls with no citation — they cannot have a valid OSCAL id.
		if req.Citation == "" {
			return nil
		}
		// Fallback: use citation as title when name is empty (common for
		// leaf requirement nodes that only have ref_id + description).
		title := req.Title
		if title == "" {
			title = req.Citation
		}
		ctrl := &catalogv1.Control{
			Id:    &commonv1.Token{Value: sanitizeToken(req.Citation)},
			Title: &commonv1.MarkupLine{Value: title},
			Props: requirementToProps(req),
			Parts: []*catalogv1.Part{
				{Id: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-stmt"}, Name: "statement", Prose: []*commonv1.MarkupMultiline{{Value: req.Text}}},
			},
		}
		// Add parameters for requirements with sections/subsections.
		if req.Section != "" {
			ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
				Id: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-param"},
				Label: &commonv1.MarkupLine{
					Value: fmt.Sprintf("Parameter for %s", title),
				},
			})
		}
		if req.Subsection != "" {
			ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
				Id: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-subparam"},
				Label: &commonv1.MarkupLine{
					Value: fmt.Sprintf("Sub-parameter for %s", title),
				},
			})
		}
		// Add guidance part if text contains guidance-like sentences.
		if guidance := extractGuidance(req.Text); guidance != "" {
			ctrl.Parts = append(ctrl.Parts, &catalogv1.Part{
				Id:    &commonv1.Token{Value: sanitizeToken(req.Citation) + "-guidance"},
				Name:  "guidance",
				Prose: []*commonv1.MarkupMultiline{{Value: guidance}},
			})
		}
		return ctrl
	}

	var buildGroup func(urn string) *catalogv1.Group
	buildGroup = func(urn string) *catalogv1.Group {
		n, ok := byURN[urn]
		if !ok {
			return nil
		}
		// Fallback: use ref_id as title when node name is empty.
		groupTitle := n.req.NodeName
		if groupTitle == "" {
			groupTitle = n.req.NodeRefID
		}
		g := &catalogv1.Group{
			Id:    &commonv1.Token{Value: sanitizeToken(n.req.NodeRefID)},
			Title: &commonv1.MarkupLine{Value: groupTitle},
		}
		if n.req.Assessable {
			if c := buildControl(n.req); c != nil {
				g.Controls = append(g.Controls, c)
			}
		}
		for _, childURN := range n.children {
			childNode, ok := byURN[childURN]
			if !ok {
				continue
			}
			// Assessable leaf nodes (no children) become controls
			// directly in the parent group, avoiding wrapper groups
			// with single controls.
			if childNode.req.Assessable && len(childNode.children) == 0 {
				if c := buildControl(childNode.req); c != nil {
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
		return g
	}

	var groups []*catalogv1.Group
	var controls []*catalogv1.Control
	for _, urn := range roots {
		n, ok := byURN[urn]
		if !ok {
			continue
		}
		if n.req.Assessable && len(n.children) == 0 {
			if c := buildControl(n.req); c != nil {
				controls = append(controls, c)
			}
		} else {
			if g := buildGroup(urn); g != nil {
				groups = append(groups, g)
			}
		}
	}

	return &catalogv1.Catalog{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("catalog-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Control Catalog", framework),
			Version: snapshotName,
		},
		Controls:   controls,
		Groups:     groups,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GenerateProfile builds an OSCAL Profile from a snapshot with selected controls.
