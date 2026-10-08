package vectorstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSuppliedEvidenceCannotVerifyWithoutChecks(t *testing.T) {
	metadata := map[string]interface{}{"title": "Empty", "last-modified": "2026-10-08T00:00:00Z", "version": "1", "oscal-version": "1.2.3"}
	for _, kind := range []string{"catalog", "profile"} {
		t.Run(kind, func(t *testing.T) {
			root := map[string]interface{}{"uuid": "6ba7b810-9dad-51d1-80b4-00c04fd430c8", "metadata": metadata}
			if kind == "profile" {
				root["imports"] = []interface{}{map[string]interface{}{"href": "https://example.org/catalog.json", "include-all": map[string]interface{}{}}}
			}
			data, err := json.Marshal(map[string]interface{}{kind: root})
			if err != nil {
				t.Fatal(err)
			}
			parent := t.TempDir()
			artifact := filepath.Join(parent, "artifact.json")
			if err := os.WriteFile(artifact, data, 0600); err != nil {
				t.Fatal(err)
			}
			evidence := filepath.Join(parent, "evidence")
			if err := os.Mkdir(evidence, 0700); err != nil {
				t.Fatal(err)
			}
			state, err := EvaluateArtifact(artifact, evidence)
			if err != nil || state.AntiCount != 0 || state.TotalControls != 0 || state.Evidence != EvidenceUnverified || state.Lifecycle != ModeDegraded || state.Coherence != PartiallyCoherent {
				t.Fatalf("unchecked supplied evidence posture: %+v %v", state, err)
			}
			if len(state.Findings) != 1 || state.Findings[0].ControlID != "EVIDENCE-CHECK" {
				t.Fatal("missing no-check diagnostic")
			}
			optional, err := EvaluateArtifact(artifact, "")
			if err != nil || optional.Lifecycle != ModeNormal {
				t.Fatalf("optional empty evidence behavior changed: %+v %v", optional, err)
			}
		})
	}
}

func TestSchemaFailureDoesNotVerifySuppliedEvidence(t *testing.T) {
	parent := t.TempDir()
	artifact := filepath.Join(parent, "artifact.json")
	if err := os.WriteFile(artifact, []byte(`{"catalog":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := EvaluateArtifact(artifact, parent)
	if err != nil || state.Lifecycle != ModeFailed || state.Evidence != EvidenceUnverified {
		t.Fatalf("unchecked schema failure claimed evidence verification: %+v %v", state, err)
	}
}
