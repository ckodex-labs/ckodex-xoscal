package oscal

import (
	"context"
	"fmt"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
)

func (g *Generator) GenerateSSP(ctx context.Context, snapshotName string, framework string) (*sspv1.SystemSecurityPlan, error) {
	snap, err := g.loadParsedSnapshot(ctx, snapshotName, framework, false)
	if err != nil {
		return nil, err
	}
	return g.generateSSPFromSnapshot(snap, snapshotName, framework), nil
}

func (g *Generator) generateSSPFromSnapshot(snap *parsedSnapshot, snapshotName string, framework string) *sspv1.SystemSecurityPlan {
	compUUID := newUUID(uuidNamespace, fmt.Sprintf("component-default-%s", framework))
	components := []*sspv1.SystemComponent{{
		Uuid:        compUUID,
		Type:        "software",
		Title:       fmt.Sprintf("%s Default System Component", framework),
		Description: fmt.Sprintf("Default governed component for %s", framework),
		Status:      &sspv1.Status{State: "operational"},
	}}
	implementedReqs := make([]*sspv1.ImplementedRequirement, 0, len(snap.assessableReqs))
	roles := []*commonv1.Role{{
		Id:    &commonv1.Token{Value: "sec-admin"},
		Title: "Security Administrator",
	}}
	var parties []*commonv1.Party
	var responsibleParties []*commonv1.ResponsibleParty
	roleSeen := map[string]bool{"sec-admin": true}

	for _, req := range snap.assessableReqs {
		if req.Role != "" && !roleSeen[req.Role] {
			roleSeen[req.Role] = true
			roleTok := sanitizeToken(req.Role)
			partyUUID := newUUID(uuidNamespace, fmt.Sprintf("party-%s-%s", framework, req.Role))
			roles = append(roles, &commonv1.Role{
				Id:    &commonv1.Token{Value: roleTok},
				Title: singleLine(req.Role),
			})
			parties = append(parties, &commonv1.Party{
				Uuid: partyUUID,
				Type: "organization",
				Name: singleLine(req.Role),
			})
			responsibleParties = append(responsibleParties, &commonv1.ResponsibleParty{
				RoleId:     &commonv1.Token{Value: roleTok},
				PartyUuids: []*commonv1.UUID{partyUUID},
			})
		}

		ctrlID := sanitizeToken(req.Citation)
		if ctrlID == "" {
			ctrlID = sanitizeToken(req.NodeRefID)
		}
		if ctrlID == "" {
			continue
		}
		title := singleLine(req.Title)
		if title == "" {
			title = singleLine(ctrlID)
		}
		byComp := &sspv1.ByComponent{
			ComponentUuid: compUUID,
			Uuid:          newUUID(uuidNamespace, fmt.Sprintf("bycomp-%s", ctrlID)),
			Description:   &commonv1.MarkupMultiline{Value: fmt.Sprintf("Baseline implementation narrative for %s (%s).", ctrlID, title)},
		}
		var sspProps []*commonv1.Property
		for _, p := range requirementToProps(req) {
			if p == nil {
				continue
			}
			ns := p.Ns
			if ns == "" && p.Name != "control-origination" && p.Name != "marking" {
				ns = xoscalPropNamespace
			}
			sspProps = append(sspProps, &commonv1.Property{
				Name:  p.Name,
				Value: p.Value,
				Ns:    ns,
				Class: p.Class,
			})
		}
		implReq := &sspv1.ImplementedRequirement{
			Uuid:         newUUID(uuidNamespace, fmt.Sprintf("impl-%s", ctrlID)),
			ControlId:    &commonv1.Token{Value: ctrlID},
			Props:        sspProps,
			ByComponents: []*sspv1.ByComponent{byComp},
		}
		seenParamIDs := make(map[string]bool)
		for _, p := range req.Params {
			paramID := sanitizeToken(p.ID)
			if paramID == "" || seenParamIDs[paramID] {
				continue
			}
			seenParamIDs[paramID] = true
			var vals []*commonv1.MarkupLine
			for _, v := range p.Values {
				if sv := singleLine(v); sv != "" {
					vals = append(vals, &commonv1.MarkupLine{Value: sv})
				}
			}
			if len(vals) == 0 {
				lbl := singleLine(p.Label)
				if lbl == "" {
					lbl = "[Assignment: organization-defined value]"
				}
				vals = []*commonv1.MarkupLine{{Value: lbl}}
			}
			implReq.SetParameters = append(implReq.SetParameters, &sspv1.SetParameter{
				ParamId: &commonv1.Token{Value: paramID},
				Values:  vals,
			})
		}
		// Add set-parameters for requirements with sections when no explicit params exist.
		if req.Section != "" && len(req.Params) == 0 && req.NodeRefID == "" {
			implReq.SetParameters = append(implReq.SetParameters, &sspv1.SetParameter{
				ParamId: &commonv1.Token{Value: ctrlID + "-param"},
				Values:  []*commonv1.MarkupLine{{Value: singleLine(req.Section)}},
			})
		}
		implementedReqs = append(implementedReqs, implReq)
	}

	if len(implementedReqs) == 0 {
		implementedReqs = append(implementedReqs, &sspv1.ImplementedRequirement{
			Uuid:      newUUID(uuidNamespace, fmt.Sprintf("impl-default-%s", framework)),
			ControlId: &commonv1.Token{Value: "baseline-control"},
			ByComponents: []*sspv1.ByComponent{{
				ComponentUuid: components[0].Uuid,
				Uuid:          newUUID(uuidNamespace, fmt.Sprintf("bycomp-default-%s", framework)),
				Description:   &commonv1.MarkupMultiline{Value: fmt.Sprintf("Default control implementation for %s", framework)},
			}},
		})
	}

	ssp := &sspv1.SystemSecurityPlan{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("ssp-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:              fmt.Sprintf("%s System Security Plan", framework),
			Version:            snapshotName,
			Roles:              roles,
			Parties:            parties,
			ResponsibleParties: responsibleParties,
		},
		ImportProfile: &sspv1.ImportProfile{
			Href: &commonv1.URIReference{Value: fmt.Sprintf("profile-%s-%s", framework, snapshotName)},
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds:   []*sspv1.SystemId{{Id: framework + "-system"}},
			SystemName:  fmt.Sprintf("%s AI System", framework),
			Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("System security plan for %s framework", framework)},
			SystemInformation: &sspv1.SystemInformation{
				InformationTypes: []*sspv1.InformationType{{
					Uuid:                  newUUID(uuidNamespace, fmt.Sprintf("infotype-%s", framework)),
					Title:                 &commonv1.MarkupLine{Value: fmt.Sprintf("%s Compliance Information", framework)},
					Description:           &commonv1.MarkupMultiline{Value: fmt.Sprintf("Information processed under %s governance.", framework)},
					ConfidentialityImpact: &sspv1.Impact{Base: "fips-199-moderate"},
					IntegrityImpact:       &sspv1.Impact{Base: "fips-199-moderate"},
					AvailabilityImpact:    &sspv1.Impact{Base: "fips-199-moderate"},
				}},
			},
			SecurityImpactLevel: &sspv1.SecurityImpactLevel{
				SecurityObjectiveConfidentiality: "fips-199-moderate",
				SecurityObjectiveIntegrity:       "fips-199-moderate",
				SecurityObjectiveAvailability:    "fips-199-moderate",
			},
			Status: &sspv1.Status{State: "operational"},
			AuthorizationBoundary: &sspv1.AuthorizationBoundary{
				Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("Authorization boundary for %s governed system.", framework)},
			},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Users: []*sspv1.SystemUser{{
				Uuid:  newUUID(uuidNamespace, fmt.Sprintf("user-admin-%s", framework)),
				Title: "System Administrator",
				Props: []*commonv1.Property{{Name: "type", Value: "internal"}},
			}},
			Components: components,
			InventoryItems: []*sspv1.InventoryItem{{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("inventory-%s", framework)),
				Description: &commonv1.MarkupLine{Value: fmt.Sprintf("AI system inventory for %s", framework)},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			Description:             &commonv1.MarkupMultiline{Value: fmt.Sprintf("Control implementation for %s.", framework)},
			ImplementedRequirements: implementedReqs,
		},
		BackMatter: buildBackMatter(framework, snapshotName),
	}

	return ssp
}

// GenerateComponentDefinition builds an OSCAL Component Definition from a snapshot.
