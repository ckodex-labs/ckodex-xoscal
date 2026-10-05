package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func actualDocument(t *testing.T) (object, map[string]route) {
	t.Helper()
	data, err := buildSpec("../../../proto/oscal/gen/openapi/services/v1")
	if err != nil {
		t.Fatal(err)
	}
	var document object
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	expected, err := contractRoutes()
	if err != nil {
		t.Fatal(err)
	}
	return document, expected
}

func TestActualContractIsCompleteAndDeterministic(t *testing.T) {
	input := "../../../proto/oscal/gen/openapi/services/v1"
	first, err := buildSpec(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildSpec(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("generation is not deterministic")
	}
	document, expected := actualDocument(t)
	counts := map[string]int{}
	for id := range expected {
		parts := strings.Split(id, ".")
		counts[parts[len(parts)-2]]++
	}
	for _, service := range []string{"OscalService", "GovernanceService", "TransparencyExchangeService", "TransparencyGraphService"} {
		if counts[service] == 0 {
			t.Errorf("service %s missing operations", service)
		}
	}
	if len(expected) != 96 {
		t.Fatalf("operation count = %d, update reviewed contract expectation from 96 if RPCs changed", len(expected))
	}
	for path, method := range map[string]string{
		"/v1/component-definitions": "get", "/v1/transparency/claims": "post",
		"/v1/transparency/claims/{claim_id}/receipt": "get", "/v1/search": "get",
		"/v1/search/semantic": "get", "/v1/graph/verify-closure": "post",
	} {
		if item, ok := document["paths"].(object)[path].(object); !ok || item[method] == nil {
			t.Errorf("missing %s %s", method, path)
		}
	}
	checkedIn, err := os.ReadFile("../../../site/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, checkedIn) {
		t.Fatal("site/openapi.json differs from generated complete contract")
	}
	t.Logf("validated %d RPCs across %d services", len(expected), len(counts))
}

func TestMissingServiceOutputBlocksGeneration(t *testing.T) {
	if _, err := buildSpec(t.TempDir()); err == nil || !strings.Contains(err.Error(), "required service document") {
		t.Fatalf("missing-service error = %v", err)
	}
}

func TestContractRejectsMissingOperation(t *testing.T) {
	document, expected := actualDocument(t)
	delete(document["paths"].(object)["/v1/transparency/claims/{claim_id}/receipt"].(object), "get")
	if err := validateDocument(document, expected); err == nil || !strings.Contains(err.Error(), "missing required operation") {
		t.Fatalf("missing-operation error = %v", err)
	}
}

func TestContractRejectsWrongRoute(t *testing.T) {
	document, expected := actualDocument(t)
	paths := document["paths"].(object)
	paths["/v1/components"] = paths["/v1/component-definitions"]
	delete(paths, "/v1/component-definitions")
	if err := validateDocument(document, expected); err == nil || !strings.Contains(err.Error(), "unexpected route") {
		t.Fatalf("wrong-route error = %v", err)
	}
}

func TestContractRejectsBrokenReference(t *testing.T) {
	document, expected := actualDocument(t)
	document["components"].(object)["schemas"].(object)["broken"] = object{"$ref": "#/components/schemas/missing"}
	if err := validateDocument(document, expected); err == nil || !strings.Contains(err.Error(), "unresolved reference") {
		t.Fatalf("broken-reference error = %v", err)
	}
}

func TestContractRejectsRemoteReference(t *testing.T) {
	document, expected := actualDocument(t)
	document["components"].(object)["schemas"].(object)["remote"] = object{"$ref": "https://example.invalid/model.json"}
	if err := validateDocument(document, expected); err == nil || !strings.Contains(err.Error(), "non-local") {
		t.Fatalf("remote-reference error = %v", err)
	}
}

func TestMergeRejectsDuplicateRouteAndConflictingSchema(t *testing.T) {
	document, _ := actualDocument(t)
	if _, err := mergeDocuments([]object{document, document}); err == nil || !strings.Contains(err.Error(), "duplicate path member") {
		t.Fatalf("duplicate-route error = %v", err)
	}
	one := object{"openapi": "3.1.0", "paths": object{"/one": object{}}, "components": object{"schemas": object{"shared": object{"type": "string"}}}}
	two := object{"openapi": "3.1.0", "paths": object{"/two": object{}}, "components": object{"schemas": object{"shared": object{"type": "integer"}}}}
	if _, err := mergeDocuments([]object{one, two}); err == nil || !strings.Contains(err.Error(), "conflicting component") {
		t.Fatalf("schema-conflict error = %v", err)
	}
}

func TestMissingPathParameterBlocksContract(t *testing.T) {
	document, expected := actualDocument(t)
	item := document["paths"].(object)["/v1/transparency/claims/{claim_id}/receipt"].(object)
	item["get"].(object)["parameters"] = []any{}
	if err := validateDocument(document, expected); err == nil || !strings.Contains(err.Error(), "required path parameter") {
		t.Fatalf("parameter error = %v", err)
	}
}

func TestDocsExposeCanonicalRoutesAndPorts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "site", "docs.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{"port 50051", "port 9090", "/v1/component-definitions", "/v1/transparency/claims/{claim_id}/receipt", "/v1/search/semantic", "data-url=\"./openapi.json\"", "src=\"./scalar.js\""} {
		if !strings.Contains(text, required) {
			t.Errorf("docs missing %s", required)
		}
	}
	for _, stale := range []string{"/v1/components", "/v1/claims", "/v1/receipts", "gRPC on port 9090"} {
		if strings.Contains(text, stale) {
			t.Errorf("docs contain stale %s", stale)
		}
	}
}
