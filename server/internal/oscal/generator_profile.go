package oscal

import (
	"context"
	"fmt"
	"strings"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func hasImplementationGroup(req kg.Requirement, group string) bool {
	for _, g := range req.ImplementationGroups {
		if strings.EqualFold(g, group) {
			return true
		}
	}
	return false
}

func (g *Generator) GenerateProfile(ctx context.Context, snapshotName string, framework string, selected []string) (*profilev1.Profile, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generateProfileFromSnapshot(snap, snapshotName, framework, selected), nil
}

func (g *Generator) generateProfileFromSnapshot(snap *parsedSnapshot, snapshotName string, framework string, selected []string) *profilev1.Profile {
	allControls := make([]string, 0, len(snap.assessableReqs))
	var profileSelectedControls []string
	for _, req := range snap.assessableReqs {
		ctrlID := sanitizeToken(req.Citation)
		if ctrlID == "" {
			ctrlID = sanitizeToken(req.NodeRefID)
		}
		if ctrlID == "" {
			continue
		}
		allControls = append(allControls, ctrlID)
		if hasImplementationGroup(req, "profile-selected") {
			profileSelectedControls = append(profileSelectedControls, ctrlID)
		}
	}

	if len(selected) == 0 {
		if len(profileSelectedControls) > 0 {
			selected = profileSelectedControls
		} else {
			selected = allControls
		}
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[strings.ToLower(sanitizeToken(s))] = true
	}

	var alterations []*profilev1.Alter
	var setParams []*profilev1.SetParameters
	seenParamIDs := make(map[string]bool)

	for _, req := range snap.assessableReqs {
		ctrlID := sanitizeToken(req.Citation)
		if ctrlID == "" {
			ctrlID = sanitizeToken(req.NodeRefID)
		}
		if ctrlID == "" || !selectedSet[strings.ToLower(ctrlID)] {
			continue
		}

		props := requirementToProps(req)
		if len(props) > 0 {
			var addParts []*catalogv1.Part
			seenPartIDs := make(map[string]int)
			for _, p := range props {
				pNs := p.Ns
				if pNs == "" {
					pNs = xoscalPropNamespace
				}
				basePartID := sanitizeToken(ctrlID + "-" + p.Name)
				seenPartIDs[basePartID]++
				partID := basePartID
				if seenPartIDs[basePartID] > 1 {
					partID = fmt.Sprintf("%s-%d", basePartID, seenPartIDs[basePartID])
				}
				addParts = append(addParts, &catalogv1.Part{
					Id:    &commonv1.Token{Value: partID},
					Name:  sanitizeToken(p.Name),
					Ns:    pNs,
					Prose: []*commonv1.MarkupMultiline{{Value: p.Value}},
				})
			}
			guidance := req.Guidance
			if guidance == "" {
				guidance = extractGuidance(req.Text)
			}
			if guidance == "" {
				guidance = req.Text
			}
			if guidance == "" {
				guidance = req.Title
			}
			guidanceLine := singleLine(guidance)
			if guidanceLine == "" {
				guidanceLine = singleLine(ctrlID)
			}
			alter := &profilev1.Alter{
				ControlId: ctrlID,
				Adds: []*profilev1.Add{{
					Props: []*commonv1.Property{{
						Name:  "guidance",
						Value: guidanceLine,
						Ns:    xoscalPropNamespace,
					}},
					Parts: addParts,
				}},
			}
			alterations = append(alterations, alter)
		}

		for _, p := range req.Params {
			paramID := sanitizeToken(p.ID)
			if paramID == "" || seenParamIDs[paramID] {
				continue
			}
			seenParamIDs[paramID] = true
			lbl := singleLine(p.Label)
			if lbl == "" {
				lbl = fmt.Sprintf("Parameter %s for %s", paramID, singleLine(req.Title))
			}
			var vals []*commonv1.MarkupLine
			for _, v := range p.Values {
				if sv := singleLine(v); sv != "" {
					vals = append(vals, &commonv1.MarkupLine{Value: sv})
				}
			}
			if len(vals) == 0 {
				vals = []*commonv1.MarkupLine{{Value: lbl}}
			}
			setParams = append(setParams, &profilev1.SetParameters{
				WithIds: []string{ctrlID},
				Params: []*catalogv1.Parameter{{
					Id:     &commonv1.Token{Value: paramID},
					Label:  &commonv1.MarkupLine{Value: lbl},
					Values: vals,
				}},
			})
		}
		if req.Section != "" && len(req.Params) == 0 && req.NodeRefID == "" {
			paramID := ctrlID + "-param"
			if !seenParamIDs[paramID] {
				seenParamIDs[paramID] = true
				setParams = append(setParams, &profilev1.SetParameters{
					WithIds: []string{ctrlID},
					Params: []*catalogv1.Parameter{{
						Id: &commonv1.Token{Value: paramID},
						Label: &commonv1.MarkupLine{
							Value: fmt.Sprintf("Parameter for %s", singleLine(req.Title)),
						},
						Values: []*commonv1.MarkupLine{{Value: singleLine(req.Section)}},
					}},
				})
			}
		}
	}

	imp := &profilev1.Import{
		Href: &commonv1.URIReference{
			Value: fmt.Sprintf("catalog-%s-%s", framework, snapshotName),
		},
	}
	if len(selected) > 0 && len(selected) < len(allControls) {
		imp.Custom = &profilev1.Custom{
			WithIds: selected,
		}
	} else {
		imp.IncludeAll = &profilev1.IncludeAll{}
	}

	profile := &profilev1.Profile{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("profile-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Baseline Profile", framework),
			Version: snapshotName,
		},
		Imports: []*profilev1.Import{imp},
		Modify: &profilev1.Modify{
			SetParameters: setParams,
			Alters:        alterations,
		},
		BackMatter: buildBackMatter(framework, snapshotName),
	}

	return profile
}

// GenerateSSP builds an OSCAL System Security Plan from a snapshot.
