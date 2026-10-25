package frameworks

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var projectUUIDNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func makeProjectUUID(parts ...string) string {
	val := strings.Join(parts, "/")
	return uuid.NewSHA1(projectUUIDNamespace, []byte(val)).String()
}

// ProjectedLayers encapsulates all 7 layers of OSCAL artifacts for a framework.
type ProjectedLayers struct {
	Catalog             []byte `json:"-"`
	Profile             []byte `json:"-"`
	ComponentDefinition []byte `json:"-"`
	SystemSecurityPlan  []byte `json:"-"`
	AssessmentPlan      []byte `json:"-"`
	AssessmentResults   []byte `json:"-"`
	POAM                []byte `json:"-"`
}

// ProjectAllLayers generates complete, schema-valid OSCAL 1.2.3 JSON templates
// for all layers of a given framework.
func ProjectAllLayers(f Framework) (*ProjectedLayers, error) {
	ctrls, err := LoadControls(f)
	if err != nil {
		return nil, fmt.Errorf("load controls for %s: %w", f.ID(), err)
	}

	catData, err := f.CatalogJSON()
	if err != nil {
		return nil, fmt.Errorf("get catalog for %s: %w", f.ID(), err)
	}

	var profData []byte
	if f.HasProfile() {
		profData, err = f.ProfileJSON()
		if err != nil {
			return nil, fmt.Errorf("get profile for %s: %w", f.ID(), err)
		}
	} else {
		profData, err = ProjectProfile(f, ctrls)
		if err != nil {
			return nil, fmt.Errorf("project profile for %s: %w", f.ID(), err)
		}
	}

	cdData, err := ProjectComponentDefinition(f, ctrls)
	if err != nil {
		return nil, fmt.Errorf("project component-definition for %s: %w", f.ID(), err)
	}

	sspData, err := ProjectSystemSecurityPlan(f, ctrls)
	if err != nil {
		return nil, fmt.Errorf("project system-security-plan for %s: %w", f.ID(), err)
	}

	apData, err := ProjectAssessmentPlan(f, ctrls)
	if err != nil {
		return nil, fmt.Errorf("project assessment-plan for %s: %w", f.ID(), err)
	}

	arData, err := ProjectAssessmentResults(f, ctrls)
	if err != nil {
		return nil, fmt.Errorf("project assessment-results for %s: %w", f.ID(), err)
	}

	poamData, err := ProjectPOAM(f, ctrls)
	if err != nil {
		return nil, fmt.Errorf("project poam for %s: %w", f.ID(), err)
	}

	return &ProjectedLayers{
		Catalog:             catData,
		Profile:             profData,
		ComponentDefinition: cdData,
		SystemSecurityPlan:  sspData,
		AssessmentPlan:      apData,
		AssessmentResults:   arData,
		POAM:                poamData,
	}, nil
}

// ProjectProfile generates an OSCAL 1.2.3 Profile importing the framework's catalog.
func ProjectProfile(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var withIDs []string
	for _, c := range ctrls {
		withIDs = append(withIDs, c.ID)
	}
	sort.Strings(withIDs)

	profDoc := map[string]interface{}{
		"profile": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "profile"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s Security Baseline Profile", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"imports": []map[string]interface{}{
				{
					"href": "./catalog.json",
					"include-controls": []map[string]interface{}{
						{
							"with-ids": withIDs,
						},
					},
				},
			},
		},
	}

	return json.MarshalIndent(profDoc, "", "  ")
}

// ProjectComponentDefinition generates an OSCAL 1.2.3 Component Definition.
func ProjectComponentDefinition(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	compUUID := makeProjectUUID(f.ID(), "component", "core-platform")
	implUUID := makeProjectUUID(f.ID(), "component-impl", "baseline")

	var implReqs []map[string]interface{}
	for _, c := range ctrls {
		req := map[string]interface{}{
			"uuid":        makeProjectUUID(f.ID(), "component-impl-req", c.ID),
			"control-id":  c.ID,
			"description": fmt.Sprintf("Component capability addressing control %s (%s).", c.ID, c.Title),
		}
		implReqs = append(implReqs, req)
	}

	cdDoc := map[string]interface{}{
		"component-definition": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "component-definition"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s Component Definition Template", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"components": []map[string]interface{}{
				{
					"uuid":        compUUID,
					"type":        "software",
					"title":       fmt.Sprintf("%s Platform Component", f.Title()),
					"description": fmt.Sprintf("Enterprise technical and operational capabilities implementing controls for %s.", f.Title()),
					"control-implementations": []map[string]interface{}{
						{
							"uuid":                     implUUID,
							"source":                   "./catalog.json",
							"description":              fmt.Sprintf("Security control implementations for %s.", f.Title()),
							"implemented-requirements": implReqs,
						},
					},
				},
			},
		},
	}

	return json.MarshalIndent(cdDoc, "", "  ")
}

