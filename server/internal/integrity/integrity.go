package integrity

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// Severity indicates the seriousness of an integrity finding.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
)

// Finding represents a semantic integrity finding beyond JSON schema validation.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Message  string   `json:"message"`
}

var (
	uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[45][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	paramInsert = regexp.MustCompile(`\{\{\s*insert:\s*param,\s*([^}\s]+)\s*\}\}`)
)

// Check inspects an OSCAL document and returns all semantic integrity findings.
func Check(data []byte, kind schemavalidate.ArtifactKind) ([]Finding, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	var findings []Finding

	// Universal checks: UUID format, duplicate IDs, fragment links
	resourceUUIDs := make(map[string]bool)
	collectResourceUUIDs(root, resourceUUIDs)

	checkUUIDsAndFragments(root, "", resourceUUIDs, &findings)

	// Kind-specific semantic checks
	switch kind.RootKey {
	case "system-security-plan":
		if ssp, ok := root["system-security-plan"].(map[string]interface{}); ok {
			checkSSP(ssp, &findings)
		}
	case "component-definition":
		if compDef, ok := root["component-definition"].(map[string]interface{}); ok {
			checkComponentDefinition(compDef, &findings)
		}
	case "catalog":
		if cat, ok := root["catalog"].(map[string]interface{}); ok {
			checkCatalog(cat, &findings)
		}
	}

	return findings, nil
}

func collectResourceUUIDs(root map[string]interface{}, out map[string]bool) {
	for _, top := range root {
		topMap, ok := top.(map[string]interface{})
		if !ok {
			continue
		}
		bm, ok := topMap["back-matter"].(map[string]interface{})
		if !ok {
			continue
		}
		resList, ok := bm["resources"].([]interface{})
		if !ok {
			continue
		}
		for _, r := range resList {
			rMap, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			if u, ok := rMap["uuid"].(string); ok && u != "" {
				out[strings.ToLower(u)] = true
			}
		}
	}
}

func checkUUIDsAndFragments(node interface{}, currentPath string, resourceUUIDs map[string]bool, findings *[]Finding) {
	switch v := node.(type) {
	case map[string]interface{}:
		for k, val := range v {
			subPath := currentPath + "/" + k
			if k == "uuid" {
				if s, ok := val.(string); ok {
					if !uuidRegex.MatchString(s) {
						*findings = append(*findings, Finding{
							Rule:     "INT-UUID-RFC4122",
							Severity: SeverityError,
							Path:     subPath,
							Message:  fmt.Sprintf("UUID '%s' is not RFC 4122 compliant (must be UUIDv4 or UUIDv5)", s),
						})
					}
				}
			} else if k == "href" {
				if s, ok := val.(string); ok && strings.HasPrefix(s, "#") {
					target := strings.TrimPrefix(s, "#")
					targetNorm := strings.ToLower(target)
					if uuidRegex.MatchString(target) && len(resourceUUIDs) > 0 && !resourceUUIDs[targetNorm] {
						*findings = append(*findings, Finding{
							Rule:     "INT-FRAGMENT-REF-NOT-FOUND",
							Severity: SeverityWarning,
							Path:     subPath,
							Message:  fmt.Sprintf("fragment reference '%s' does not match any resource in back-matter", s),
						})
					}
				}
			} else if k == "prose" || k == "description" {
				if s, ok := val.(string); ok {
					matches := paramInsert.FindAllStringSubmatch(s, -1)
					for _, m := range matches {
						if len(m) > 1 && strings.TrimSpace(m[1]) == "" {
							*findings = append(*findings, Finding{
								Rule:     "INT-PARAM-EMPTY",
								Severity: SeverityError,
								Path:     subPath,
								Message:  "empty parameter placeholder in insert statement",
							})
						}
					}
				}
			}
			checkUUIDsAndFragments(val, subPath, resourceUUIDs, findings)
		}
	case []interface{}:
		for i, item := range v {
			checkUUIDsAndFragments(item, fmt.Sprintf("%s/%d", currentPath, i), resourceUUIDs, findings)
		}
	}
}

func checkSSP(ssp map[string]interface{}, findings *[]Finding) {
	// Collect defined component UUIDs
	definedComponents := make(map[string]bool)
	if sysImpl, ok := ssp["system-implementation"].(map[string]interface{}); ok {
		if comps, ok := sysImpl["components"].([]interface{}); ok {
			for _, c := range comps {
				if cMap, ok := c.(map[string]interface{}); ok {
					if u, ok := cMap["uuid"].(string); ok && u != "" {
						definedComponents[strings.ToLower(u)] = true
					}
				}
			}
		}
	}

	// Verify implemented-requirements
	if ctrlImpl, ok := ssp["control-implementation"].(map[string]interface{}); ok {
		if reqs, ok := ctrlImpl["implemented-requirements"].([]interface{}); ok {
			seenControls := make(map[string]int)
			for i, r := range reqs {
				rMap, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				ctrlID, _ := rMap["control-id"].(string)
				rPath := fmt.Sprintf("/system-security-plan/control-implementation/implemented-requirements/%d", i)

				if ctrlID != "" {
					seenControls[ctrlID]++
					if seenControls[ctrlID] > 1 {
						*findings = append(*findings, Finding{
							Rule:     "INT-DUPLICATE-CONTROL-ID",
							Severity: SeverityWarning,
							Path:     rPath + "/control-id",
							Message:  fmt.Sprintf("duplicate implemented-requirement for control '%s'", ctrlID),
						})
					}
				}

				// Check by-components references
				byComps, _ := rMap["by-components"].([]interface{})
				desc, _ := rMap["description"].(string)
				remarks, _ := rMap["remarks"].(string)

				if len(byComps) == 0 && desc == "" && remarks == "" {
					*findings = append(*findings, Finding{
						Rule:     "INT-HOLLOW-IMPLEMENTATION",
						Severity: SeverityWarning,
						Path:     rPath,
						Message:  fmt.Sprintf("control '%s' has empty implementation (no description, remarks, or by-components)", ctrlID),
					})
				}

				for j, bc := range byComps {
					bcMap, ok := bc.(map[string]interface{})
					if !ok {
						continue
					}
					bcPath := fmt.Sprintf("%s/by-components/%d", rPath, j)
					compUUID, _ := bcMap["component-uuid"].(string)
					if compUUID != "" && !definedComponents[strings.ToLower(compUUID)] {
						*findings = append(*findings, Finding{
							Rule:     "INT-SSP-ORPHAN-COMPONENT-REF",
							Severity: SeverityError,
							Path:     bcPath + "/component-uuid",
							Message:  fmt.Sprintf("referenced component-uuid '%s' does not exist in system-implementation.components", compUUID),
						})
					}
				}
			}
		}
	}
}

func checkComponentDefinition(compDef map[string]interface{}, findings *[]Finding) {
	comps, ok := compDef["components"].([]interface{})
	if !ok {
		return
	}

	seenCompUUIDs := make(map[string]bool)
	for i, c := range comps {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		cPath := fmt.Sprintf("/component-definition/components/%d", i)
		cu, _ := cMap["uuid"].(string)
		if cu != "" {
			norm := strings.ToLower(cu)
			if seenCompUUIDs[norm] {
				*findings = append(*findings, Finding{
					Rule:     "INT-DUPLICATE-COMPONENT-UUID",
					Severity: SeverityError,
					Path:     cPath + "/uuid",
					Message:  fmt.Sprintf("duplicate component uuid '%s'", cu),
				})
			}
			seenCompUUIDs[norm] = true
		}

		// Check implemented-requirements
		if ctrlImpls, ok := cMap["control-implementations"].([]interface{}); ok {
			for j, ci := range ctrlImpls {
				ciMap, ok := ci.(map[string]interface{})
				if !ok {
					continue
				}
				ciPath := fmt.Sprintf("%s/control-implementations/%d", cPath, j)
				if reqs, ok := ciMap["implemented-requirements"].([]interface{}); ok {
					seenControls := make(map[string]bool)
					for k, req := range reqs {
						reqMap, ok := req.(map[string]interface{})
						if !ok {
							continue
						}
						reqPath := fmt.Sprintf("%s/implemented-requirements/%d", ciPath, k)
						ctrlID, _ := reqMap["control-id"].(string)
						if ctrlID != "" {
							if seenControls[ctrlID] {
								*findings = append(*findings, Finding{
									Rule:     "INT-DUPLICATE-CONTROL-ID",
									Severity: SeverityWarning,
									Path:     reqPath + "/control-id",
									Message:  fmt.Sprintf("duplicate implemented-requirement for control '%s' in component '%s'", ctrlID, cu),
								})
							}
							seenControls[ctrlID] = true
						}
					}
				}
			}
		}
	}
}

func checkCatalog(cat map[string]interface{}, findings *[]Finding) {
	seenControlIDs := make(map[string]bool)
	var inspectControls func(ctrls []interface{}, p string)
	inspectControls = func(ctrls []interface{}, p string) {
		for i, c := range ctrls {
			cMap, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			cPath := fmt.Sprintf("%s/controls/%d", p, i)
			cid, _ := cMap["id"].(string)
			if cid != "" {
				if seenControlIDs[cid] {
					*findings = append(*findings, Finding{
						Rule:     "INT-CATALOG-DUPLICATE-CONTROL-ID",
						Severity: SeverityError,
						Path:     cPath + "/id",
						Message:  fmt.Sprintf("duplicate control id '%s' in catalog", cid),
					})
				}
				seenControlIDs[cid] = true
			}
			if subCtrls, ok := cMap["controls"].([]interface{}); ok {
				inspectControls(subCtrls, cPath)
			}
		}
	}

	if groups, ok := cat["groups"].([]interface{}); ok {
		for i, g := range groups {
			if gMap, ok := g.(map[string]interface{}); ok {
				gPath := fmt.Sprintf("/catalog/groups/%d", i)
				if ctrls, ok := gMap["controls"].([]interface{}); ok {
					inspectControls(ctrls, gPath)
				}
			}
		}
	}
	if ctrls, ok := cat["controls"].([]interface{}); ok {
		inspectControls(ctrls, "/catalog")
	}
}
