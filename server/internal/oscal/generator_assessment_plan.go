package oscal

import (
	"context"
	"fmt"
	"strings"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
)

func (g *Generator) GenerateAssessmentPlan(ctx context.Context, snapshotName string, framework string) (*assessment_planv1.AssessmentPlan, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generateAssessmentPlanFromSnapshot(snap, snapshotName, framework), nil
}

func (g *Generator) generateAssessmentPlanFromSnapshot(snap *parsedSnapshot, snapshotName string, framework string) *assessment_planv1.AssessmentPlan {
	const maxSampleTasks = 25
	taskCap := len(snap.assessableReqs)
	if taskCap > maxSampleTasks {
		taskCap = maxSampleTasks
	}
	tasks := make([]*assessment_planv1.AssessmentTask, 0, taskCap)
	includeControls := make([]*assessment_planv1.SelectedControl, 0, len(snap.assessableReqs))
	seenControls := make(map[string]bool, len(snap.assessableReqs))

	for _, req := range snap.assessableReqs {
		ctrlID := sanitizeToken(req.Citation)
		if ctrlID == "" {
			ctrlID = sanitizeToken(req.NodeRefID)
		}
		if ctrlID != "" {
			key := strings.ToLower(ctrlID)
			if !seenControls[key] {
				seenControls[key] = true
				includeControls = append(includeControls, &assessment_planv1.SelectedControl{
					ControlId: &commonv1.Token{Value: ctrlID},
				})
			}
		}

		if len(tasks) < maxSampleTasks {
			title := singleLine(req.Title)
			if title == "" {
				title = singleLine(req.Citation)
			}
			tasks = append(tasks, &assessment_planv1.AssessmentTask{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("task-%s", req.Citation)),
				Type:        "examination",
				Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Assess %s", title)},
				Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("Assess control %s (%s)", req.Citation, title)},
			})
		}
	}

	sel := &assessment_planv1.ControlSelection{
		Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("Controls reviewed for %s", framework)},
	}
	if len(includeControls) > 0 {
		sel.IncludeControls = includeControls
	} else {
		sel.IncludeAll = &assessment_planv1.IncludeAll{}
	}

	return &assessment_planv1.AssessmentPlan{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("ap-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Assessment Plan", framework),
			Version: snapshotName,
		},
		ImportSsp: &assessment_planv1.ImportSsp{
			Href: &commonv1.URIReference{Value: fmt.Sprintf("ssp-%s-%s", framework, snapshotName)},
		},
		ReviewedControls: &assessment_planv1.ReviewedControls{
			ControlSelections: []*assessment_planv1.ControlSelection{sel},
		},
		Tasks:      tasks,
		BackMatter: buildBackMatter(framework, snapshotName),
	}
}

// GeneratePOAM builds an OSCAL Plan of Action and Milestones from a snapshot.
