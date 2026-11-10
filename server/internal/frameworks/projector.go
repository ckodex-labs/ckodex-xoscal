package frameworks

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mchorfa/xoscal/server/internal/oscal"
)

const (
	fallbackOSCALTimestamp = "2026-01-01T00:00:00Z"
	xoscalPropNS           = "https://ckodex.io/ns/xoscal/props"
)

var projectUUIDNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func makeProjectUUID(parts ...string) string {
	val := strings.Join(parts, "/")
	return uuid.NewSHA1(projectUUIDNamespace, []byte(val)).String()
}

// deterministicTimestamp extracts a valid RFC3339 timestamp from catalog.metadata.last-modified
// (or catalog.metadata.published), falling back to "2026-01-01T00:00:00Z" so all 7 projected
// OSCAL layers are byte-for-byte reproducible across exports.
func deterministicTimestamp(catData []byte) string {
	if len(catData) > 0 {
		var doc struct {
			Catalog struct {
				Metadata struct {
					LastModified string `json:"last-modified"`
					Published    string `json:"published"`
				} `json:"metadata"`
			} `json:"catalog"`
		}
		if err := json.Unmarshal(catData, &doc); err == nil {
			for _, cand := range []string{doc.Catalog.Metadata.LastModified, doc.Catalog.Metadata.Published} {
				cand = strings.TrimSpace(cand)
				if cand == "" {
					continue
				}
				if _, err := time.Parse(time.RFC3339, cand); err == nil {
					return cand
				}
				if _, err := time.Parse(time.RFC3339Nano, cand); err == nil {
					return cand
				}
			}
		}
	}
	return fallbackOSCALTimestamp
}

func frameworkTimestamp(f Framework) string {
	if f == nil {
		return fallbackOSCALTimestamp
	}
	catData, err := f.CatalogJSON()
	if err != nil {
		return fallbackOSCALTimestamp
	}
	return deterministicTimestamp(catData)
}

