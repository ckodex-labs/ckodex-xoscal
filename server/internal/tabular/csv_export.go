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
	} else {
		return nil, fmt.Errorf("unsupported artifact type for CSV export: expected component-definition")
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
