// xoscal-openapi assembles the portal's API contract from pinned Buf outputs.
// It rejects ambiguous routes and checks every operation against proto HTTP annotations.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	services "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

type object = map[string]any
type route struct{ method, path string }

var serviceFiles = []protoreflect.FileDescriptor{
	services.File_services_v1_oscal_service_proto,
	services.File_services_v1_governance_service_proto,
	services.File_services_v1_transparency_exchange_service_proto,
	services.File_services_v1_transparency_graph_service_proto,
}

func contractRoutes() (map[string]route, error) {
	routes := map[string]route{}
	owners := map[route]string{}
	for _, file := range serviceFiles {
		for i := 0; i < file.Services().Len(); i++ {
			service := file.Services().Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)
				if !proto.HasExtension(method.Options(), annotations.E_Http) {
					return nil, fmt.Errorf("RPC %s has no HTTP annotation", method.FullName())
				}
				http := proto.GetExtension(method.Options(), annotations.E_Http).(*annotations.HttpRule)
				if len(http.AdditionalBindings) != 0 {
					return nil, fmt.Errorf("RPC %s needs explicit additional-binding coverage", method.FullName())
				}
				var r route
				switch pattern := http.Pattern.(type) {
				case *annotations.HttpRule_Get:
					r = route{"get", pattern.Get}
				case *annotations.HttpRule_Post:
					r = route{"post", pattern.Post}
				case *annotations.HttpRule_Put:
					r = route{"put", pattern.Put}
				case *annotations.HttpRule_Delete:
					r = route{"delete", pattern.Delete}
				case *annotations.HttpRule_Patch:
					r = route{"patch", pattern.Patch}
				default:
					return nil, fmt.Errorf("RPC %s has unsupported HTTP pattern", method.FullName())
				}
				id := string(method.FullName())
				if previous, exists := owners[r]; exists {
					return nil, fmt.Errorf("duplicate proto route %s %s: %s and %s", r.method, r.path, previous, id)
				}
				owners[r], routes[id] = id, r
			}
		}
	}
	return routes, nil
}

func mergeDocuments(documents []object) (object, error) {
	merged := object{
		"openapi": "3.1.0",
		"info":    object{"title": "xOSCAL REST Gateway", "version": "v1", "description": "Generated from the HTTP annotations of all four oscal.services.v1 services. Requests and responses use protobuf JSON envelopes."},
		"paths":   object{}, "components": object{}, "security": []any{},
	}
	tags := map[string]any{}
	for _, document := range documents {
		if document["openapi"] != "3.1.0" {
			return nil, fmt.Errorf("unsupported OpenAPI version: %v", document["openapi"])
		}
		if security, ok := document["security"].([]any); ok && len(security) > 0 {
			return nil, fmt.Errorf("document-level security requires an explicit merge policy")
		}
		paths, ok := document["paths"].(object)
		if !ok || len(paths) == 0 {
			return nil, fmt.Errorf("service document has no paths")
		}
		for path, value := range paths {
			item, ok := value.(object)
			if !ok {
				return nil, fmt.Errorf("invalid path item %s", path)
			}
			target, exists := merged["paths"].(object)[path].(object)
			if !exists {
				target = object{}
				merged["paths"].(object)[path] = target
			}
			for key, operation := range item {
				if _, exists := target[key]; exists {
					return nil, fmt.Errorf("duplicate path member %s %s", key, path)
				}
				target[key] = operation
			}
		}
		components, ok := document["components"].(object)
		if !ok {
			return nil, fmt.Errorf("service document has no components")
		}
		for category, raw := range components {
			entries, ok := raw.(object)
			if !ok {
				return nil, fmt.Errorf("invalid components category %s", category)
			}
			target, exists := merged["components"].(object)[category].(object)
			if !exists {
				target = object{}
				merged["components"].(object)[category] = target
			}
			for name, entry := range entries {
				if previous, exists := target[name]; exists && !reflect.DeepEqual(previous, entry) {
					return nil, fmt.Errorf("conflicting component %s/%s", category, name)
				}
				target[name] = entry
			}
		}
		for _, raw := range asArray(document["tags"]) {
			tag, ok := raw.(object)
			if !ok {
				return nil, fmt.Errorf("invalid service tag")
			}
			name, ok := tag["name"].(string)
			if !ok || name == "" {
				return nil, fmt.Errorf("missing service tag name")
			}
			if previous, exists := tags[name]; exists && !reflect.DeepEqual(previous, tag) {
				return nil, fmt.Errorf("conflicting service tag %s", name)
			}
			tags[name] = tag
		}
	}
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	sort.Strings(names)
	ordered := []any{}
	for _, name := range names {
		ordered = append(ordered, tags[name])
	}
	merged["tags"] = ordered
	return merged, nil
}