func sortedUniqueControlIDs(ctrls []ControlInfo) []string {
	seen := make(map[string]bool, len(ctrls))
	ids := make([]string, 0, len(ctrls))
	for _, c := range ctrls {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if !seen[key] {
			seen[key] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func sortedUniqueControls(ctrls []ControlInfo) []ControlInfo {
	seen := make(map[string]bool, len(ctrls))
	out := make([]ControlInfo, 0, len(ctrls))
	for _, c := range ctrls {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i].ID), strings.ToLower(out[j].ID)
		if li != lj {
			return li < lj
		}
		return out[i].ID < out[j].ID
	})
	return out
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
	ts := deterministicTimestamp(catData)

	var profData []byte
	if f.HasProfile() {
		profData, err = f.ProfileJSON()
		if err != nil {
			return nil, fmt.Errorf("get profile for %s: %w", f.ID(), err)
		}
	} else {
		profData, err = projectProfileWithTimestamp(f, ctrls, ts)
		if err != nil {
			return nil, fmt.Errorf("project profile for %s: %w", f.ID(), err)
		}
	}

	cdData, err := projectComponentDefinitionWithTimestamp(f, ctrls, ts)
	if err != nil {
		return nil, fmt.Errorf("project component-definition for %s: %w", f.ID(), err)
	}

	sspData, err := projectSystemSecurityPlanWithTimestamp(f, ctrls, ts)
	if err != nil {
		return nil, fmt.Errorf("project system-security-plan for %s: %w", f.ID(), err)
	}

	apData, err := projectAssessmentPlanWithTimestamp(f, ctrls, ts)
	if err != nil {
		return nil, fmt.Errorf("project assessment-plan for %s: %w", f.ID(), err)
	}

	arData, err := projectAssessmentResultsWithTimestamp(f, ctrls, ts)
	if err != nil {
		return nil, fmt.Errorf("project assessment-results for %s: %w", f.ID(), err)
	}

	poamData, err := projectPOAMWithTimestamp(f, ctrls, ts)
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
	return projectProfileWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectProfileWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	withIDs := sortedUniqueControlIDs(ctrls)

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
	return projectComponentDefinitionWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectComponentDefinitionWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	compUUID := makeProjectUUID(f.ID(), "component", "core-platform")
	implUUID := makeProjectUUID(f.ID(), "component-impl", "baseline")

	var implReqs []map[string]interface{}
	for _, c := range sortedUniqueControls(ctrls) {
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
	return projectSystemSecurityPlanWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectSystemSecurityPlanWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	userUUID := makeProjectUUID(f.ID(), "user", "sec-admin")
	compUUID := makeProjectUUID(f.ID(), "ssp-component", "core-system")

	importHref := "./catalog.json"
	if f.HasProfile() {
		importHref = "./profile.json"
	}

	var implReqs []map[string]interface{}
	for _, c := range sortedUniqueControls(ctrls) {
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
				val := norm(c.Parameters[pk])
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
				"roles": []map[string]interface{}{
					{
						"id":    "sec-admin",
						"title": "Security Administrator",
					},
				},
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
							"uuid":        makeProjectUUID(f.ID(), "info-type", "sovereign-program-data"),
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
	return projectAssessmentPlanWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectAssessmentPlanWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	// Group tasks by control family
	famMap := make(map[string][]ControlInfo)
	for _, c := range sortedUniqueControls(ctrls) {
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
	if len(tasks) == 0 {
		tasks = []map[string]interface{}{
			{
				"uuid":        makeProjectUUID(f.ID(), "ap-task", "baseline"),
				"type":        "examination",
				"title":       fmt.Sprintf("Assess Baseline Controls (%s)", f.Title()),
				"description": fmt.Sprintf("Verify and examine baseline control implementations for %s.", f.Title()),
			},
		}
	}

	sortedIDs := sortedUniqueControlIDs(ctrls)
	selection := map[string]interface{}{
		"description": fmt.Sprintf("All controls in %s", f.Title()),
	}
	if len(sortedIDs) > 0 {
		incControls := make([]map[string]interface{}, 0, len(sortedIDs))
		for _, cid := range sortedIDs {
			incControls = append(incControls, map[string]interface{}{
				"control-id": cid,
			})
		}
		selection["include-controls"] = incControls
	} else {
		selection["include-all"] = map[string]interface{}{}
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
				"control-selections": []map[string]interface{}{selection},
			},
			"tasks": tasks,
		},
	}

	return json.MarshalIndent(apDoc, "", "  ")
}

// ProjectAssessmentResults generates an OSCAL 1.2.3 Assessment Results document.
func ProjectAssessmentResults(f Framework, ctrls []ControlInfo) ([]byte, error) {
	return projectAssessmentResultsWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectAssessmentResultsWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	resUUID := makeProjectUUID(f.ID(), "ar-result", "baseline")

	var observations []map[string]interface{}
	var findings []map[string]interface{}

	// Sample representative observations & findings across families in deterministic order
	famSeen := make(map[string]bool)
	for _, c := range sortedUniqueControls(ctrls) {
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

	sortedIDs := sortedUniqueControlIDs(ctrls)
	selection := map[string]interface{}{
		"description": fmt.Sprintf("All controls evaluated for %s", f.Title()),
	}
	if len(sortedIDs) > 0 {
		incControls := make([]map[string]interface{}, 0, len(sortedIDs))
		for _, cid := range sortedIDs {
			incControls = append(incControls, map[string]interface{}{
				"control-id": cid,
			})
		}
		selection["include-controls"] = incControls
	} else {
		selection["include-all"] = map[string]interface{}{}
	}

	resultEntry := map[string]interface{}{
		"uuid":        resUUID,
		"title":       fmt.Sprintf("%s Assessment Results", f.Title()),
		"description": fmt.Sprintf("Assessment evaluation findings and observations for %s.", f.Title()),
		"start":       now,
		"reviewed-controls": map[string]interface{}{
			"control-selections": []map[string]interface{}{selection},
		},
	}
	if len(observations) > 0 {
		resultEntry["observations"] = observations
	}
	if len(findings) > 0 {
		resultEntry["findings"] = findings
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
			"results": []map[string]interface{}{resultEntry},
		},
	}

	return json.MarshalIndent(arDoc, "", "  ")
}

// ProjectPOAM generates an OSCAL 1.2.3 Plan of Action and Milestones.
func ProjectPOAM(f Framework, ctrls []ControlInfo) ([]byte, error) {
	return projectPOAMWithTimestamp(f, ctrls, frameworkTimestamp(f))
}

func projectPOAMWithTimestamp(f Framework, ctrls []ControlInfo, now string) ([]byte, error) {
	sortedCtrls := sortedUniqueControls(ctrls)
	var poamItems []map[string]interface{}
	if len(sortedCtrls) > 0 {
		poamItems = make([]map[string]interface{}, 0, len(sortedCtrls))
		for _, c := range sortedCtrls {
			title := strings.TrimSpace(c.Title)
			if title == "" {
				title = c.ID
			}
			poamItems = append(poamItems, map[string]interface{}{
				"uuid":        makeProjectUUID(f.ID(), "poam-item", c.ID),
				"title":       fmt.Sprintf("Continuous Monitoring & Tailoring: %s (%s)", c.ID, title),
				"description": fmt.Sprintf("Continuous monitoring, parameter assignment reviews, and periodic assessment milestones for control %s in %s.", c.ID, f.Title()),
				"props": []map[string]interface{}{
					{
						"name":  "control-id",
						"ns":    xoscalPropNS,
						"value": c.ID,
					},
				},
			})
		}
	} else {
		poamItems = []map[string]interface{}{
			{
				"uuid":        makeProjectUUID(f.ID(), "poam-item", "1"),
				"title":       "Initial Baseline Tailoring and Continuous Monitoring",
				"description": fmt.Sprintf("Continuous monitoring, parameter assignment reviews, and periodic assessment milestones for %s.", f.Title()),
			},
		}
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
			"poam-items": poamItems,
		},
	}

	return json.MarshalIndent(poamDoc, "", "  ")
}

