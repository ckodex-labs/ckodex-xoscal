// Package schemavalidate validates OSCAL JSON artifacts against the official
// NIST OSCAL 1.1.2 complete JSON schema. The complete schema is embedded so
// all cross-model $ref references resolve without network access.
package schemavalidate

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/v1.1.2/oscal_complete_schema.json
var completeSchemaFS embed.FS

// SchemaVersion is the OSCAL version the embedded schema conforms to.
const SchemaVersion = "1.1.2"

// ArtifactKind identifies an OSCAL artifact type and its root key.
type ArtifactKind struct {
	Name    string // e.g., "catalog"
	RootKey string // root wrapper key, e.g., "catalog"
}

// Artifact kinds supported by this validator.
var (
	KindCatalog             = ArtifactKind{"catalog", "catalog"}
	KindProfile             = ArtifactKind{"profile", "profile"}
	KindSSP                 = ArtifactKind{"ssp", "system-security-plan"}
	KindComponentDefinition = ArtifactKind{"component-definition", "component-definition"}
	KindAssessmentPlan      = ArtifactKind{"assessment-plan", "assessment-plan"}
	KindAssessmentResults   = ArtifactKind{"assessment-results", "assessment-results"}
	KindPOAM                = ArtifactKind{"poam", "plan-of-action-and-milestones"}
)

// Validator validates OSCAL JSON artifacts against the official complete schema.
type Validator struct {
	schemas map[string]*jsonschema.Schema // root key → compiled subschema
}

// NewValidator loads the embedded OSCAL complete schema and compiles a
// separate subschema for each of the 7 OSCAL model types.
func NewValidator() (*Validator, error) {
	data, err := completeSchemaFS.ReadFile("schemas/v1.1.2/oscal_complete_schema.json")
	if err != nil {
		return nil, fmt.Errorf("read embedded schema: %w", err)
	}

	var schemaDoc interface{}
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	// Rewrite $id anchors to #/definitions/ paths for the compiler
	rewriteAnchors(schemaDoc)

	root, ok := schemaDoc.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema is not an object")
	}
	defs, ok := root["definitions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema has no definitions")
	}

	// Build a per-model schema that only includes the matching oneOf branch.
	// This avoids oneOf noise where non-matching branches produce errors.
	oneOf, ok := root["oneOf"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("schema has no oneOf")
	}

	v := &Validator{schemas: make(map[string]*jsonschema.Schema)}

	for _, branch := range oneOf {
		bm, ok := branch.(map[string]interface{})
		if !ok {
			continue
		}
		props, ok := bm["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		// Find the model root key (e.g., "catalog", "profile")
		for key := range props {
			if key == "$schema" {
				continue
			}
			// Build a standalone schema for this model
			modelSchema := buildModelSchema(root, defs, bm, key)
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource(key+"_schema.json", modelSchema); err != nil {
				return nil, fmt.Errorf("add schema resource for %s: %w", key, err)
			}
			compiled, err := compiler.Compile(key + "_schema.json")
			if err != nil {
				return nil, fmt.Errorf("compile schema for %s: %w", key, err)
			}
			v.schemas[key] = compiled
		}
	}

	return v, nil
}

// buildModelSchema creates a standalone schema for a single OSCAL model type.
// It takes the root schema's definitions and wraps the model's oneOf branch
// so that only the matching model is validated.
func buildModelSchema(root, defs map[string]interface{}, branch map[string]interface{}, modelKey string) map[string]interface{} {
	schema := make(map[string]interface{})
	schema["$schema"] = "http://json-schema.org/draft-07/schema#"
	schema["type"] = "object"
	schema["properties"] = branch["properties"]
	schema["required"] = branch["required"]
	if add, ok := branch["additionalProperties"]; ok {
		schema["additionalProperties"] = add
	}
	// Include all definitions for $ref resolution
	schema["definitions"] = defs
	return schema
}

