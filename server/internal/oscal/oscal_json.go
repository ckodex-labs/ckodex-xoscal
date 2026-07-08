package oscal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// OSCALVersion is the OSCAL version all artifacts conform to.
const OSCALVersion = "1.1.2"

// oscalRootKeys maps protobuf message types to their OSCAL JSON root wrapper key.
var oscalRootKeys = map[string]string{
	"Catalog":                   "catalog",
	"Profile":                   "profile",
	"SystemSecurityPlan":        "system-security-plan",
	"ComponentDefinition":       "component-definition",
	"AssessmentPlan":            "assessment-plan",
	"AssessmentResults":         "assessment-results",
	"PlanOfActionAndMilestones": "plan-of-action-and-milestones",
}

// marshalOSCALJSON serializes a protobuf message to OSCAL-compliant JSON.
// It transforms protojson output (camelCase, {"value":"..."} wrappers) into
// proper OSCAL JSON (kebab-case, plain strings, root model wrapper, required
// metadata fields).
func marshalOSCALJSON(msg proto.Message, rootKey string) ([]byte, error) {
	// Step 1: Marshal to protojson
	raw, err := protojson.MarshalOptions{Multiline: true, EmitUnpopulated: false}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("protojson marshal: %w", err)
	}

	// Step 2: Unmarshal to generic map
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("unmarshal protojson: %w", err)
	}

	// Step 3: Transform to OSCAL format
	transformed := transformOSCAL(data)

	// Step 4: Ensure required metadata fields
	if t, ok := transformed.(map[string]interface{}); ok {
		ensureMetadata(t)
	}

	// Step 5: Wrap in root model key
	wrapped := map[string]interface{}{rootKey: transformed}

	// Step 6: Marshal to JSON
	return json.MarshalIndent(wrapped, "", "  ")
}

// oscalSingleValueFields are fields that OSCAL expects as a single value
// but the proto defines as repeated. Single-element arrays are unwrapped.
var oscalSingleValueFields = map[string]bool{
	"prose":   true,
	"remarks": true,
}

// oscalFieldRenames maps protojson field names to their OSCAL JSON key name
// when camelToKebab alone is not sufficient (e.g., to replace a deprecated
// scalar field with a structured one under the same OSCAL key).
var oscalFieldRenames = map[string]string{
	"citationObj": "citation",
}

// oscalDeprecatedFields are proto field names that have been superseded by
// newer schema-compliant fields. They are stripped from the JSON output to
// avoid violating additionalProperties:false constraints in the OSCAL schema.
// The replacement field is listed for documentation.
var oscalDeprecatedFields = map[string]string{
	"risks":            "replaced by poam-items",
	"guidance":         "replaced by guidelines",
	"citation":         "replaced by citation-obj (renamed to citation)",
	"controlSelection": "replaced by control-selections",
	"alterControls":    "replaced by alters",
	"assessmentTasks":  "replaced by tasks",
	"itemName":         "not in OSCAL schema (use by-id)",
	// "value" in SetParameter is deprecated in favor of "values", but
	// "value" is also the unwrap key for wrapper types, so it cannot be
	// globally suppressed. It is handled contextually by the proto itself
	// (the deprecated scalar value field is only emitted if populated).
}

// transformOSCAL recursively transforms a protojson map into OSCAL JSON format:
//   - camelCase keys → kebab-case
//   - {"value": "x"} → "x" (unwrap single-value wrapper types)
//   - Single-element arrays for prose/remarks → single string
//   - Remove empty objects, empty arrays, and null values
func transformOSCAL(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Unwrap {"value": "x"} → "x"
		if len(v) == 1 {
			if val, ok := v["value"]; ok {
				return transformOSCAL(val)
			}
		}
		// Unwrap {"value": null} → nil
		if len(v) == 1 {
			if val, ok := v["value"]; ok && val == nil {
				return nil
			}
		}

		result := make(map[string]interface{})
		for key, val := range v {
			// Skip the protojson "@type" field
			if key == "@type" {
				continue
			}
			// Skip deprecated fields that have schema-compliant replacements
			if _, deprecated := oscalDeprecatedFields[key]; deprecated {
				continue
			}
			// Skip null values
			if val == nil {
				continue
			}

			transformed := transformOSCAL(val)
			// Skip empty results
			if isEmpty(transformed) {
				continue
			}

			// Convert camelCase to kebab-case, with overrides for renamed fields
			oscalKey := camelToKebab(key)
			if rename, ok := oscalFieldRenames[key]; ok {
				oscalKey = rename
			}

			// Unwrap single-element arrays for fields that OSCAL expects as single values
			if oscalSingleValueFields[oscalKey] {
				if arr, ok := transformed.([]interface{}); ok && len(arr) == 1 {
					transformed = arr[0]
				}
			}

			result[oscalKey] = transformed
		}
		if len(result) == 0 {
			return nil
		}
		return result

	case []interface{}:
		var result []interface{}
		for _, item := range v {
			transformed := transformOSCAL(item)
			if !isEmpty(transformed) {
				result = append(result, transformed)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result

	default:
		return v
	}
}

// isEmpty returns true for nil, empty map, empty array, or empty string.
func isEmpty(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return true
	case map[string]interface{}:
		return len(val) == 0
	case []interface{}:
		return len(val) == 0
	case string:
		return val == ""
	default:
		return false
	}
}

// camelToKebab converts a camelCase string to kebab-case.
// e.g., "backMatter" → "back-matter", "lastModified" → "last-modified"
func camelToKebab(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32) // to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ensureMetadata ensures required OSCAL metadata fields are present.
func ensureMetadata(data map[string]interface{}) {
	meta, ok := data["metadata"].(map[string]interface{})
	if !ok {
		meta = make(map[string]interface{})
		data["metadata"] = meta
	}
	// Ensure oscal-version is set
	if _, ok := meta["oscal-version"]; !ok {
		meta["oscal-version"] = OSCALVersion
	}
	// Ensure last-modified is set (RFC 3339 with timezone)
	if _, ok := meta["last-modified"]; !ok {
		meta["last-modified"] = time.Now().UTC().Format(time.RFC3339)
	}
	// Ensure version is set
	if _, ok := meta["version"]; !ok {
		meta["version"] = "1.0"
	}
}
