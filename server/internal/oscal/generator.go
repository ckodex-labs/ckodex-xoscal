package oscal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Generator converts KG snapshots into OSCAL protobuf artifacts.
type Generator struct {
	store kg.Store
}

func NewGenerator(store kg.Store) *Generator {
	return &Generator{store: store}
}

// newUUID generates a deterministic v5 UUID from the given namespace and name.
func newUUID(ns uuid.UUID, name string) *commonv1.UUID {
	u := uuid.NewSHA1(ns, []byte(name))
	return &commonv1.UUID{Value: u.String()}
}

var uuidNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace

// GenerateCatalog builds an OSCAL Catalog from a snapshot of requirements.
// Supports hierarchical groups derived from parent_urn / depth relationships.
func (g *Generator) GenerateCatalog(ctx context.Context, snapshotName string, framework string) (*catalogv1.Catalog, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	type node struct {
		req      kg.Requirement
		children []string
	}
	byURN := make(map[string]*node)
	var roots []string

	// First pass: index all requirements.
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
		byURN[req.URN] = &node{req: req}
	}

	// Second pass: build parent-child relationships.
	for _, n := range byURN {
		if n.req.ParentURN != "" {
			if parent, ok := byURN[n.req.ParentURN]; ok {
				parent.children = append(parent.children, n.req.URN)
			} else {
				roots = append(roots, n.req.URN)
			}
		} else {
			roots = append(roots, n.req.URN)
		}
	}

	buildControl := func(req kg.Requirement) *catalogv1.Control {
		ctrl := &catalogv1.Control{
			Id:    &commonv1.Token{Value: req.Citation},
			Title: &commonv1.MarkupLine{Value: req.Title},
			Props: requirementToProps(req),
			Parts: []*catalogv1.Part{
				{Id: &commonv1.Token{Value: req.Citation + "-stmt"}, Name: "statement", Prose: []*commonv1.MarkupMultiline{{Value: req.Text}}},
			},
		}
		// Add parameters for requirements with sections/subsections.
		if req.Section != "" {
			ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
				Id: &commonv1.Token{Value: req.Citation + "-param"},
				Label: &commonv1.MarkupLine{
					Value: fmt.Sprintf("Parameter for %s", req.Title),
				},
			})
		}
		if req.Subsection != "" {
			ctrl.Params = append(ctrl.Params, &catalogv1.Parameter{
				Id: &commonv1.Token{Value: req.Citation + "-subparam"},
				Label: &commonv1.MarkupLine{
					Value: fmt.Sprintf("Sub-parameter for %s", req.Title),
				},
			})
		}
		// Add guidance part if text contains guidance-like sentences.
		if guidance := extractGuidance(req.Text); guidance != "" {
			ctrl.Parts = append(ctrl.Parts, &catalogv1.Part{
				Id:    &commonv1.Token{Value: req.Citation + "-guidance"},
				Name:  "guidance",
				Prose: []*commonv1.MarkupMultiline{{Value: guidance}},
			})
		}
		return ctrl
	}

	var buildGroup func(urn string) *catalogv1.Group
	buildGroup = func(urn string) *catalogv1.Group {
		n, ok := byURN[urn]
		if !ok {
			return nil
		}
		g := &catalogv1.Group{
			Id:    &commonv1.Token{Value: n.req.NodeRefID},
			Title: &commonv1.MarkupLine{Value: n.req.NodeName},
		}
		if n.req.Assessable {
			g.Controls = append(g.Controls, buildControl(n.req))
		}
		for _, childURN := range n.children {
			if childGroup := buildGroup(childURN); childGroup != nil {
				g.Groups = append(g.Groups, childGroup)
			}
		}
		return g
	}

	var groups []*catalogv1.Group
	var controls []*catalogv1.Control
	for _, urn := range roots {
		if n, ok := byURN[urn]; ok && n.req.Assessable {
			controls = append(controls, buildControl(n.req))
		} else {
			if g := buildGroup(urn); g != nil {
				groups = append(groups, g)
			}
		}
	}

	return &catalogv1.Catalog{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("catalog-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Control Catalog", framework),
			Version: snapshotName,
		},
		Controls:   controls,
		Groups:     groups,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GenerateProfile builds an OSCAL Profile from a snapshot with selected controls.
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
		allControls = append(allControls, req.Citation)
		props := requirementToProps(req)
		if len(props) > 0 {
			var addParts []*catalogv1.Part
			for _, p := range props {
				addParts = append(addParts, &catalogv1.Part{
					Id:    &commonv1.Token{Value: req.Citation + "-" + p.Name},
					Name:  p.Name,
					Prose: []*commonv1.MarkupMultiline{{Value: p.Value}},
				})
			}
			guidance := extractGuidance(req.Text)
			if guidance == "" {
				guidance = req.Text
			}
			alter := &profilev1.Alter{
				ControlId: req.Citation,
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
				WithIds: []string{req.Citation},
				Params: []*catalogv1.Parameter{{
					Id: &commonv1.Token{Value: req.Citation + "-param"},
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
			ControlId: &commonv1.Token{Value: req.Citation},
			Props:     requirementToProps(req),
			Statements: []*sspv1.Statement{{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("stmt-%s", req.Citation)),
				StatementId: &commonv1.Token{Value: req.Citation + "-stmt"},
			}},
		}
		// Add set-parameters for requirements with sections.
		if req.Section != "" {
			implReq.SetParameters = append(implReq.SetParameters, &sspv1.SetParameter{
				ParamId: &commonv1.Token{Value: req.Citation + "-param"},
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
func (g *Generator) GenerateComponentDefinition(ctx context.Context, snapshotName string, framework string) (*componentv1.ComponentDefinition, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var components []*componentv1.DefinedComponent
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
		var controlImplementations []*componentv1.ControlImplementation
		if req.Citation != "" {
			controlImplementations = append(controlImplementations, &componentv1.ControlImplementation{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("ctrlimpl-%s", req.Citation)),
				Source:      &commonv1.URIReference{Value: fmt.Sprintf("catalog-%s-%s", framework, snapshotName)},
				Description: req.Text,
				ImplementedRequirements: []*componentv1.ImplementedRequirement{{
					Uuid:        newUUID(uuidNamespace, fmt.Sprintf("implreq-%s", req.Citation)),
					ControlId:   req.Citation,
					Description: req.Text,
				}},
			})
		}
		components = append(components, &componentv1.DefinedComponent{
			Uuid:                   newUUID(uuidNamespace, fmt.Sprintf("comp-%s", req.Citation)),
			Type:                   "software",
			Title:                  &commonv1.MarkupLine{Value: req.Title},
			Description:            &commonv1.MarkupMultiline{Value: req.Text},
			ControlImplementations: controlImplementations,
		})
	}

	return &componentv1.ComponentDefinition{
		Uuid: newUUID(uuidNamespace, fmt.Sprintf("compdef-%s-%s", framework, snapshotName)),
		Metadata: &commonv1.Metadata{
			Title:   fmt.Sprintf("%s Component Definition", framework),
			Version: snapshotName,
		},
		BackMatter: buildBackMatter(framework, snapshotName),
		Components: components,
	}, nil
}

// GenerateMappings builds OSCAL Control Mapping entries from a snapshot.
func (g *Generator) GenerateMappings(ctx context.Context, snapshotName string) ([]*mappingv1.Map, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var maps []*mappingv1.Map
	for _, e := range entities {
		if e.Type != "reg:Mapping" {
			continue
		}
		var m kg.Mapping
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			continue
		}
		maps = append(maps, &mappingv1.Map{
			Uuid:         &commonv1.UUID{Value: m.URN},
			Relationship: &commonv1.Token{Value: m.Relationship},
			Sources: []*mappingv1.MappingItem{{
				IdRef: m.From,
			}},
			Targets: []*mappingv1.MappingItem{{
				IdRef: m.To,
			}},
		})
	}
	return maps, nil
}

// GenerateAssessmentPlan builds an OSCAL Assessment Plan from a snapshot.
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
			}},
		},
		Tasks:      tasks,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GeneratePOAM builds an OSCAL Plan of Action and Milestones from a snapshot.