func asArray(value any) []any { result, _ := value.([]any); return result }

// The pinned OpenAPI plugin spells path placeholders using protobuf JSON names.
// Restore the annotation's field spelling without changing URL matching, while
// checking the generated names against descriptor JSONName (not a guessed case rule).
func normalizePathParameters(document object, expected map[string]route) error {
	inputs := map[string]protoreflect.MessageDescriptor{}
	for _, file := range serviceFiles {
		for i := 0; i < file.Services().Len(); i++ {
			methods := file.Services().Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				method := methods.Get(j)
				inputs[string(method.FullName())] = method.Input()
			}
		}
	}
	paths, ok := document["paths"].(object)
	if !ok {
		return fmt.Errorf("service document has no paths")
	}
	normalized := object{}
	for path, raw := range paths {
		item, ok := raw.(object)
		if !ok {
			return fmt.Errorf("invalid path item %s", path)
		}
		canonical := ""
		for verb, rawOperation := range item {
			operation, ok := rawOperation.(object)
			if !ok {
				return fmt.Errorf("unsupported path item member %s %s", verb, path)
			}
			id, _ := operation["operationId"].(string)
			r, exists := expected[id]
			if !exists || verb != r.method {
				return fmt.Errorf("unexpected generated operation %s %s %s", id, verb, path)
			}
			jsonPath := r.path
			for _, part := range strings.Split(r.path, "{")[1:] {
				name, _, closed := strings.Cut(part, "}")
				if !closed {
					return fmt.Errorf("invalid annotated path %s", r.path)
				}
				message := inputs[id]
				jsonNames := []string{}
				for _, segment := range strings.Split(name, ".") {
					if message == nil {
						return fmt.Errorf("invalid annotated field %s", name)
					}
					field := message.Fields().ByName(protoreflect.Name(segment))
					if field == nil {
						return fmt.Errorf("unknown annotated field %s", name)
					}
					jsonNames = append(jsonNames, field.JSONName())
					message = field.Message()
				}
				jsonName := strings.Join(jsonNames, ".")
				jsonPath = strings.ReplaceAll(jsonPath, "{"+name+"}", "{"+jsonName+"}")
				for _, rawParameter := range asArray(operation["parameters"]) {
					parameter, ok := rawParameter.(object)
					if ok && parameter["in"] == "path" && parameter["name"] == jsonName {
						parameter["name"] = name
					}
				}
			}
			if path != jsonPath {
				return fmt.Errorf("operation %s generated unexpected route %s (want %s)", id, path, jsonPath)
			}
			if canonical != "" && canonical != r.path {
				return fmt.Errorf("ambiguous canonical path %s", path)
			}
			canonical = r.path
		}
		if _, exists := normalized[canonical]; exists {
			return fmt.Errorf("duplicate canonical path %s", canonical)
		}
		normalized[canonical] = item
	}
	document["paths"] = normalized
	return nil
}

