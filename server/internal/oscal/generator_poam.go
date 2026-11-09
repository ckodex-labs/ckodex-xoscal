package oscal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GeneratePOAM(ctx context.Context, snapshotName string, framework string) (*poamv1.PlanOfActionAndMilestones, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var risks []*poamv1.Risk
	var poamItems []*poamv1.PoamItem
	var roles []*commonv1.Role
	roleSeen := make(map[string]bool)
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err != nil {
			continue
		}
		if !strings.EqualFold(req.Framework, framework) || !req.Assessable {
			continue
		}
		title := singleLine(req.Title)
		if title == "" {
			title = singleLine(req.Citation)
		}
		desc := req.Text
		if desc == "" {
			desc = fmt.Sprintf("Action item for control %s (%s)", req.Citation, title)
		}
		risk := &poamv1.Risk{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("risk-%s", req.Citation)),
			Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Risk: %s", title)},
			Description: &commonv1.MarkupMultiline{Value: desc},
			Status:      &poamv1.RiskStatus{State: "open"},
		}
		// Build the POAM item with a distinct deterministic UUID.
		poamItem := &poamv1.PoamItem{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("poam-item-%s", req.Citation)),
			Title:       risk.Title,
			Description: risk.Description,
		}
		// Add remediation task for high-risk requirements.
		if strings.ToLower(req.RiskLevel) == "high" {
			resp := &poamv1.Response{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("remediate-%s", req.Citation)),
				Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Remediate %s", title)},
				Description: &commonv1.MarkupMultiline{Value: desc},
				Status:      &poamv1.ResponseStatus{State: "open"},
			}
			if req.Role != "" {
				roleTok := sanitizeToken(req.Role)
				if roleTok != "" {
					if !roleSeen[roleTok] {
						roleSeen[roleTok] = true
						roles = append(roles, &commonv1.Role{
							Id:    &commonv1.Token{Value: roleTok},
							Title: singleLine(req.Role),
						})
					}
					resp.ResponsibleRoles = append(resp.ResponsibleRoles, &poamv1.ResponsibleRole{
						RoleId: &commonv1.Token{Value: roleTok},
					})
				}
			}
			resp.RelatedTasks = append(resp.RelatedTasks, &poamv1.RelatedTask{
				TaskUuid: newUUID(uuidNamespace, fmt.Sprintf("task-%s", req.Citation)),
				Title:    &commonv1.MarkupLine{Value: fmt.Sprintf("Implement controls for %s", title)},
			})
			risk.Remediations = append(risk.Remediations, resp)
		}
		risks = append(risks, risk)
		poamItems = append(poamItems, poamItem)
	}

	return &poamv1.PlanOfActionAndMilestones{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("poam-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Plan of Action and Milestones", framework),
			Version: snapshotName,
			Roles:   roles,
		},
		ImportSsp: &poamv1.ImportSsp{
			Href: &commonv1.URIReference{Value: fmt.Sprintf("ssp-%s-%s", framework, snapshotName)},
		},
		Risks:      risks,
		PoamItems:  poamItems,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GenerateAssessmentResults builds OSCAL Assessment Results from a snapshot.