// ProjectSystemSecurityPlan generates an OSCAL 1.2.3 System Security Plan.
func ProjectSystemSecurityPlan(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	userUUID := makeProjectUUID(f.ID(), "user", "sec-admin")
	compUUID := makeProjectUUID(f.ID(), "ssp-component", "core-system")

	importHref := "./catalog.json"
	if f.HasProfile() {
		importHref = "./profile.json"
	}

	var implReqs []map[string]interface{}
	for _, c := range ctrls {
		byCompUUID := makeProjectUUID(f.ID(), "by-comp", c.ID)
		irUUID := makeProjectUUID(f.ID(), "ssp-impl-req", c.ID)

		desc := fmt.Sprintf("Control %s (%s) is satisfied by the %s Core System Component through automated policy enforcement, configuration controls, and administrative review.", c.ID, c.Title, f.Title())

		ir := map[string]interface{}{
			"uuid":       irUUID,
			"control-id": c.ID,
			"remarks":    desc,
			"by-components": []map[string]interface{}{
				{
					"component-uuid": compUUID,
					"uuid":           byCompUUID,
					"description":    desc,
					"implementation-status": map[string]interface{}{
						"state": "implemented",
					},
				},
			},
		}

		// Include parameter settings if parameters exist
		if len(c.Parameters) > 0 {
			var setParams []map[string]interface{}
			var paramKeys []string
			for p := range c.Parameters {
				paramKeys = append(paramKeys, p)
			}
			sort.Strings(paramKeys)
			for _, pk := range paramKeys {
				val := c.Parameters[pk]
				if val == "" {
					val = "organization-defined-default"
				}
				setParams = append(setParams, map[string]interface{}{
					"param-id": pk,
					"values":   []string{val},
				})
			}
			ir["set-parameters"] = setParams
		}

		implReqs = append(implReqs, ir)
	}

	sspDoc := map[string]interface{}{
		"system-security-plan": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "system-security-plan"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s System Security Plan Template", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"import-profile": map[string]interface{}{
				"href": importHref,
			},
			"system-characteristics": map[string]interface{}{
				"system-ids": []map[string]interface{}{
					{"id": f.ID() + "-system"},
				},
				"system-name":       fmt.Sprintf("%s Reference System", f.Title()),
				"system-name-short": f.ID(),
				"description":       fmt.Sprintf("Reference system architecture implementing security controls for %s.", f.Title()),
				"system-information": map[string]interface{}{
					"information-types": []map[string]interface{}{
						{
							"title":       "Sovereign Program Data",
							"description": fmt.Sprintf("Operational and compliance data processed within the %s jurisdiction boundary.", f.Jurisdiction()),
						},
					},
				},
				"security-sensitivity-level": "moderate",
				"security-impact-level": map[string]interface{}{
					"security-objective-confidentiality": "moderate",
					"security-objective-integrity":       "high",
					"security-objective-availability":    "moderate",
				},
				"status": map[string]interface{}{
					"state": "operational",
				},
				"authorization-boundary": map[string]interface{}{
					"description": fmt.Sprintf("The authorization boundary encompasses all computing resources, cloud tenancies, and network zones supporting %s operations.", f.Title()),
				},
			},
			"system-implementation": map[string]interface{}{
				"users": []map[string]interface{}{
					{
						"uuid":     userUUID,
						"title":    "Security Administrator",
						"role-ids": []string{"sec-admin"},
					},
				},
				"components": []map[string]interface{}{
					{
						"uuid":        compUUID,
						"type":        "software",
						"title":       fmt.Sprintf("%s Core System Component", f.Title()),
						"description": fmt.Sprintf("Central execution platform providing control implementations for %s.", f.Title()),
						"status": map[string]interface{}{
							"state": "operational",
						},
					},
				},
			},
			"control-implementation": map[string]interface{}{
				"description":              fmt.Sprintf("Control implementations for %s.", f.Title()),
				"implemented-requirements": implReqs,
			},
		},
	}

	return json.MarshalIndent(sspDoc, "", "  ")
}

