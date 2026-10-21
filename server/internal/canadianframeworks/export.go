package canadianframeworks

import (
	"github.com/mchorfa/xoscal/server/internal/frameworks"
)

// FrameworkInfo provides metadata about an authoritative Canadian cybersecurity framework.
type FrameworkInfo struct {
	RefID        string `json:"ref_id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Locale       string `json:"locale"`
	Provider     string `json:"provider"`
	ControlCount int    `json:"control_count"`
	HasProfile   bool   `json:"has_profile"`
}

// ListFrameworks returns metadata for all authoritative Canadian cybersecurity frameworks.
func ListFrameworks() []FrameworkInfo {
	fws := frameworks.DefaultRegistry().FindByJurisdiction("ca")
	res := make([]FrameworkInfo, len(fws))
	for i, f := range fws {
		res[i] = FrameworkInfo{
			RefID:        f.ID(),
			Name:         f.Title(),
			Version:      f.Version(),
			Locale:       f.Jurisdiction(),
			Provider:     f.Authority(),
			ControlCount: f.ControlCount(),
			HasProfile:   f.HasProfile(),
		}
	}
	return res
}

// IsCanadian returns true if the given refID corresponds to an authoritative Canadian framework.
func IsCanadian(refID string) bool {
	f, ok := frameworks.DefaultRegistry().Get(refID)
	return ok && f.Jurisdiction() == "ca"
}

// Export exports a specific Canadian framework to outDir/<refID>/catalog.json
// (and profile.json if present) with schema validation and SHA-256 sidecars.
func Export(outDir, refID string) error {
	return frameworks.DefaultRegistry().Export(outDir, refID)
}

// ExportAll exports all authoritative Canadian frameworks to outDir.
func ExportAll(outDir string) error {
	return frameworks.DefaultRegistry().ExportJurisdiction(outDir, "ca")
}