// GenerateAllLayersFromKG seeds the framework into an in-memory Knowledge Graph store
// and synthesizes all 7 OSCAL 1.2.3 layers via a single snapshot pass in oscal.Generator.
func GenerateAllLayersFromKG(ctx context.Context, f Framework) (*ProjectedLayers, error) {
	store := newMemKGStore()
	defer func() { _ = store.Close() }()

	if err := SeedFramework(ctx, store, f); err != nil {
		return nil, fmt.Errorf("seed framework %s: %w", f.ID(), err)
	}
	snapName := f.Version()
	if snapName == "" {
		snapName = "v1"
	}
	if _, err := store.CreateSnapshot(ctx, snapName); err != nil {
		return nil, fmt.Errorf("create snapshot for %s: %w", f.ID(), err)
	}

	gen := oscal.NewGenerator(store)
	all, err := gen.GenerateAllArtifacts(ctx, snapName, f.ID(), nil)
	if err != nil {
		return nil, fmt.Errorf("generate all artifacts for %s: %w", f.ID(), err)
	}

	catBytes, err := oscal.ExportCatalogJSON(all.Catalog)
	if err != nil {
		return nil, fmt.Errorf("export catalog for %s: %w", f.ID(), err)
	}
	profBytes, err := oscal.ExportProfileJSON(all.Profile)
	if err != nil {
		return nil, fmt.Errorf("export profile for %s: %w", f.ID(), err)
	}
	cdBytes, err := oscal.ExportComponentDefinitionJSON(all.ComponentDefinition)
	if err != nil {
		return nil, fmt.Errorf("export component-definition for %s: %w", f.ID(), err)
	}
	sspBytes, err := oscal.ExportSSPJSON(all.SSP)
	if err != nil {
		return nil, fmt.Errorf("export ssp for %s: %w", f.ID(), err)
	}
	apBytes, err := oscal.ExportAssessmentPlanJSON(all.AssessmentPlan)
	if err != nil {
		return nil, fmt.Errorf("export assessment-plan for %s: %w", f.ID(), err)
	}
	arBytes, err := oscal.ExportAssessmentResultsJSON(all.AssessmentResults)
	if err != nil {
		return nil, fmt.Errorf("export assessment-results for %s: %w", f.ID(), err)
	}
	poamBytes, err := oscal.ExportPOAMJSON(all.POAM)
	if err != nil {
		return nil, fmt.Errorf("export poam for %s: %w", f.ID(), err)
	}

	return &ProjectedLayers{
		Catalog:             catBytes,
		Profile:             profBytes,
		ComponentDefinition: cdBytes,
		SystemSecurityPlan:  sspBytes,
		AssessmentPlan:      apBytes,
		AssessmentResults:   arBytes,
		POAM:                poamBytes,
	}, nil
}
