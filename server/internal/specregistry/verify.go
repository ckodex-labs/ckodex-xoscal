package specregistry

// ModelStatus is the per-model outcome of a verify run.
type ModelStatus struct {
	Model    string `json:"model"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Status   string `json:"status"` // "match" | "drift"
}

// VerifyResult aggregates per-model statuses for one OSCAL version.
type VerifyResult struct {
	OSCALVersion string        `json:"oscal_version"`
	Drift        bool          `json:"drift"`
	Models       []ModelStatus `json:"models"`
}

// CompareModel hashes the fetched schema and compares it to the expected hash.
func CompareModel(model, expected string, fetched []byte) ModelStatus {
	actual := Hash(fetched)
	status := "drift"
	if actual == expected {
		status = "match"
	}
	return ModelStatus{Model: model, Expected: expected, Actual: actual, Status: status}
}

// Aggregate folds per-model statuses into a result; Drift is true if any drifted.
func Aggregate(oscalVersion string, statuses []ModelStatus) VerifyResult {
	drift := false
	for _, s := range statuses {
		if s.Status == "drift" {
			drift = true
		}
	}
	return VerifyResult{OSCALVersion: oscalVersion, Drift: drift, Models: statuses}
}

// ExitCode maps a result to the verify exit-code contract (0 ok, 3 drift).
func (v VerifyResult) ExitCode() int {
	if v.Drift {
		return 3
	}
	return 0
}
