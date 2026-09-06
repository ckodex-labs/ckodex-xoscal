package oscal

import (
	"context"
	"encoding/json"
	"fmt"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateProfile(ctx context.Context, snapshotName string, framework string, selected []string) (*profilev1.Profile, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var allControls []string
	var alterations []*profilev1.Alter
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
		allControls = append(allControls, sanitizeToken(req.Citation))
		props := requirementToProps(req)
		if len(props) > 0 {
			var addParts []*catalogv1.Part
			for _, p := range props {
				addParts = append(addParts, &catalogv1.Part{
					Id:    &commonv1.Token{Value: sanitizeToken(req.Citation) + "-" + p.Name},
					Name:  p.Name,
					Prose: []*commonv1.MarkupMultiline{{Value: p.Value}},
				})
			}
			guidance := extractGuidance(req.Text)
			if guidance == "" {
				guidance = req.Text
			}
			alter := &profilev1.Alter{
				ControlId: sanitizeToken(req.Citation),
				Adds: []*profilev1.Add{{
					Props: []*commonv1.Property{{
						Name:  "guidance",
						Value: guidance,
					}},
					Parts: addParts,
				}},
			}
			alterations = append(alterations, alter)
		}
	}

	// If selected is empty, include all controls.
	if len(selected) == 0 {
		selected = allControls
	}

	var setParams []*profilev1.SetParameters
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
		if req.Section != "" {
			setParams = append(setParams, &profilev1.SetParameters{
				WithIds: []string{sanitizeToken(req.Citation)},
				Params: []*catalogv1.Parameter{{
					Id: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-param"},
					Label: &commonv1.MarkupLine{
						Value: fmt.Sprintf("Parameter for %s", req.Title),
					},
				}},
			})
		}
	}

	profile := &profilev1.Profile{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("profile-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Baseline Profile", framework),
			Version: snapshotName,
		},
		Imports: []*profilev1.Import{{
			Href: &commonv1.URIReference{
				Value: fmt.Sprintf("catalog-%s-%s", framework, snapshotName),
			},
			IncludeAll: &profilev1.IncludeAll{},
		}},
		Modify: &profilev1.Modify{
			SetParameters: setParams,
			Alters:        alterations,
		},
	}

	profile.BackMatter = buildBackMatter(framework, snapshotName)
	return profile, nil
}

// GenerateSSP builds an OSCAL System Security Plan from a snapshot.