func (g *Generator) GeneratePOAM(ctx context.Context, snapshotName string, framework string) (*poamv1.PlanOfActionAndMilestones, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var risks []*poamv1.Risk
	var poamItems []*poamv1.PoamItem
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
		risk := &poamv1.Risk{
			Uuid:        newUUID(uuidNamespace, fmt.Sprintf("risk-%s", req.Citation)),
			Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Risk: %s", req.Title)},
			Description: &commonv1.MarkupMultiline{Value: req.Text},
			Status:      &poamv1.RiskStatus{State: "open"},
		}
		// Build the POAM item (OSCAL 1.1.2 poam-item: title, description, refs).
		poamItem := &poamv1.PoamItem{
			Uuid:        risk.Uuid,
			Title:       risk.Title,
			Description: risk.Description,
		}
		// Add remediation task for high-risk requirements.
		if strings.ToLower(req.RiskLevel) == "high" {
			resp := &poamv1.Response{
				Uuid:        newUUID(uuidNamespace, fmt.Sprintf("remediate-%s", req.Citation)),
				Title:       &commonv1.MarkupLine{Value: fmt.Sprintf("Remediate %s", req.Title)},
				Description: &commonv1.MarkupMultiline{Value: req.Text},
				Status:      &poamv1.ResponseStatus{State: "open"},
			}
			if req.Role != "" {
				resp.ResponsibleRoles = append(resp.ResponsibleRoles, &poamv1.ResponsibleRole{
					RoleId: &commonv1.Token{Value: req.Role},
				})
			}
			resp.RelatedTasks = append(resp.RelatedTasks, &poamv1.RelatedTask{
				TaskUuid: newUUID(uuidNamespace, fmt.Sprintf("task-%s", req.Citation)),
				Title:    &commonv1.MarkupLine{Value: fmt.Sprintf("Implement controls for %s", req.Title)},
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
		},
		ImportSsp: &poamv1.ImportSsp{
			Href: &commonv1.URIReference{Value: fmt.Sprintf("ssp-%s-%s", framework, snapshotName)},
		},
		Risks:      risks,
		PoamItems:  poamItems,
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// GenerateAssessmentResults builds OSCAL Assessment Results from a snapshot.
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
				TargetId: &commonv1.Token{Value: req.Citation},
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
				}},
			},
		}},
		BackMatter: buildBackMatter(framework, snapshotName),
	}, nil
}

