package oscal

import (
	"context"
	"fmt"
	"strings"

	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
)

func (g *Generator) GenerateAssessmentResults(ctx context.Context, snapshotName, framework string) (*assessment_resultsv1.AssessmentResults, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generateAssessmentResultsFromSnapshot(snap, snapshotName, framework), nil
}

func (g *Generator) generateAssessmentResultsFromSnapshot(snap *parsedSnapshot, snapshotName, framework string) *assessment_resultsv1.AssessmentResults {
	ts := deterministicProtoTimestamp()
	const maxSampleResults = 25
	sampleCap := len(snap.assessableReqs)
	if sampleCap > maxSampleResults {
		sampleCap = maxSampleResults
	}
	findings := make([]*assessment_resultsv1.Finding, 0, sampleCap)
	observations := make([]*assessment_resultsv1.Observation, 0, sampleCap)
	includeControls := make([]*assessment_resultsv1.SelectedControl, 0, len(snap.assessableReqs))
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
				includeControls = append(includeControls, &assessment_resultsv1.SelectedControl{
					ControlId: &commonv1.Token{Value: ctrlID},
				})
			}
		}

		if len(findings) < maxSampleResults {
			title := singleLine(req.Title)
			if title == "" {
				title = singleLine(req.Citation)
			}
			if title == "" {
				title = singleLine(req.NodeRefID)
			}
			desc := fmt.Sprintf("Assessment evaluation observation and finding for %s (%s).", ctrlID, title)
			observations = append(observations, &assessment_resultsv1.Observation{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("obs-%s", req.Citation)),
				Description: &commonv1.MarkupMultiline{Value: desc},
				Methods:     []string{"INTERVIEW"},
				Collected:   &commonv1.DateTime{Value: ts},
				Subjects: []*assessment_resultsv1.SubjectReference{{
					SubjectUuid: newUUID(uuidNamespace, fmt.Sprintf("ctrl-%s", req.Citation)),
					Type:        "component",
					Title:       &commonv1.MarkupLine{Value: title},
				}},
			})
			targetID := ctrlID
			if targetID == "" {
				targetID = "_control"
			}
			findings = append(findings, &assessment_resultsv1.Finding{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("finding-%s", req.Citation)),
				Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Finding for %s", title)},
				Description: &commonv1.MarkupMultiline{Value: desc},
				Target: &assessment_resultsv1.FindingTarget{
					Type:     "objective-id",
					TargetId: &commonv1.Token{Value: targetID},
					Title:    &commonv1.MarkupLine{Value: title},
					Status:   &assessment_resultsv1.ObjectiveStatus{State: "not-satisfied"},
				},
			})
		}
	}

	sel := &assessment_resultsv1.ControlSelection{
		Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("All controls in %s framework", framework)},
	}
	if len(includeControls) > 0 {
		sel.IncludeControls = includeControls
	} else {
		sel.IncludeAll = &assessment_resultsv1.IncludeAll{}
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
			Start:        &commonv1.DateTime{Value: ts},
			Findings:     findings,
			Observations: observations,
			ReviewedControls: &assessment_resultsv1.ReviewedControls{
				ControlSelections: []*assessment_resultsv1.ControlSelection{sel},
			},
		}},
		BackMatter: buildBackMatter(framework, snapshotName),
	}
}

// extractGuidance pulls sentences that start with guidance keywords.
