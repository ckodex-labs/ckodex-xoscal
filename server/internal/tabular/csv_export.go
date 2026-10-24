package tabular

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
)

// ExportToCSV extracts controls from an OSCAL JSON artifact and renders them as standard CSV.
func ExportToCSV(artifactJSON []byte) ([]byte, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(artifactJSON, &root); err != nil {
		return nil, fmt.Errorf("unmarshal OSCAL JSON: %w", err)
	}

	var rows []TabularControl

	// Component-Definition handler
	if compDef, ok := root["component-definition"].(map[string]interface{}); ok {
		comps, _ := compDef["components"].([]interface{})
		for _, rawComp := range comps {
			compMap, ok := rawComp.(map[string]interface{})
			if !ok {
				continue
			}
			compTitle := "Component"
			if t, ok := compMap["title"].(string); ok && t != "" {
				compTitle = t
			} else if tMap, ok := compMap["title"].(map[string]interface{}); ok {
				if s, ok := tMap["value"].(string); ok {
					compTitle = s
				}
			}

			ctrlImpls, _ := compMap["control-implementations"].([]interface{})
			for _, rawImpl := range ctrlImpls {
				implMap, ok := rawImpl.(map[string]interface{})
				if !ok {
					continue
				}
				reqs, _ := implMap["implemented-requirements"].([]interface{})
				for _, rawReq := range reqs {
					reqMap, ok := rawReq.(map[string]interface{})
					if !ok {
						continue
					}
					ctrlID, _ := reqMap["control-id"].(string)
					desc, _ := reqMap["description"].(string)
					remarks, _ := reqMap["remarks"].(string)

					rows = append(rows, TabularControl{
						ControlID:           ctrlID,
						Component:           compTitle,
						Status:              "implemented",
						ImplementationProse: desc,
						ResponsibleRole:     "Security Engineering",
						Remarks:             remarks,
					})
				}
			}
		}
	} else if ssp, ok := root["system-security-plan"].(map[string]interface{}); ok {
		sysTitle := "System Security Plan"
		if sysChar, ok := ssp["system-characteristics"].(map[string]interface{}); ok {
			if name, ok := sysChar["system-name"].(string); ok && name != "" {
				sysTitle = name
			}
		}

		if ctrlImpl, ok := ssp["control-implementation"].(map[string]interface{}); ok {
			reqs, _ := ctrlImpl["implemented-requirements"].([]interface{})
			for _, rawReq := range reqs {
				reqMap, ok := rawReq.(map[string]interface{})
				if !ok {
					continue
				}
				ctrlID, _ := reqMap["control-id"].(string)
				desc, _ := reqMap["description"].(string)
				remarks, _ := reqMap["remarks"].(string)

				if desc == "" {
					if byComps, ok := reqMap["by-components"].([]interface{}); ok && len(byComps) > 0 {
						if bcMap, ok := byComps[0].(map[string]interface{}); ok {
							if d, ok := bcMap["description"].(string); ok {
								desc = d
							}
						}
					}
				}

				rows = append(rows, TabularControl{
					ControlID:           ctrlID,
					Component:           sysTitle,
					Status:              "implemented",
					ImplementationProse: desc,
					ResponsibleRole:     "System Owner",
					Remarks:             remarks,
				})
			}
		}
	} else if cat, ok := root["catalog"].(map[string]interface{}); ok {
		var extractControls func(ctrls []interface{}, groupTitle string)
		extractControls = func(ctrls []interface{}, groupTitle string) {
			for _, raw := range ctrls {
				cMap, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				cid, _ := cMap["id"].(string)
				title, _ := cMap["title"].(string)
				prose := ""
				if parts, ok := cMap["parts"].([]interface{}); ok {
					for _, p := range parts {
						pMap, ok := p.(map[string]interface{})
						if !ok {
							continue
						}
						if name, _ := pMap["name"].(string); name == "statement" || name == "guidance" {
							if pr, ok := pMap["prose"].(string); ok && pr != "" {
								if prose != "" {
									prose += "\n"
								}
								prose += pr
							}
						}
					}
				}
				rows = append(rows, TabularControl{
					ControlID:           cid,
					Component:           groupTitle,
					Status:              "catalog",
					ImplementationProse: prose,
					ResponsibleRole:     "Standards Authority",
					Remarks:             title,
				})
				if subCtrls, ok := cMap["controls"].([]interface{}); ok {
					extractControls(subCtrls, groupTitle)
				}
			}
		}

		if groups, ok := cat["groups"].([]interface{}); ok {
			for _, rawG := range groups {
				gMap, ok := rawG.(map[string]interface{})
				if !ok {
					continue
				}
				gTitle, _ := gMap["title"].(string)
				if gTitle == "" {
					gTitle = "Control Group"
				}
				if ctrls, ok := gMap["controls"].([]interface{}); ok {
					extractControls(ctrls, gTitle)
				}
			}
		}
		if ctrls, ok := cat["controls"].([]interface{}); ok {
			extractControls(ctrls, "Catalog")
		}
	} else {
		return nil, fmt.Errorf("unsupported artifact type for CSV export: expected component-definition, system-security-plan, or catalog")
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Write Header
	headers := []string{"Control ID", "Component", "Status", "Implementation Response", "Responsible Role", "Remarks"}
	if err := w.Write(headers); err != nil {
		return nil, err
	}

	for _, r := range rows {
		record := []string{
			r.ControlID,
			r.Component,
			r.Status,
			r.ImplementationProse,
			r.ResponsibleRole,
			r.Remarks,
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
