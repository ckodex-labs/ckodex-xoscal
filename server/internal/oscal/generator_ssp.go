package oscal

import (
	"context"
	"encoding/json"
	"fmt"

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
		if req.Framework != framework {
			continue
		}
		components = append(components, &sspv1.SystemComponent{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("component-%s", req.Citation)),
			Type:        "software",
			Title:       req.Title,
			Description: req.Text,
		})
		implReq := &sspv1.ImplementedRequirement{
			Uuid:      newUUID(uuidNamespace, fmt.Sprintf("impl-%s", req.Citation)),
			ControlId: &commonv1.Token{Value: sanitizeToken(req.Citation)},
			Props:     requirementToProps(req),
			Statements: []*sspv1.Statement{{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("stmt-%s", req.Citation)),
				StatementId: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-stmt"},
			}},
		}
		// Add set-parameters for requirements with sections.
		if req.Section != "" {
			implReq.SetParameters = append(implReq.SetParameters, &sspv1.SetParameter{
				ParamId: &commonv1.Token{Value: sanitizeToken(req.Citation) + "-param"},
				Value:   req.Section,
			})
		}
		implementedReqs = append(implementedReqs, implReq)
	}

	ssp := &sspv1.SystemSecurityPlan{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("ssp-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s System Security Plan", framework),
			Version: snapshotName,
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds:   []*sspv1.SystemId{{Id: framework + "-system"}},
			SystemName:  fmt.Sprintf("%s AI System", framework),
			Description: &commonv1.MarkupMultiline{Value: fmt.Sprintf("System security plan for %s framework", framework)},
			SecurityImpactLevel: &sspv1.SecurityImpactLevel{
				SecurityObjectiveConfidentiality: "moderate",
				SecurityObjectiveIntegrity:       "high",
				SecurityObjectiveAvailability:    "moderate",
			},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Components: components,
			InventoryItems: []*sspv1.InventoryItem{{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("inventory-%s", framework)),
				Description: &commonv1.MarkupLine{Value: fmt.Sprintf("AI system inventory for %s", framework)},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			ImplementedRequirements: implementedReqs,
		},
	}

	// Derive responsible parties from unique roles present in requirements.
	roleSeen := make(map[string]bool)
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
		if req.Role != "" && !roleSeen[req.Role] {
			roleSeen[req.Role] = true
			ssp.Metadata.ResponsibleParties = append(ssp.Metadata.ResponsibleParties, &commonv1.ResponsibleParty{
				RoleId: &commonv1.Token{Value: req.Role},
				PartyUuids: []*commonv1.UUID{
					newUUID(uuidNamespace, fmt.Sprintf("party-%s-%s", framework, req.Role)),
				},
			})
		}
	}

	ssp.BackMatter = buildBackMatter(framework, snapshotName)
	return ssp, nil
}

// GenerateComponentDefinition builds an OSCAL Component Definition from a snapshot.
