package oscal

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
)

func (g *Generator) GeneratePOAM(ctx context.Context, snapshotName string, framework string) (*poamv1.PlanOfActionAndMilestones, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generatePOAMFromSnapshot(snap, snapshotName, framework), nil
}

func (g *Generator) generatePOAMFromSnapshot(snap *parsedSnapshot, snapshotName string, framework string) *poamv1.PlanOfActionAndMilestones {
	risks := make([]*poamv1.Risk, 0, len(snap.assessableReqs))
	poamItems := make([]*poamv1.PoamItem, 0, len(snap.assessableReqs))
	var roles []*commonv1.Role
	roleSeen := make(map[string]bool)

	for _, req := range snap.assessableReqs {
		ctrlID := sanitizeToken(req.Citation)
		if ctrlID == "" {
			ctrlID = sanitizeToken(req.NodeRefID)
		}

		title := singleLine(req.Title)
		if title == "" {
			title = singleLine(req.Citation)
		}
		desc := fmt.Sprintf("Action item and continuous monitoring milestone for control %s (%s).", ctrlID, title)
		risk := &poamv1.Risk{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("risk-%s", req.Citation)),
			Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Risk: %s", title)},
			Description: &commonv1.MarkupMultiline{Value: desc},
			Status:      &poamv1.RiskStatus{State: "open"},
		}
		var itemProps []*commonv1.Property
		if ctrlID != "" {
			itemProps = []*commonv1.Property{{
				Name:  "control-id",
				Ns:    xoscalPropNamespace,
				Value: ctrlID,
			}}
		}
		// Build the POAM item with a distinct deterministic UUID and control-id reference.
		poamItem := &poamv1.PoamItem{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("poam-item-%s", req.Citation)),
			Title:       risk.Title,
			Description: risk.Description,
			Props:       itemProps,
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
		Risks:      risks, //nolint:staticcheck // SA1019: backward-compatible deprecated proto field
		PoamItems:  poamItems,
		BackMatter: buildBackMatter(framework, snapshotName),
	}
}

// GenerateAssessmentResults builds OSCAL Assessment Results from a snapshot.
