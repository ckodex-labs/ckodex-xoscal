package oscal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (g *Generator) GenerateSSP(ctx context.Context, snapshotName string, framework string) (*sspv1.SystemSecurityPlan, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var components []*sspv1.SystemComponent
	var implementedReqs []*sspv1.ImplementedRequirement
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
		desc := req.Text
		if desc == "" {
			desc = title
		}
		compUUID := newUUID(uuidNamespace, fmt.Sprintf("component-%s", ctrlID))
		components = append(components, &sspv1.SystemComponent{
			Uuid:        compUUID,
			Type:        "software",
			Title:       title,
			Description: desc,
			Status:      &sspv1.Status{State: "operational"},
		})
		byComp := &sspv1.ByComponent{
			ComponentUuid: compUUID,
			Uuid:          newUUID(uuidNamespace, fmt.Sprintf("bycomp-%s", ctrlID)),
			Description:   &commonv1.MarkupMultiline{Value: desc},
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
		stmtByComp := &sspv1.ByComponent{
			ComponentUuid: compUUID,
			Uuid:          newUUID(uuidNamespace, fmt.Sprintf("stmt-bycomp-%s", ctrlID)),
			Description:   byComp.Description,
		}
		implReq := &sspv1.ImplementedRequirement{
			Uuid:         newUUID(uuidNamespace, fmt.Sprintf("impl-%s", ctrlID)),
			ControlId:    &commonv1.Token{Value: ctrlID},
			Props:        sspProps,
			ByComponents: []*sspv1.ByComponent{byComp},
			Statements: []*sspv1.Statement{{
				Uuid:         newUUID(uuidNamespace, fmt.Sprintf("stmt-%s", ctrlID)),
				StatementId:  &commonv1.Token{Value: ctrlID + "-stmt"},
				ByComponents: []*sspv1.ByComponent{stmtByComp},
			}},
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

	if len(components) == 0 {
		defaultCompUUID := newUUID(uuidNamespace, fmt.Sprintf("component-default-%s", framework))
		components = append(components, &sspv1.SystemComponent{
			Uuid:        defaultCompUUID,
			Type:        "software",
			Title:       fmt.Sprintf("%s Default System Component", framework),
			Description: fmt.Sprintf("Default governed component for %s", framework),
			Status:      &sspv1.Status{State: "operational"},
		})
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
			Title:   fmt.Sprintf("%s System Security Plan", framework),
			Version: snapshotName,
			Roles: []*commonv1.Role{{
				Id:    &commonv1.Token{Value: "sec-admin"},
				Title: "Security Administrator",
			}},
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
	}

	// Derive responsible parties from unique roles present in requirements.
	roleSeen := map[string]bool{"sec-admin": true}
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
		if req.Role != "" && !roleSeen[req.Role] {
			roleSeen[req.Role] = true
			roleTok := sanitizeToken(req.Role)
			partyUUID := newUUID(uuidNamespace, fmt.Sprintf("party-%s-%s", framework, req.Role))
			ssp.Metadata.Roles = append(ssp.Metadata.Roles, &commonv1.Role{
				Id:    &commonv1.Token{Value: roleTok},
				Title: singleLine(req.Role),
			})
			ssp.Metadata.Parties = append(ssp.Metadata.Parties, &commonv1.Party{
				Uuid: partyUUID,
				Type: "organization",
				Name: singleLine(req.Role),
			})
			ssp.Metadata.ResponsibleParties = append(ssp.Metadata.ResponsibleParties, &commonv1.ResponsibleParty{
				RoleId:     &commonv1.Token{Value: roleTok},
				PartyUuids: []*commonv1.UUID{partyUUID},
			})
		}
	}

	ssp.BackMatter = buildBackMatter(framework, snapshotName)
	return ssp, nil
}

// GenerateComponentDefinition builds an OSCAL Component Definition from a snapshot.
