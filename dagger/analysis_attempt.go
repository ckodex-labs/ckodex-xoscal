package main

import (
	"encoding/json"

	"dagger/xoscal/attemptpolicy"
	"dagger/xoscal/internal/dagger"
)

// WithAnalysisAttempt selects a new external-evidence observation without
// rebuilding payloads, retrying failures or changing any admission policy.
func (m *Xoscal) WithAnalysisAttempt(attempt string) (*Xoscal, error) {
	if err := attemptpolicy.Validate(attempt); err != nil {
		return nil, err
	}
	copy := *m
	copy.AnalysisAttempt = attempt
	return &copy, nil
}

// Key only external observations, after immutable tool installation. An empty
// default preserves the existing graph. Prior reports retain their own bytes.
func (m *Xoscal) analysisAttempt(c *dagger.Container, purpose string) *dagger.Container {
	if m.AnalysisAttempt == "" {
		return c
	}
	record := struct {
		Version int    `json:"version"`
		Attempt string `json:"attempt"`
		Purpose string `json:"purpose"`
		Role    string `json:"role"`
	}{1, m.AnalysisAttempt, purpose, "external evidence observation; does not grant admission"}
	// This concrete record contains only integers and strings: marshaling cannot fail.
	body, _ := json.Marshal(record)
	return c.WithEnvVariable("XOSCAL_ANALYSIS_ATTEMPT", m.AnalysisAttempt).
		WithNewFile("/evidence/analysis-attempt.json", string(body)+"\n")
}