func validateDocument(document object, expected map[string]route) error {
	paths, ok := document["paths"].(object)
	if !ok || len(paths) == 0 {
		return fmt.Errorf("OpenAPI paths are empty")
	}
	seen := map[string]bool{}
	for path, raw := range paths {
		item, ok := raw.(object)
		if !ok {
			return fmt.Errorf("invalid path item %s", path)
		}
		for method, raw := range item {
			if !strings.Contains(" get post put delete patch head options trace ", " "+method+" ") {
				continue
			}
			operation, ok := raw.(object)
			if !ok {
				return fmt.Errorf("invalid operation %s %s", method, path)
			}
			id, ok := operation["operationId"].(string)
			if !ok || id == "" || seen[id] {
				return fmt.Errorf("missing or duplicate operationId %q", id)
			}
			if required, exists := expected[id]; !exists || required != (route{method, path}) {
				return fmt.Errorf("operation %s has unexpected route %s %s", id, method, path)
			}
			if responses, ok := operation["responses"].(object); !ok || len(responses) == 0 {
				return fmt.Errorf("operation %s has no responses", id)
			}
			parameters := asArray(item["parameters"])
			parameters = append(append([]any{}, parameters...), asArray(operation["parameters"])...)
			for _, part := range strings.Split(path, "{")[1:] {
				name, _, closed := strings.Cut(part, "}")
				if !closed {
					return fmt.Errorf("unclosed path parameter %s", path)
				}
				found := false
				for _, raw := range parameters {
					parameter, ok := raw.(object)
					if ok && parameter["in"] == "path" && parameter["name"] == name && parameter["required"] == true {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("operation %s lacks required path parameter %s", id, name)
				}
			}
			seen[id] = true
		}
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("missing required operation %s", id)
		}
	}
	return checkReferences(document, document)
}

func checkReferences(root object, node any) error {
	switch value := node.(type) {
	case object:
		if raw, exists := value["$ref"]; exists {
			ref, ok := raw.(string)
			if !ok || !strings.HasPrefix(ref, "#/") {
				return fmt.Errorf("non-local or invalid reference %v", raw)
			}
			var target any = root
			for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
				key := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
				container, ok := target.(object)
				if !ok {
					return fmt.Errorf("unresolved reference %s", ref)
				}
				target, ok = container[key]
				if !ok {
					return fmt.Errorf("unresolved reference %s", ref)
				}
			}
		}
		for _, child := range value {
			if err := checkReferences(root, child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := checkReferences(root, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildSpec(input string) ([]byte, error) {
	expected, err := contractRoutes()
	if err != nil {
		return nil, err
	}
	documents := []object{}
	// Explicit files prevent schema-only documents or filesystem order from selecting the API.
	for _, file := range serviceFiles {
		name := strings.TrimSuffix(filepath.Base(file.Path()), ".proto") + ".openapi.yaml"
		data, err := os.ReadFile(filepath.Join(input, name))
		if err != nil {
			return nil, fmt.Errorf("required service document %s: %w", name, err)
		}
		var document object
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := normalizePathParameters(document, expected); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		documents = append(documents, document)
	}
	document, err := mergeDocuments(documents)
	if err != nil {
		return nil, err
	}
	if err := validateDocument(document, expected); err != nil {
		return nil, err
	}
	// encoding/json sorts string map keys; tags use a sorted service order.
	data, err := json.MarshalIndent(document, "", "  ")
	return append(data, '\n'), err
}

func main() {
	input := flag.String("input", "proto/oscal/gen/openapi/services/v1", "directory of required Buf service specs")
	output := flag.String("output", "site/openapi.json", "deterministic merged JSON destination")
	check := flag.Bool("check", false, "check the destination for generation drift")
	flag.Parse()
	data, err := buildSpec(*input)
	if err == nil && *check {
		var existing []byte
		existing, err = os.ReadFile(*output)
		if err == nil && string(existing) != string(data) {
			err = fmt.Errorf("OpenAPI generation drift at %s", *output)
		}
	} else if err == nil {
		err = os.WriteFile(*output, data, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