// ProjectAssessmentPlan generates an OSCAL 1.2.3 Assessment Plan.
func ProjectAssessmentPlan(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Group tasks by control family
	famMap := make(map[string][]ControlInfo)
	for _, c := range ctrls {
		fam := c.FamilyID
		if fam == "" {
			fam = "GENERAL"
		}
		famMap[fam] = append(famMap[fam], c)
	}

	var famKeys []string
	for k := range famMap {
		famKeys = append(famKeys, k)
	}
	sort.Strings(famKeys)

	var tasks []map[string]interface{}
	for _, fam := range famKeys {
		famCtrls := famMap[fam]
		famName := famCtrls[0].FamilyTitle
		if famName == "" {
			famName = fam
		}
		taskUUID := makeProjectUUID(f.ID(), "ap-task", fam)
		tasks = append(tasks, map[string]interface{}{
			"uuid":        taskUUID,
			"type":        "examination",
			"title":       fmt.Sprintf("Assess Family %s (%s)", fam, famName),
			"description": fmt.Sprintf("Verify and examine implementation of %d controls in the %s family for %s.", len(famCtrls), famName, f.Title()),
		})
	}

	apDoc := map[string]interface{}{
		"assessment-plan": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "assessment-plan"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s Security Assessment Plan Template", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"import-ssp": map[string]interface{}{
				"href": "./system-security-plan.json",
			},
			"reviewed-controls": map[string]interface{}{
				"control-selections": []map[string]interface{}{
					{
						"description": fmt.Sprintf("All controls in %s", f.Title()),
						"include-all": map[string]interface{}{},
					},
				},
			},
			"tasks": tasks,
		},
	}

	return json.MarshalIndent(apDoc, "", "  ")
}

// ProjectAssessmentResults generates an OSCAL 1.2.3 Assessment Results document.
func ProjectAssessmentResults(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	resUUID := makeProjectUUID(f.ID(), "ar-result", "baseline")

	var observations []map[string]interface{}
	var findings []map[string]interface{}

	// Sample representative observations & findings across families
	famSeen := make(map[string]bool)
	for _, c := range ctrls {
		if !famSeen[c.FamilyID] {
			famSeen[c.FamilyID] = true
			obsUUID := makeProjectUUID(f.ID(), "obs", c.ID)
			findingUUID := makeProjectUUID(f.ID(), "finding", c.ID)

			observations = append(observations, map[string]interface{}{
				"uuid":        obsUUID,
				"description": fmt.Sprintf("Verified control %s (%s) implementation and supporting evidence.", c.ID, c.Title),
				"methods":     []string{"EXAMINE", "TEST"},
				"types":       []string{"finding"},
				"collected":   now,
			})

			findings = append(findings, map[string]interface{}{
				"uuid":        findingUUID,
				"title":       fmt.Sprintf("Assessment Finding for %s", c.ID),
				"description": fmt.Sprintf("Control %s has been evaluated and confirmed satisfied.", c.ID),
				"target": map[string]interface{}{
					"type":      "objective-id",
					"target-id": c.ID,
					"status": map[string]interface{}{
						"state": "satisfied",
					},
				},
				"related-observations": []map[string]interface{}{
					{"observation-uuid": obsUUID},
				},
			})
		}
	}

	arDoc := map[string]interface{}{
		"assessment-results": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "assessment-results"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s Assessment Results Template", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"import-ap": map[string]interface{}{
				"href": "./assessment-plan.json",
			},
			"results": []map[string]interface{}{
				{
					"uuid":        resUUID,
					"title":       fmt.Sprintf("%s Assessment Results", f.Title()),
					"description": fmt.Sprintf("Assessment evaluation findings and observations for %s.", f.Title()),
					"start":       now,
					"reviewed-controls": map[string]interface{}{
						"control-selections": []map[string]interface{}{
							{
								"description": fmt.Sprintf("All controls evaluated for %s", f.Title()),
								"include-all": map[string]interface{}{},
							},
						},
					},
					"observations": observations,
					"findings":     findings,
				},
			},
		},
	}

	return json.MarshalIndent(arDoc, "", "  ")
}

// ProjectPOAM generates an OSCAL 1.2.3 Plan of Action and Milestones.
func ProjectPOAM(f Framework, ctrls []ControlInfo) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	sampleTitle := "Initial Baseline Tailoring and Continuous Monitoring"
	if len(ctrls) > 0 {
		sampleTitle = fmt.Sprintf("Compliance Roadmap and Parameter Tailoring for %s", f.Title())
	}

	poamDoc := map[string]interface{}{
		"plan-of-action-and-milestones": map[string]interface{}{
			"uuid": makeProjectUUID(f.ID(), "poam"),
			"metadata": map[string]interface{}{
				"title":         fmt.Sprintf("%s Plan of Action and Milestones Template", f.Title()),
				"last-modified": now,
				"version":       f.Version(),
				"oscal-version": "1.2.3",
			},
			"import-ssp": map[string]interface{}{
				"href": "./system-security-plan.json",
			},
			"poam-items": []map[string]interface{}{
				{
					"uuid":        makeProjectUUID(f.ID(), "poam-item", "1"),
					"title":       sampleTitle,
					"description": fmt.Sprintf("Continuous monitoring, parameter assignment reviews, and periodic assessment milestones for %s.", f.Title()),
				},
			},
		},
	}

	return json.MarshalIndent(poamDoc, "", "  ")
}
