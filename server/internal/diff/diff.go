package diff

import (
	"fmt"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

// ChangeType classifies the modification to a control implementation.
type ChangeType string

const (
	ChangeAdded     ChangeType = "ADDED"
	ChangeRemoved   ChangeType = "REMOVED"
	ChangeModified  ChangeType = "MODIFIED"
	ChangeUnchanged ChangeType = "UNCHANGED"
)

// ControlDelta represents the difference for a single control between two artifacts.
type ControlDelta struct {
	ControlID  string     `json:"control_id"`
	Component  string     `json:"component"`
	ChangeType ChangeType `json:"change_type"`
	BaseProse  string     `json:"base_prose,omitempty"`
	HeadProse  string     `json:"head_prose,omitempty"`
	BaseStatus string     `json:"base_status,omitempty"`
	HeadStatus string     `json:"head_status,omitempty"`
	Regression bool       `json:"regression"`
}

// ComplianceDiff captures the complete semantic difference between two compliance baselines.
type ComplianceDiff struct {
	BaseArtifact  string                   `json:"base_artifact"`
	HeadArtifact  string                   `json:"head_artifact"`
	BaseVector    *vectorstate.VectorState `json:"base_vector"`
	HeadVector    *vectorstate.VectorState `json:"head_vector"`
	Deltas        []ControlDelta           `json:"deltas"`
	HasRegression bool                     `json:"has_regression"`
	AddedCount    int                      `json:"added_count"`
	RemovedCount  int                      `json:"removed_count"`
	ModifiedCount int                      `json:"modified_count"`
}

// CompareArtifacts evaluates and diffs two OSCAL JSON artifacts.
func CompareArtifacts(basePath, headPath string) (*ComplianceDiff, error) {
	baseState, err := vectorstate.EvaluateArtifact(basePath, "")
	if err != nil {
		return nil, fmt.Errorf("evaluate base artifact %s: %w", basePath, err)
	}

	headState, err := vectorstate.EvaluateArtifact(headPath, "")
	if err != nil {
		return nil, fmt.Errorf("evaluate head artifact %s: %w", headPath, err)
	}

	diff := &ComplianceDiff{
		BaseArtifact: basePath,
		HeadArtifact: headPath,
		BaseVector:   baseState,
		HeadVector:   headState,
		Deltas:       make([]ControlDelta, 0),
	}

	baseMap := make(map[string]vectorstate.ControlFinding)
	for _, f := range baseState.Findings {
		key := fmt.Sprintf("%s:%s", f.Component, f.ControlID)
		baseMap[key] = f
	}

	headMap := make(map[string]vectorstate.ControlFinding)
	for _, f := range headState.Findings {
		key := fmt.Sprintf("%s:%s", f.Component, f.ControlID)
		headMap[key] = f
	}

	// Check head controls against base
	for key, headFinding := range headMap {
		baseFinding, exists := baseMap[key]
		if !exists {
			diff.AddedCount++
			diff.Deltas = append(diff.Deltas, ControlDelta{
				ControlID:  headFinding.ControlID,
				Component:  headFinding.Component,
				ChangeType: ChangeAdded,
				HeadProse:  headFinding.Description,
				HeadStatus: headFinding.Status,
			})
			continue
		}

		// Both exist: check for changes
		proseChanged := strings.TrimSpace(baseFinding.Description) != strings.TrimSpace(headFinding.Description)
		statusChanged := baseFinding.Status != headFinding.Status

		isRegression := false
		if isStatusWorse(baseFinding.Status, headFinding.Status) {
			isRegression = true
			diff.HasRegression = true
		}

		if proseChanged || statusChanged {
			diff.ModifiedCount++
			diff.Deltas = append(diff.Deltas, ControlDelta{
				ControlID:  headFinding.ControlID,
				Component:  headFinding.Component,
				ChangeType: ChangeModified,
				BaseProse:  baseFinding.Description,
				HeadProse:  headFinding.Description,
				BaseStatus: baseFinding.Status,
				HeadStatus: headFinding.Status,
				Regression: isRegression,
			})
		}
	}

	// Check for removed controls
	for key, baseFinding := range baseMap {
		if _, exists := headMap[key]; !exists {
			diff.RemovedCount++
			diff.HasRegression = true // Dropping a control is considered a regression
			diff.Deltas = append(diff.Deltas, ControlDelta{
				ControlID:  baseFinding.ControlID,
				Component:  baseFinding.Component,
				ChangeType: ChangeRemoved,
				BaseProse:  baseFinding.Description,
				BaseStatus: baseFinding.Status,
				Regression: true,
			})
		}
	}

	// Vector State regressions
	if headState.AntiCount > baseState.AntiCount {
		diff.HasRegression = true
	}
	if headState.Lifecycle == vectorstate.ModeFailed && baseState.Lifecycle != vectorstate.ModeFailed {
		diff.HasRegression = true
	}

	return diff, nil
}

func isStatusWorse(oldStatus, newStatus string) bool {
	rank := map[string]int{
		"coherent":           4,
		"derogated":          3,
		"degraded":           2,
		"incomplete":         1,
		"expired_derogation": 0,
		"violation":          0,
	}
	return rank[newStatus] < rank[oldStatus]
}
