package tabular

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/scaffold"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestTabular_RoundTrip(t *testing.T) {
	// Get JSON from scaffold generator
	tmpDir := t.TempDir()
	_, baseJSON, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	// 1. Export to CSV
	csvBytes, err := ExportToCSV(baseJSON)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "Control ID,Component,Status") {
		t.Errorf("expected CSV headers, got:\n%s", csvStr)
	}
	if !strings.Contains(csvStr, "ac-2") {
		t.Errorf("expected ac-2 row in CSV")
	}

	// 2. Simulate GRC Analyst edit in spreadsheet
	modifiedCSV := strings.Replace(csvStr, "Account management implemented via service authentication and credential checks.", "Enhanced enterprise MFA via FIDO2 WebAuthn keys and strict role separation.", 1)

	// 3. Import modified CSV
	updatedJSON, err := ImportFromCSV(baseJSON, []byte(modifiedCSV))
	if err != nil {
		t.Fatalf("ImportFromCSV failed: %v", err)
	}

	// 4. Verify updated text in JSON
	if !strings.Contains(string(updatedJSON), "Enhanced enterprise MFA via FIDO2 WebAuthn keys") {
		t.Errorf("expected updated text in JSON")
	}

	// 5. Verify OSCAL 1.2.3 Schema Validation
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	if err := v.Validate(updatedJSON, schemavalidate.KindComponentDefinition); err != nil {
		t.Fatalf("schema validation failed on imported artifact: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(updatedJSON, &parsed); err != nil {
		t.Fatalf("parse updated JSON: %v", err)
	}
}
