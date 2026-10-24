package tabular

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

type controlUpdate struct {
	description string
	remarks     string
}

// ImportFromCSV merges edited CSV rows back into a base OSCAL JSON artifact and
// guarantees the resulting output passes OSCAL schema validation.
func ImportFromCSV(baseArtifactJSON []byte, csvData []byte) ([]byte, error) {
	r := csv.NewReader(bytes.NewReader(csvData))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must contain at least a header row and one data row")
	}

	headers := records[0]
	ctrlIDIdx := -1
	descIdx := -1
	remarksIdx := -1

	for i, h := range headers {
		norm := strings.ToLower(strings.TrimSpace(h))
		norm = strings.ReplaceAll(norm, "_", " ")
		switch norm {
		case "control id", "control", "id":
			ctrlIDIdx = i
		case "implementation response", "implementation", "description", "prose":
			descIdx = i
		case "remarks", "notes", "comment":
			remarksIdx = i
		}
	}

	if ctrlIDIdx == -1 {
		return nil, fmt.Errorf("missing 'Control ID' column in CSV")
	}
	if descIdx == -1 {
		return nil, fmt.Errorf("missing 'Implementation Response' column in CSV")
	}

	updates := make(map[string]controlUpdate)
	for _, row := range records[1:] {
		if len(row) <= ctrlIDIdx || len(row) <= descIdx {
			continue
		}
		ctrlID := strings.TrimSpace(row[ctrlIDIdx])
		if ctrlID == "" {
			continue
		}
		desc := strings.TrimSpace(row[descIdx])
		remarks := ""
		if remarksIdx != -1 && len(row) > remarksIdx {
			remarks = strings.TrimSpace(row[remarksIdx])
		}
		updates[ctrlID] = controlUpdate{description: desc, remarks: remarks}
	}

	var root map[string]interface{}
	if err := json.Unmarshal(baseArtifactJSON, &root); err != nil {
		return nil, fmt.Errorf("unmarshal base OSCAL JSON: %w", err)
	}

	var targetKind schemavalidate.ArtifactKind

	if compDef, ok := root["component-definition"].(map[string]interface{}); ok {
		targetKind = schemavalidate.KindComponentDefinition
		comps, _ := compDef["components"].([]interface{})
		for _, rawComp := range comps {
			compMap, ok := rawComp.(map[string]interface{})
			if !ok {
				continue
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
					if u, exists := updates[ctrlID]; exists {
						reqMap["description"] = u.description
						if u.remarks != "" {
							reqMap["remarks"] = u.remarks
						}
					}
				}
			}
		}
	} else if ssp, ok := root["system-security-plan"].(map[string]interface{}); ok {
		targetKind = schemavalidate.KindSSP
		if ctrlImpl, ok := ssp["control-implementation"].(map[string]interface{}); ok {
			reqs, _ := ctrlImpl["implemented-requirements"].([]interface{})
			for _, rawReq := range reqs {
				reqMap, ok := rawReq.(map[string]interface{})
				if !ok {
					continue
				}
				ctrlID, _ := reqMap["control-id"].(string)
				if u, exists := updates[ctrlID]; exists {
					if u.remarks != "" {
						reqMap["remarks"] = u.remarks
					}
					if byComps, ok := reqMap["by-components"].([]interface{}); ok && len(byComps) > 0 {
						if bcMap, ok := byComps[0].(map[string]interface{}); ok {
							bcMap["description"] = u.description
						}
					}
				}
			}
		}
	} else {
		return nil, fmt.Errorf("base artifact must be an OSCAL component-definition or system-security-plan")
	}

	updatedJSON, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal updated JSON: %w", err)
	}

	// Schema Validation Gate
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}
	if err := v.Validate(updatedJSON, targetKind); err != nil {
		return nil, fmt.Errorf("updated OSCAL artifact failed schema validation: %w", err)
	}

	return updatedJSON, nil
}
