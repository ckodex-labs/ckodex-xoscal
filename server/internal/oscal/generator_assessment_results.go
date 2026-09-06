package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (g *Generator) GenerateAssessmentResults(ctx context.Context, snapshotName, framework string) (*assessment_resultsv1.AssessmentResults, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var findings []*assessment_resultsv1.Finding
	var observations []*assessment_resultsv1.Observation
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
		observations = append(observations, &assessment_resultsv1.Observation{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("obs-%s", req.Citation)),
			Description: &commonv1.MarkupMultiline{Value: req.Text},
			Methods:     []string{"INTERVIEW"},
			Collected:   &commonv1.DateTime{Value: timestamppb.Now()},
			Subjects: []*assessment_resultsv1.SubjectReference{{
				SubjectUuid: newUUID(uuidNamespace, fmt.Sprintf("ctrl-%s", req.Citation)),
				Type:        "component",
				Title:       &commonv1.MarkupLine{Value: req.Title},
			}},
		})
		if !req.Assessable {
			continue
		}
		findings = append(findings, &assessment_resultsv1.Finding{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("finding-%s", req.Citation)),
			Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Finding for %s", req.Title)},
			Description: &commonv1.MarkupMultiline{Value: req.Text},
			Target: &assessment_resultsv1.FindingTarget{
				Type:     "objective-id",
				TargetId: &commonv1.Token{Value: sanitizeToken(req.Citation)},
				Title:    &commonv1.MarkupLine{Value: req.Title},
				Status:   &assessment_resultsv1.ObjectiveStatus{State: "not-satisfied"},
			},
		})
	}

	return &assessment_resultsv1.AssessmentResults{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("ar-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Assessment Results", framework),
			Version: snapshotName,
		},
		ImportAp: &assessment_resultsv1.ImportAp{
			Href: &commonv1.URIReference{Value: fmt.Sprintf("ap-%s-%s", framework, snapshotName)},
		},
		Results: []*assessment_resultsv1.Result{{
			Uuid:         newUUID(uuidNamespace, fmt.Sprintf("result-%s-%s", framework, snapshotName)),
			Title:        &commonv1.MarkupLine{Value: fmt.Sprintf("Assessment of %s", framework)},
			Description:  &commonv1.MarkupMultiline{Value: fmt.Sprintf("Assessment results for %s framework", framework)},
			Start:        &commonv1.DateTime{Value: timestamppb.Now()},
			Findings:     findings,
			Observations: observations,
			ReviewedControls: &assessment_resultsv1.ReviewedControls{
				ControlSelections: []*assessment_resultsv1.ControlSelection{{
					Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("All controls in %s framework", framework)},
					IncludeAll:  &assessment_resultsv1.IncludeAll{},
				}},
			},
		}},
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// extractGuidance pulls sentences that start with guidance keywords.
