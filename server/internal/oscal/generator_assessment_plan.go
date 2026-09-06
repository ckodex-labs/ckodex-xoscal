package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateAssessmentPlan(ctx context.Context, snapshotName string, framework string) (*assessment_planv1.AssessmentPlan, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var tasks []*assessment_planv1.AssessmentTask
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
		tasks = append(tasks, &assessment_planv1.AssessmentTask{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("task-%s", req.Citation)),
			Type:        "examination",
			Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Assess %s", req.Title)},
			Description: &commonv1.MarkupMultiline{Value: req.Text},
		})
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
			ControlSelections: []*assessment_planv1.ControlSelection{{
				Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("Controls reviewed for %s", framework)},
				IncludeAll:  &assessment_planv1.IncludeAll{},
			}},
		},
		Tasks:      tasks,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GeneratePOAM builds an OSCAL Plan of Action and Milestones from a snapshot.