// rewriteAnchors converts OSCAL's $id anchor pattern to standard #/definitions/
// references that the santhosh-tekuri/jsonschema compiler can resolve.
//
// OSCAL schemas define each subschema with both a key in "definitions" and a
// "$id" anchor like "#assembly_oscal-catalog_catalog". Other subschemas reference
// these via "$ref": "#assembly_oscal-catalog_catalog". We:
//  1. Build a map from $id anchor → definitions key
//  2. Rewrite all $ref "#anchor" → "#/definitions/<key>"
//  3. Remove all $id fields
func rewriteAnchors(doc interface{}) {
	root, ok := doc.(map[string]interface{})
	if !ok {
		return
	}
	defs, ok := root["definitions"].(map[string]interface{})
	if !ok {
		return
	}

	// Build anchor → key map
	anchorToKey := make(map[string]string)
	for key, def := range defs {
		if dm, ok := def.(map[string]interface{}); ok {
			if id, ok := dm["$id"].(string); ok && strings.HasPrefix(id, "#") {
				anchorToKey[id] = key
			}
		}
	}

	// Rewrite all $ref and remove $id
	var rewrite func(v interface{})
	rewrite = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			if ref, ok := val["$ref"].(string); ok {
				if newRef, ok := resolveRef(ref, anchorToKey); ok {
					val["$ref"] = newRef
				}
			}
			delete(val, "$id")
			for _, child := range val {
				rewrite(child)
			}
		case []interface{}:
			for _, item := range val {
				rewrite(item)
			}
		}
	}

	// Also rewrite the top-level oneOf $refs
	rewrite(root)
}

// resolveRef converts an anchor-style $ref to a #/definitions/... path.
func resolveRef(ref string, anchorToKey map[string]string) (string, bool) {
	if !strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "#/") {
		return ref, false
	}
	// ref is like "#assembly_oscal-catalog_catalog"
	key, ok := anchorToKey[ref]
	if !ok {
		// Try to find a definitions key that matches the anchor name
		anchorName := strings.TrimPrefix(ref, "#")
		if _, exists := anchorToKey["#"+anchorName]; !exists {
			// Check if it's a direct definitions key
			return "#/definitions/" + anchorName, true
		}
		return ref, false
	}
	return "#/definitions/" + key, true
}

// ValidationError describes a single schema violation.
type ValidationError struct {
	Message  string
	Instance string
}

// Error implements error.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s at %s", e.Message, e.Instance)
}

// Validate validates raw OSCAL JSON bytes against the per-model schema.
func (v *Validator) Validate(data []byte, kind ArtifactKind) error {
	schema, ok := v.schemas[kind.RootKey]
	if !ok {
		return fmt.Errorf("no schema loaded for root key: %s", kind.RootKey)
	}

	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("unmarshal artifact: %w", err)
	}

	if err := schema.Validate(doc); err != nil {
		var errs []ValidationError
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			for _, e := range flattenErrors(ve) {
				errs = append(errs, e)
			}
		} else {
			errs = append(errs, ValidationError{Message: err.Error()})
		}
		return &ValidationResult{Errors: errs, Kind: kind.Name}
	}
	return nil
}

// ValidationResult holds multiple validation errors.
type ValidationResult struct {
	Errors []ValidationError
	Kind   string
}

// Error implements error.
func (r *ValidationResult) Error() string {
	if len(r.Errors) == 0 {
		return "validation failed"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OSCAL %s schema validation failed (%d errors):\n", r.Kind, len(r.Errors)))
	for i, e := range r.Errors {
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, e.Error()))
	}
	return sb.String()
}

// flattenErrors recursively extracts leaf validation errors.
func flattenErrors(ve *jsonschema.ValidationError) []ValidationError {
	if len(ve.Causes) == 0 {
		msg := fmt.Sprintf("%v", ve.ErrorKind)
		return []ValidationError{{
			Message:  msg,
			Instance: instancePath(ve),
		}}
	}
	var errs []ValidationError
	for _, cause := range ve.Causes {
		errs = append(errs, flattenErrors(cause)...)
	}
	return errs
}

func instancePath(ve *jsonschema.ValidationError) string {
	if len(ve.InstanceLocation) == 0 {
		return "(root)"
	}
	return "/" + strings.Join(ve.InstanceLocation, "/")
}