// extractGuidance pulls sentences that start with guidance keywords.
func extractGuidance(text string) string {
	if text == "" {
		return ""
	}
	// Simple heuristic: look for sentences starting with should/may/can.
	// A real implementation would use NLP; this is a lightweight data-driven fallback.
	var guidance []string
	// Naive sentence split on ". "
	sentences := strings.Split(text, ". ")
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "should ") || strings.HasPrefix(lower, "may ") || strings.HasPrefix(lower, "can ") || strings.HasPrefix(lower, "it is recommended") {
			guidance = append(guidance, trimmed)
		}
	}
	return strings.Join(guidance, ". ")
}

// requirementToProps extracts data-driven OSCAL properties from a requirement.
func requirementToProps(req kg.Requirement) []*commonv1.Property {
	var props []*commonv1.Property
	if req.Role != "" {
		props = append(props, &commonv1.Property{Name: "role", Value: req.Role})
	}
	if req.RiskLevel != "" {
		props = append(props, &commonv1.Property{Name: "risk-level", Value: req.RiskLevel})
	}
	if req.Lifecycle != "" {
		props = append(props, &commonv1.Property{Name: "lifecycle", Value: req.Lifecycle})
	}
	if len(req.ImplementationGroups) > 0 {
		props = append(props, &commonv1.Property{
			Name:  "implementation-groups",
			Value: fmt.Sprintf("%v", req.ImplementationGroups),
		})
	}
	return props
}

// buildBackMatter creates a BackMatter with a resource linking to the framework.
func buildBackMatter(framework, snapshotName string) *commonv1.BackMatter {
	return &commonv1.BackMatter{
		Resources: []*commonv1.Resource{{
			Uuid:  newUUID(uuidNamespace, fmt.Sprintf("resource-%s-%s", framework, snapshotName)),
			Title: fmt.Sprintf("%s Framework Reference", framework),
			Description: &commonv1.MarkupMultiline{
				Value: fmt.Sprintf("Source framework metadata for %s snapshot %s", framework, snapshotName),
			},
			Props: []*commonv1.Property{
				{Name: "framework", Value: framework},
				{Name: "snapshot", Value: snapshotName},
			},
		}},
	}
}

// GenerateCrossFrameworkMappings builds OSCAL Maps from cross-framework requirement mappings.
func (g *Generator) GenerateCrossFrameworkMappings(ctx context.Context, snapshotName, sourceFramework, targetFramework string) ([]*mappingv1.Map, error) {
	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	// Build URN -> Requirement lookup for resolving mapping references.
	reqByURN := make(map[string]kg.Requirement)
	for _, e := range entities {
		if e.Type != "reg:Requirement" {
			continue
		}
		var r kg.Requirement
		if err := json.Unmarshal(e.Payload, &r); err != nil {
			continue
		}
		reqByURN[e.URN] = r
	}

	var maps []*mappingv1.Map
	for _, e := range entities {
		if e.Type != "reg:RequirementMapping" {
			continue
		}
		var m kg.RequirementMapping
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			continue
		}
		// Filter by source/target framework when specified.
		srcReq, srcOk := reqByURN[m.SourceURN]
		tgtReq, tgtOk := reqByURN[m.TargetURN]
		if sourceFramework != "" && (!srcOk || srcReq.Framework != sourceFramework) {
			continue
		}
		if targetFramework != "" && (!tgtOk || tgtReq.Framework != targetFramework) {
			continue
		}

		// Resolve IdRef from framework:citation (fall back to raw URN if lookup fails).
		srcIdRef := m.SourceURN
		if srcOk && srcReq.Citation != "" {
			srcIdRef = fmt.Sprintf("%s:%s", srcReq.Framework, srcReq.Citation)
		}
		tgtIdRef := m.TargetURN
		if tgtOk && tgtReq.Citation != "" {
			tgtIdRef = fmt.Sprintf("%s:%s", tgtReq.Framework, tgtReq.Citation)
		}

		confidence := m.Strength
		if confidence == 0 {
			confidence = 0.5
		}

		maps = append(maps, &mappingv1.Map{
			Uuid:              &commonv1.UUID{Value: m.URN},
			MatchingRationale: m.Rationale,
			Relationship:      &commonv1.Token{Value: m.Relationship},
			Sources: []*mappingv1.MappingItem{{
				IdRef: srcIdRef,
			}},
			Targets: []*mappingv1.MappingItem{{
				IdRef: tgtIdRef,
			}},
			ConfidenceScore: &mappingv1.ConfidenceScore{Value: confidence},
		})
	}
	return maps, nil
}
