package integrity_test

import (
	"testing"

	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestIntegrity_ValidSSP(t *testing.T) {
	validSSP := `{
		"system-security-plan": {
			"uuid": "550e8400-e29b-41d4-a716-446655440001",
			"metadata": {
				"title": "Valid SSP",
				"last-modified": "2026-10-23T00:00:00Z",
				"version": "1.0",
				"oscal-version": "1.2.3"
			},
			"system-implementation": {
				"components": [
					{
						"uuid": "550e8400-e29b-41d4-a716-446655440002",
						"type": "software",
						"title": "Auth Component"
					}
				]
			},
			"control-implementation": {
				"implemented-requirements": [
					{
						"uuid": "550e8400-e29b-41d4-a716-446655440003",
						"control-id": "ac-2",
						"by-components": [
							{
								"component-uuid": "550e8400-e29b-41d4-a716-446655440002",
								"uuid": "550e8400-e29b-41d4-a716-446655440004",
								"description": "Implemented account management"
							}
						]
					}
				]
			}
		}
	}`

	findings, err := integrity.Check([]byte(validSSP), schemavalidate.KindSSP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range findings {
		if f.Severity == integrity.SeverityError {
			t.Errorf("expected 0 errors on valid SSP, got: %+v", f)
		}
	}
}

func TestIntegrity_OrphanComponentRef(t *testing.T) {
	sspWithOrphan := `{
		"system-security-plan": {
			"uuid": "550e8400-e29b-41d4-a716-446655440001",
			"system-implementation": {
				"components": [
					{
						"uuid": "550e8400-e29b-41d4-a716-446655440002",
						"type": "software",
						"title": "Auth Component"
					}
				]
			},
			"control-implementation": {
				"implemented-requirements": [
					{
						"uuid": "550e8400-e29b-41d4-a716-446655440003",
						"control-id": "ac-2",
						"by-components": [
							{
								"component-uuid": "550e8400-e29b-41d4-a716-446655449999",
								"uuid": "550e8400-e29b-41d4-a716-446655440004",
								"description": "Orphan reference"
							}
						]
					}
				]
			}
		}
	}`

	findings, err := integrity.Check([]byte(sspWithOrphan), schemavalidate.KindSSP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundOrphan := false
	for _, f := range findings {
		if f.Rule == "INT-SSP-ORPHAN-COMPONENT-REF" {
			foundOrphan = true
			break
		}
	}
	if !foundOrphan {
		t.Errorf("expected INT-SSP-ORPHAN-COMPONENT-REF finding for missing component UUID")
	}
}

func TestIntegrity_InvalidUUID(t *testing.T) {
	badUUIDDoc := `{
		"component-definition": {
			"uuid": "not-a-valid-uuid-12345",
			"metadata": {
				"title": "Test CompDef"
			}
		}
	}`

	findings, err := integrity.Check([]byte(badUUIDDoc), schemavalidate.KindComponentDefinition)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundBadUUID := false
	for _, f := range findings {
		if f.Rule == "INT-UUID-RFC4122" {
			foundBadUUID = true
			break
		}
	}
	if !foundBadUUID {
		t.Errorf("expected INT-UUID-RFC4122 finding for bad UUID")
	}
}
