package vectorstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EvaluateArtifact inspects an OSCAL JSON artifact, validates its schema,
// checks evidence integrity, and computes its Vector State.
func EvaluateArtifact(artifactPath, evidenceDir string) (result *VectorState, resultErr error) {
	evidence, err := openEvidence(evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("open evidence directory: %w", err)
	}
	if evidence != nil {
		defer func() { resultErr = errors.Join(resultErr, evidence.root.Close()) }()
	}
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	kind, rootKey, err := artifactKind(root)
	if err != nil {
		return nil, err
	}
	state := initialState(artifactPath, kind)
	if evidence != nil {
		state.Evidence = EvidenceUnverified
	}
	valid, err := validateArtifact(data, kind, state)
	if err != nil {
		return nil, err
	}
	if !valid {
		return state, nil
	}
	target, _ := root[rootKey].(map[string]interface{})
	evaluateRequirements(target, rootKey, state, evidence, loadDerogations(artifactPath))
	if evidence != nil {
		evidence.finish(state)
	}
	synthesizeState(state)
	return state, nil
}

func artifactKind(root map[string]interface{}) (schemavalidate.ArtifactKind, string, error) {
	var kind schemavalidate.ArtifactKind
	var rootKey string
	for key := range root {
		switch key {
		case "component-definition":
			kind = schemavalidate.KindComponentDefinition
			rootKey = key
		case "system-security-plan":
			kind = schemavalidate.KindSSP
			rootKey = key
		case "catalog":
			kind = schemavalidate.KindCatalog
			rootKey = key
		case "profile":
			kind = schemavalidate.KindProfile
			rootKey = key
		case "assessment-results":
			kind = schemavalidate.KindAssessmentResults
			rootKey = key
		case "plan-of-action-and-milestones":
			kind = schemavalidate.KindPOAM
			rootKey = key
		case "assessment-plan":
			kind = schemavalidate.KindAssessmentPlan
			rootKey = key
		case "mapping-collection", "mapping":
			kind = schemavalidate.KindMapping
			rootKey = key
		}
		if rootKey != "" {
			break
		}
	}

	if rootKey == "" {
		return kind, "", fmt.Errorf("unrecognized or unsupported OSCAL artifact root key")
	}

	return kind, rootKey, nil
}

func initialState(artifactPath string, kind schemavalidate.ArtifactKind) *VectorState {
	return &VectorState{
		ArtifactName: filepath.Base(artifactPath),
		ArtifactKind: kind.Name,
		Presence:     PresencePresent,
		Valence:      ValencePositive,
		AntiCount:    0,
		Coherence:    Coherent,
		Evidence:     EvidenceVerified,
		Lifecycle:    ModeNormal,
		Epoch:        time.Now().UTC(),
		Findings:     make([]ControlFinding, 0),
	}

}

func validateArtifact(data []byte, kind schemavalidate.ArtifactKind, state *VectorState) (bool, error) {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return false, fmt.Errorf("init validator: %w", err)
	}

	if err := v.Validate(data, kind); err != nil {
		state.Lifecycle = ModeFailed
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
		state.AntiCount++
		state.Findings = append(state.Findings, ControlFinding{
			ControlID:   "SCHEMA-ROOT",
			Component:   state.ArtifactName,
			Status:      "violation",
			Valence:     "NEGATIVE",
			Description: fmt.Sprintf("OSCAL %s schema validation failed", schemavalidate.SchemaVersion),
			Remediation: err.Error(),
		})
		return false, nil
	}

	return true, nil
}

func loadDerogations(artifactPath string) *derogation.Store {
	path := filepath.Join(".xoscal", "derogations.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(filepath.Dir(artifactPath), ".xoscal", "derogations.json")
	}
	store, _ := derogation.Load(path)
	return store
}

func synthesizeState(state *VectorState) {
	// Synthesize overall vector dimensions
	if state.Evidence == EvidenceTampered {
		state.Lifecycle = ModeQuarantined
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
	} else if state.AntiCount > 0 {
		state.Lifecycle = ModeFailed
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
	} else if state.DegradedCount > 0 || state.Evidence == EvidenceUnverified {
		state.Lifecycle = ModeDegraded
		state.Coherence = PartiallyCoherent
		state.Valence = ValenceMixed
	} else if state.DerogatedCount > 0 {
		state.Lifecycle = ModeSafeHold
		state.Coherence = PartiallyCoherent
		state.Valence = ValenceMixed
	} else {
		state.Lifecycle = ModeNormal
		state.Coherence = Coherent
		state.Valence = ValencePositive
	}

}

func evaluateControl(ctrlID, compTitle, desc string, state *VectorState, evidence *evidenceSource, store *derogation.Store) {
	finding := ControlFinding{ControlID: ctrlID, Component: compTitle, Status: "coherent", Valence: "POSITIVE", Description: desc}
	if strings.TrimSpace(desc) == "" {
		finding.Status, finding.Valence = "incomplete", "NEUTRAL"
		finding.Remediation = "Provide detailed implementation statement prose"
	} else if len(strings.TrimSpace(desc)) < 15 {
		finding.Status, finding.Valence = "degraded", "MIXED"
		finding.Remediation = "Implementation statement is overly terse; elaborate mechanism and verification"
	}
	if evidence != nil {
		evidence.evaluate(ctrlID, &finding, state)
	}
	applyDerogation(ctrlID, &finding, store)
	recordFinding(finding, state)
}

func applyDerogation(ctrlID string, finding *ControlFinding, store *derogation.Store) {
	if store == nil || finding.Status == "violation" {
		return
	}
	if active := store.GetActiveForControl(ctrlID, time.Now()); active != nil {
		finding.Status, finding.Valence = "derogated", "MIXED"
		finding.Remediation = fmt.Sprintf("Derogated by %s (TTL remaining: %s; justification: %s)",
			active.Authority, active.TimeRemaining(time.Now()).Round(time.Hour), active.Justification)
	} else if expired := store.GetExpiredForControl(ctrlID, time.Now()); expired != nil {
		finding.Status, finding.Valence = "expired_derogation", "NEGATIVE"
		finding.Remediation = fmt.Sprintf("CRITICAL: Derogation expired on %s! Authority %s must re-evaluate",
			expired.ExpiresAt.Format(time.RFC3339), expired.Authority)
	}
}

func recordFinding(finding ControlFinding, state *VectorState) {
	state.TotalControls++
	switch finding.Status {
	case "coherent":
		state.PassingCount++
	case "incomplete", "degraded":
		state.DegradedCount++
	case "derogated":
		state.DerogatedCount++
	case "violation", "expired_derogation":
		state.AntiCount++
	}
	state.Findings = append(state.Findings, finding)
}
