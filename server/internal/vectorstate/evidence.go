package vectorstate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/rootfs"
)

type evidenceSource struct {
	root             *os.Root
	entries          []os.DirEntry
	verifiedControls int
}

func openEvidence(dir string) (*evidenceSource, error) {
	if dir == "" {
		return nil, nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	entries, err := rootfs.ReadDir(root, ".")
	if err != nil {
		return nil, fmt.Errorf("enumerate evidence: %w", errorsWithClose(err, root))
	}
	return &evidenceSource{root: root, entries: entries}, nil
}

func errorsWithClose(err error, root *os.Root) error {
	if closeErr := root.Close(); closeErr != nil {
		return fmt.Errorf("%w; close directory: %v", err, closeErr)
	}
	return err
}

func (source *evidenceSource) evaluate(ctrlID string, finding *ControlFinding, state *VectorState) {
	matched, allVerified := false, true
	for _, entry := range source.entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, ".sha256") || !strings.Contains(name, ctrlID) {
			continue
		}
		info, err := source.root.Stat(name)
		if err == nil && info.IsDir() {
			continue
		}
		matched = true
		if !source.verifyBlob(name, finding, state) {
			allVerified = false
		}
	}
	if !matched {
		unverified(finding, state, fmt.Sprintf("No corroborating evidence blob found matching control %s", ctrlID))
	}
	if matched && allVerified {
		source.verifiedControls++
	}
}

func (source *evidenceSource) verifyBlob(name string, finding *ControlFinding, state *VectorState) bool {
	data, err := rootfs.ReadFile(source.root, name)
	if err != nil {
		unverified(finding, state, "Evidence blob cannot be read within the selected directory")
		return false
	}
	finding.Evidence = append(finding.Evidence, name)
	sidecar, err := rootfs.ReadFile(source.root, name+".sha256")
	expected := strings.TrimSpace(string(sidecar))
	decoded, decodeErr := hex.DecodeString(expected)
	if err != nil || decodeErr != nil || len(decoded) != sha256.Size {
		unverified(finding, state, "Evidence requires a readable SHA-256 sidecar containing exactly 64 hexadecimal characters")
		return false
	}
	actual := sha256.Sum256(data)
	if !strings.EqualFold(expected, hex.EncodeToString(actual[:])) {
		finding.Status, finding.Valence = "violation", "NEGATIVE"
		finding.Remediation = fmt.Sprintf("Evidence blob %s checksum mismatch", name)
		state.Evidence = EvidenceTampered
		return false
	}
	return true
}

func (source *evidenceSource) finish(state *VectorState) {
	if state.TotalControls > 0 && source.verifiedControls == state.TotalControls && state.Evidence != EvidenceTampered {
		state.Evidence = EvidenceVerified
	}
	if state.TotalControls == 0 {
		state.Findings = append(state.Findings, ControlFinding{ControlID: "EVIDENCE-CHECK", Component: state.ArtifactName,
			Status: "degraded", Valence: "MIXED", Description: "No control evidence checks ran for this artifact",
			Remediation: "Use a supported control-bearing artifact before claiming evidence verification"})
	}
}

func unverified(finding *ControlFinding, state *VectorState, remediation string) {
	if state.Evidence != EvidenceTampered {
		state.Evidence = EvidenceUnverified
	}
	if finding.Status == "violation" {
		return
	}
	finding.Status, finding.Valence = "degraded", "MIXED"
	finding.Remediation = remediation
}
