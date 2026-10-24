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

func TestTabular_SSPRoundTrip(t *testing.T) {
	u1 := "550e8400-e29b-41d4-a716-446655440001"
	u2 := "550e8400-e29b-41d4-a716-446655440002"
	u3 := "550e8400-e29b-41d4-a716-446655440003"
	u4 := "550e8400-e29b-41d4-a716-446655440004"
	u5 := "550e8400-e29b-41d4-a716-446655440005"

	sampleSSP := `{
		"system-security-plan": {
			"uuid": "` + u1 + `",
			"metadata": {
				"title": "Production Microservices SSP",
				"last-modified": "2026-10-23T20:00:00Z",
				"version": "1.0",
				"oscal-version": "1.2.3"
			},
			"import-profile": {
				"href": "https://csrc.nist.gov/oscal/profiles/baseline.json"
			},
			"system-characteristics": {
				"system-ids": [{"id": "production-core"}],
				"system-name": "Production Core Engine",
				"description": "High-assurance containerized microservices platform",
				"system-information": {
					"information-types": [
						{
							"uuid": "` + u2 + `",
							"title": "Authentication Data",
							"description": "User credentials and session tokens"
						}
					]
				},
				"authorization-boundary": {
					"description": "AWS us-east-1 VPC with private subnets"
				},
				"status": {
					"state": "operational"
				}
			},
			"system-implementation": {
				"users": [
					{
						"uuid": "` + u3 + `",
						"title": "System Administrator"
					}
				],
				"components": [
					{
						"uuid": "` + u4 + `",
						"type": "software",
						"title": "Auth Gateway",
						"description": "Gateway enforcing OAuth2/OIDC",
						"status": {
							"state": "operational"
						}
					}
				]
			},
			"control-implementation": {
				"description": "Implemented controls for production microservices",
				"implemented-requirements": [
					{
						"uuid": "` + u5 + `",
						"control-id": "ac-2",
						"by-components": [
							{
								"component-uuid": "` + u4 + `",
								"uuid": "550e8400-e29b-41d4-a716-446655440006",
								"description": "Initial account management implementation."
							}
						]
					}
				]
			}
		}
	}`

	// 1. Export SSP to CSV
	csvBytes, err := ExportToCSV([]byte(sampleSSP))
	if err != nil {
		t.Fatalf("ExportToCSV failed for SSP: %v", err)
	}

	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "ac-2") {
		t.Errorf("expected ac-2 in CSV export of SSP")
	}
	if !strings.Contains(csvStr, "Production Core Engine") {
		t.Errorf("expected system name in CSV export of SSP")
	}

	// 2. Modify CSV
	modifiedCSV := strings.Replace(csvStr, "Initial account management implementation.", "Hardened automated provisioning with quarterly account reconciliation.", -1)

	// 3. Import back into SSP
	updatedJSON, err := ImportFromCSV([]byte(sampleSSP), []byte(modifiedCSV))
	if err != nil {
		t.Fatalf("ImportFromCSV failed for SSP: %v", err)
	}

	// 4. Verify updated description
	if !strings.Contains(string(updatedJSON), "Hardened automated provisioning with quarterly account reconciliation") {
		t.Errorf("updated description not found in SSP output")
	}

	// 5. Schema validation
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("init validator: %v", err)
	}
	if err := v.Validate(updatedJSON, schemavalidate.KindSSP); err != nil {
		t.Fatalf("schema validation failed on updated SSP: %v", err)
	}
}

func TestTabular_CatalogExport(t *testing.T) {
	catJSON := `{
		"catalog": {
			"uuid": "550e8400-e29b-41d4-a716-446655440010",
			"metadata": {
				"title": "Minimal Security Catalog",
				"last-modified": "2026-10-23T20:00:00Z",
				"version": "1.0",
				"oscal-version": "1.2.3"
			},
			"groups": [
				{
					"id": "ac",
					"title": "Access Control",
					"controls": [
						{
							"id": "ac-1",
							"title": "Policy and Procedures",
							"parts": [
								{
									"name": "statement",
									"prose": "Develop, document, and disseminate access control policy."
								}
							]
						}
					]
				}
			]
		}
	}`

	csvBytes, err := ExportToCSV([]byte(catJSON))
	if err != nil {
		t.Fatalf("ExportToCSV failed for catalog: %v", err)
	}

	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "ac-1") {
		t.Errorf("expected ac-1 in CSV export of catalog")
	}
	if !strings.Contains(csvStr, "Access Control") {
		t.Errorf("expected Access Control group in CSV export of catalog")
	}
	if !strings.Contains(csvStr, "Develop, document, and disseminate") {
		t.Errorf("expected statement prose in CSV export of catalog")
	}
}
