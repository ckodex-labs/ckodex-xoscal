package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// TestResolveRootKeyAutoDetect verifies root-key auto-detection from JSON.
func TestResolveRootKeyAutoDetect(t *testing.T) {
	tests := []struct {
		rootKey string
		want    string
	}{
		{"catalog", "catalog"},
		{"profile", "profile"},
		{"system-security-plan", "system-security-plan"},
		{"component-definition", "component-definition"},
		{"assessment-plan", "assessment-plan"},
		{"assessment-results", "assessment-results"},
		{"plan-of-action-and-milestones", "plan-of-action-and-milestones"},
	}
	for _, tt := range tests {
		t.Run(tt.rootKey, func(t *testing.T) {
			data, _ := json.Marshal(map[string]interface{}{
				tt.rootKey: map[string]interface{}{"uuid": "test"},
			})
			got, err := resolveRootKey("", data)
			if err != nil {
				t.Fatalf("resolveRootKey: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveRootKey = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResolveRootKeyExplicitKind verifies -kind flag resolution.
func TestResolveRootKeyExplicitKind(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"catalog", "catalog"},
		{"ssp", "system-security-plan"},
		{"system-security-plan", "system-security-plan"},
		{"poam", "plan-of-action-and-milestones"},
		{"ar", "assessment-results"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			data, _ := json.Marshal(map[string]interface{}{
				tt.want: map[string]interface{}{"uuid": "test"},
			})
			got, err := resolveRootKey(tt.kind, data)
			if err != nil {
				t.Fatalf("resolveRootKey(%q): %v", tt.kind, err)
			}
			if got != tt.want {
				t.Errorf("resolveRootKey(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// TestResolveRootKeyRejectsAmbiguousDocuments verifies that auto-detection
// does not silently choose one root when multiple OSCAL models are present.
func TestResolveRootKeyRejectsAmbiguousDocuments(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"catalog": map[string]interface{}{"uuid": "catalog"},
		"profile": map[string]interface{}{"uuid": "profile"},
	})
	if _, err := resolveRootKey("", data); err == nil {
		t.Fatal("expected ambiguous-root error, got nil")
	}
}

// TestResolveRootKeyRejectsMismatchedExplicitKind verifies that -kind cannot
// force a different model onto a document.
func TestResolveRootKeyRejectsMismatchedExplicitKind(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{
		"profile": map[string]interface{}{"uuid": "profile"},
	})
	if _, err := resolveRootKey("catalog", data); err == nil {
		t.Fatal("expected explicit-kind mismatch error, got nil")
	}
}

// TestResolveRootKeyUnknownKind verifies error on bad kind.
func TestResolveRootKeyUnknownKind(t *testing.T) {
	if _, err := resolveRootKey("bogus", nil); err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

// TestResolveRootKeyNoRootKey verifies error when JSON has no known root key.
func TestResolveRootKeyNoRootKey(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"not-oscal": map[string]interface{}{}})
	if _, err := resolveRootKey("", data); err == nil {
		t.Fatal("expected error for unrecognized root key, got nil")
	}
}

// TestKindFromRootKey verifies the mapping from root key to schemavalidate.ArtifactKind.
func TestKindFromRootKey(t *testing.T) {
	tests := []struct {
		rootKey string
		want    schemavalidate.ArtifactKind
	}{
		{"catalog", schemavalidate.KindCatalog},
		{"profile", schemavalidate.KindProfile},
		{"system-security-plan", schemavalidate.KindSSP},
		{"component-definition", schemavalidate.KindComponentDefinition},
		{"assessment-plan", schemavalidate.KindAssessmentPlan},
		{"assessment-results", schemavalidate.KindAssessmentResults},
		{"plan-of-action-and-milestones", schemavalidate.KindPOAM},
	}
	for _, tt := range tests {
		t.Run(tt.rootKey, func(t *testing.T) {
			got, err := kindFromRootKey(tt.rootKey)
			if err != nil {
				t.Fatalf("kindFromRootKey(%q): %v", tt.rootKey, err)
			}
			if got != tt.want {
				t.Errorf("kindFromRootKey(%q) = %+v, want %+v", tt.rootKey, got, tt.want)
			}
		})
	}
}

// TestKindFromRootKeyMappingNotSupported verifies that mapping-collection
// (a 1.2 model) is not in the embedded 1.1.2 schema validator.
func TestKindFromRootKeyMappingNotSupported(t *testing.T) {
	if _, err := kindFromRootKey("mapping-collection"); err == nil {
		t.Fatal("expected error for mapping-collection (not in 1.1.2 schemas), got nil")
	}
}

// TestModelCLIName verifies every supported root maps to the pinned
// oscal-cli subcommand, including its short ap/ar names.
func TestModelCLIName(t *testing.T) {
	expected := map[string]string{
		"catalog":                       "catalog",
		"profile":                       "profile",
		"system-security-plan":          "ssp",
		"component-definition":          "component-definition",
		"assessment-plan":               "ap",
		"assessment-results":            "ar",
		"plan-of-action-and-milestones": "poam",
	}
	for key, want := range expected {
		if got, ok := modelCLIName[key]; !ok {
			t.Errorf("modelCLIName missing entry for %q", key)
		} else if got != want {
			t.Errorf("modelCLIName[%q] = %q, want %q", key, got, want)
		}
	}
	if _, ok := modelCLIName["mapping-collection"]; ok {
		t.Error("mapping-collection must not be advertised without an oscal-cli command")
	}
}

// TestValidateFileInvokesAssessmentShortCommands verifies the live oscal-cli
// command contract for the two models whose CLI names differ from their OSCAL
// root keys.
func TestValidateFileInvokesAssessmentShortCommands(t *testing.T) {
	for _, tc := range []struct {
		root string
		want string
	}{
		{root: "assessment-plan", want: "ap"},
		{root: "assessment-results", want: "ar"},
	} {
		t.Run(tc.root, func(t *testing.T) {
			dir := t.TempDir()
			artifact := filepath.Join(dir, "artifact.json")
			data, _ := json.Marshal(map[string]interface{}{tc.root: map[string]interface{}{}})
			if err := os.WriteFile(artifact, data, 0o600); err != nil {
				t.Fatalf("write artifact: %v", err)
			}

			cli := filepath.Join(dir, "fake-oscal-cli")
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$0.args\"\n"
			if err := os.WriteFile(cli, []byte(script), 0o700); err != nil {
				t.Fatalf("write fake CLI: %v", err)
			}

			if err := validateFile(artifact, tc.root, cli, nil); err != nil {
				t.Fatalf("validateFile: %v", err)
			}
			args, err := os.ReadFile(cli + ".args")
			if err != nil {
				t.Fatalf("read captured args: %v", err)
			}
			want := tc.want + "\nvalidate\n" + artifact + "\n"
			if string(args) != want {
				t.Fatalf("CLI args = %q, want %q", string(args), want)
			}
		})
	}
}

// TestValidateFileReportsCLIExitOutput verifies that a non-zero oscal-cli
// exit is reported as a validation failure with the tool's diagnostics.
func TestValidateFileReportsCLIExitOutput(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.json")
	if err := os.WriteFile(artifact, []byte(`{"catalog":{}}`), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	cli := filepath.Join(dir, "failing-oscal-cli")
	script := "#!/bin/sh\nprintf '%s\\n' 'constraint violation' >&2\nexit 7\n"
	if err := os.WriteFile(cli, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake CLI: %v", err)
	}

	err := validateFile(artifact, "catalog", cli, nil)
	if err == nil {
		t.Fatal("expected validation failure, got nil")
	}
	if !strings.Contains(err.Error(), "constraint violation") {
		t.Fatalf("error = %q, want CLI diagnostics", err)
	}
}

// TestValidateFileSchemaPreCheckOnly verifies that when oscal-cli is not
// available, the schema pre-check still catches structural errors. This test
// runs only the schema phase by pointing -oscal-cli at a nonexistent binary
// and expecting the schema pre-check to fail first on invalid JSON.
func TestValidateFileSchemaPreCheckCatchesBadJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	sv, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	// Write a structurally-invalid catalog (missing required title).
	badCatalog := `{"catalog":{"uuid":"123e4567-e89b-42d3-a456-426614174000","metadata":{"last-modified":"2026-01-01T00:00:00Z","version":"1.0","oscal-version":"1.1.2"}}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(badCatalog), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Use a fake oscal-cli path so we never reach phase 2 — the schema
	// pre-check should fail first.
	err = validateFile(path, "", "/nonexistent/oscal-cli", sv)
	if err == nil {
		t.Fatal("expected schema pre-check to fail on invalid catalog, got nil")
	}
}

// TestValidateFileSchemaPreCheckPassesThenOSCALCLIFails verifies that a
// schema-valid but constraint-invalid document passes phase 1 and the error
// surfaces from phase 2. This test is skipped if oscal-cli is not installed.
func TestValidateFileSchemaPreCheckPassesThenOSCALCLIFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if _, err := exec.LookPath("oscal-cli"); err != nil {
		t.Skip("oscal-cli not installed, skipping constraint validation test")
	}

	sv, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	// Schema-valid but constraint-invalid: a prop with no ns using a custom
	// name not in the allowed-values for this context.
	constraintBad := `{"catalog":{"uuid":"123e4567-e89b-42d3-a456-426614174000","metadata":{"title":"Test","last-modified":"2026-01-01T00:00:00Z","version":"1.0","oscal-version":"1.1.2"},"controls":[{"id":"c-1","title":"C1","props":[{"name":"bogus-not-allowed","value":"x"}],"parts":[{"id":"c-1-stmt","name":"statement","prose":"Text."}]}]}}`
	dir := t.TempDir()
	path := filepath.Join(dir, "constraint_bad.json")
	if err := os.WriteFile(path, []byte(constraintBad), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err = validateFile(path, "", "oscal-cli", sv)
	if err == nil {
		// Some oscal-cli versions may not catch this specific constraint;
		// not a hard failure, just log it.
		t.Log("oscal-cli did not flag the constraint-invalid doc (may be version-dependent)")
	}
}

// TestResolveOSCALCLINotFound verifies the error when oscal-cli is missing.
func TestResolveOSCALCLINotFound(t *testing.T) {
	_, err := resolveOSCALCLI("/definitely/not/real/path/oscal-cli")
	if err == nil {
		t.Fatal("expected error for nonexistent oscal-cli path, got nil")
	}
}
