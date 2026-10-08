package vectorstate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
)

func TestEvidenceMustActuallyBeVerified(t *testing.T) {
	for _, scenario := range []string{"missing", "sidecar-only", "directory-only", "missing-checksum", "broken-blob", "broken-sidecar", "malformed-secret", "uppercase", "mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			artifact, evidence := evidenceFixture(t)
			blob := filepath.Join(evidence, "ac-2.log")
			checksum := evidenceHash([]byte("valid evidence"))
			if scenario != "missing" && scenario != "sidecar-only" && scenario != "directory-only" {
				writeEvidence(t, blob, "valid evidence")
			}
			switch scenario {
			case "sidecar-only":
				writeEvidence(t, blob+".sha256", checksum)
			case "directory-only":
				if err := os.Mkdir(blob, 0700); err != nil {
					t.Fatal(err)
				}
			case "broken-blob":
				if err := os.Remove(blob); err != nil {
					t.Fatal(err)
				}
				linkEvidence(t, "missing", blob)
				writeEvidence(t, blob+".sha256", checksum)
			case "broken-sidecar":
				linkEvidence(t, "missing", blob+".sha256")
			case "malformed-secret":
				writeEvidence(t, blob+".sha256", "outside secret must never appear")
			case "uppercase":
				writeEvidence(t, blob+".sha256", "\n"+strings.ToUpper(checksum)+"\n")
			case "mismatch":
				writeEvidence(t, blob+".sha256", strings.Repeat("0", 64))
			}
			state, err := EvaluateArtifact(artifact, evidence)
			if err != nil {
				t.Fatal(err)
			}
			want, lifecycle, passing, degraded, anti := EvidenceUnverified, ModeDegraded, 0, 1, 0
			if scenario == "uppercase" {
				want, lifecycle, passing, degraded = EvidenceVerified, ModeNormal, 1, 0
			}
			if scenario == "mismatch" {
				want, lifecycle, degraded, anti = EvidenceTampered, ModeQuarantined, 0, 1
			}
			if state.Evidence != want || state.Lifecycle != lifecycle || state.PassingCount != passing || state.DegradedCount != degraded || state.AntiCount != anti {
				t.Fatalf("inconsistent posture: %+v", state)
			}
			assertNoSecret(t, state)
		})
	}
}

func TestEvidenceOutsideLinksAreUnverified(t *testing.T) {
	for _, member := range []string{"blob", "sidecar"} {
		for _, kind := range []string{"absolute", "relative", "chain"} {
			t.Run(member+"/"+kind, func(t *testing.T) {
				artifact, evidence := evidenceFixture(t)
				outside := filepath.Join(filepath.Dir(evidence), "evidence[literal]-sibling")
				if err := os.Mkdir(outside, 0700); err != nil {
					t.Fatal(err)
				}
				writeEvidence(t, filepath.Join(outside, "secret"), "outside sentinel")
				blob := filepath.Join(evidence, "ac-2.log")
				writeEvidence(t, blob, "valid evidence")
				writeEvidence(t, blob+".sha256", evidenceHash([]byte("valid evidence")))
				link := blob
				if member == "sidecar" {
					link += ".sha256"
				}
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(outside, "secret")
				if kind == "relative" {
					target = "../evidence[literal]-sibling/secret"
				}
				if kind == "chain" {
					linkEvidence(t, "../evidence[literal]-sibling/secret", filepath.Join(evidence, "bridge"))
					target = "bridge"
				}
				linkEvidence(t, target, link)
				state, err := EvaluateArtifact(artifact, evidence)
				if err != nil {
					t.Fatal(err)
				}
				if state.Evidence != EvidenceUnverified || state.Lifecycle != ModeDegraded || state.PassingCount != 0 {
					t.Fatalf("unsafe evidence posture: %+v", state)
				}
				if strings.Contains(state.FormatTable(false), "outside sentinel") {
					t.Fatal("outside bytes disclosed")
				}
			})
		}
	}
}

func TestEvidenceInsideLinksAndSelectedRootLink(t *testing.T) {
	artifact, evidence := evidenceFixture(t)
	writeEvidence(t, filepath.Join(evidence, "payload"), "valid evidence")
	writeEvidence(t, filepath.Join(evidence, "receipt"), evidenceHash([]byte("valid evidence")))
	linkEvidence(t, "payload", filepath.Join(evidence, "ac-2.log"))
	linkEvidence(t, "receipt", filepath.Join(evidence, "ac-2.log.sha256"))
	rootLink := filepath.Join(filepath.Dir(evidence), "root-link")
	linkEvidence(t, evidence, rootLink)
	state, err := EvaluateArtifact(artifact, rootLink)
	if err != nil || state.Evidence != EvidenceVerified || state.PassingCount != 1 {
		t.Fatalf("inside links/root link failed: %+v %v", state, err)
	}
	if _, err := EvaluateArtifact(artifact, filepath.Join(evidence, "absent")); err == nil {
		t.Fatal("missing selected evidence root accepted")
	}
}

func TestDerogationCannotMaskEvidenceTampering(t *testing.T) {
	artifact, evidence := evidenceFixture(t)
	writeEvidence(t, filepath.Join(evidence, "ac-2.log"), "valid evidence")
	writeEvidence(t, filepath.Join(evidence, "ac-2.log.sha256"), strings.Repeat("0", 64))
	store := &derogation.Store{Entries: []derogation.DerogationEntry{{ControlID: "ac-2", Authority: "test", ExpiresAt: time.Now().Add(time.Hour)}}}
	if err := store.Save(filepath.Join(filepath.Dir(artifact), ".xoscal", "derogations.json")); err != nil {
		t.Fatal(err)
	}
	state, err := EvaluateArtifact(artifact, evidence)
	if err != nil || state.Lifecycle != ModeQuarantined || state.Findings[0].Status != "violation" || state.DerogatedCount != 0 {
		t.Fatalf("tampering masked: %+v %v", state, err)
	}
}

func evidenceFixture(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	scan := &scaffold.ProjectScan{Name: "test", Version: "1", Components: []scaffold.ComponentCandidate{{Name: "core", Type: "software", Title: "Core", Description: "Core service", Controls: []scaffold.SuggestedControl{{ControlID: "ac-2", Description: "Account management configured via policy engine."}}}}}
	data, err := oscal.ExportComponentDefinitionJSON(scaffold.GenerateComponentDefinition(scan, ""))
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(parent, "artifact.json")
	if err := os.WriteFile(artifact, data, 0600); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(parent, "evidence[literal]")
	if err := os.Mkdir(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	return artifact, evidence
}

func evidenceHash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func writeEvidence(t *testing.T, name, data string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func linkEvidence(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}

func assertNoSecret(t *testing.T, state *VectorState) {
	t.Helper()
	if strings.Contains(state.FormatTable(false), "outside secret") {
		t.Fatal("sidecar bytes disclosed")
	}
}
