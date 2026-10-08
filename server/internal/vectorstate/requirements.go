package vectorstate

import "github.com/mchorfa/xoscal/server/internal/derogation"

func evaluateRequirements(target map[string]interface{}, key string, state *VectorState, evidence *evidenceSource, store *derogation.Store) {
	switch key {
	case "component-definition":
		components, _ := target["components"].([]interface{})
		for _, raw := range components {
			component, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			title := textValue(component["title"])
			if title == "" {
				title = "Component"
			}
			implementations, _ := component["control-implementations"].([]interface{})
			for _, raw := range implementations {
				implementation, _ := raw.(map[string]interface{})
				evaluateImplemented(implementation, title, false, state, evidence, store)
			}
		}
	case "system-security-plan":
		implementation, _ := target["control-implementation"].(map[string]interface{})
		evaluateImplemented(implementation, "System Security Plan", true, state, evidence, store)
	case "catalog":
		evalCatalogControls(target, "Catalog Root", state, evidence, store)
	}
}

func evaluateImplemented(implementation map[string]interface{}, title string, useRemarks bool, state *VectorState, evidence *evidenceSource, store *derogation.Store) {
	requirements, _ := implementation["implemented-requirements"].([]interface{})
	for _, raw := range requirements {
		requirement, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := requirement["control-id"].(string)
		description, _ := requirement["description"].(string)
		if useRemarks && description == "" {
			description, _ = requirement["remarks"].(string)
		}
		evaluateControl(id, title, description, state, evidence, store)
	}
}

func evalCatalogControls(target map[string]interface{}, title string, state *VectorState, evidence *evidenceSource, store *derogation.Store) {
	controls, _ := target["controls"].([]interface{})
	for _, raw := range controls {
		control, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id := textValue(control["id"])
		if id == "" {
			continue
		}
		evaluateControl(id, title, catalogDescription(control), state, evidence, store)
	}
	groups, _ := target["groups"].([]interface{})
	for _, raw := range groups {
		group, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := group["title"].(string)
		if title == "" {
			title = "Catalog Group"
		}
		evalCatalogControls(group, title, state, evidence, store)
	}
}

func catalogDescription(control map[string]interface{}) string {
	parts, _ := control["parts"].([]interface{})
	for _, raw := range parts {
		part, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if prose, ok := part["prose"].(string); ok && prose != "" {
			return prose
		}
		if list, ok := part["prose"].([]interface{}); ok && len(list) > 0 {
			if prose, ok := list[0].(string); ok && prose != "" {
				return prose
			}
		}
	}
	return textValue(control["title"])
}

func textValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	if object, ok := value.(map[string]interface{}); ok {
		text, _ := object["value"].(string)
		return text
	}
	return ""
}
