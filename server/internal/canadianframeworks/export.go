package canadianframeworks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
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
	return []FrameworkInfo{
		{
			RefID:        "cccs-itsg-33",
			Name:         "CCCS ITSG-33 IT Security Risk Management: A Lifecycle Approach (Annex 3)",
			Version:      "1.2.3",
			Locale:       "ca",
			Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
			ControlCount: 18,
			HasProfile:   false,
		},
		{
			RefID:        "cybersecure-canada",
			Name:         "CyberSecure Canada (CAN/DGSI 104)",
			Version:      "CAN/DGSI 104",
			Locale:       "ca",
			Provider:     "Innovation, Science and Economic Development Canada (ISED) / CCCS",
			ControlCount: 46,
			HasProfile:   false,
		},
		{
			RefID:        "cccs-itsp-10-171",
			Name:         "CCCS ITSP.10.171 Security Requirements for Contracting with the Government of Canada",
			Version:      "1.2.3",
			Locale:       "ca",
			Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
			ControlCount: 267,
			HasProfile:   false,
		},
		{
			RefID:        "cccs-medium-cloud-pbmm",
			Name:         "CCCS Medium Cloud PBMM Baseline (ITSP.50.103 / ITSG-33 Annex 4A Profile 1)",
			Version:      "ITSP.50.103",
			Locale:       "ca",
			Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
			ControlCount: 30,
			HasProfile:   true,
		},
	}
}

// IsCanadian returns true if the given refID corresponds to an authoritative Canadian framework.
func IsCanadian(refID string) bool {
	switch refID {
	case "cccs-itsg-33", "cybersecure-canada", "cccs-itsp-10-171", "itsp.10.171", "cccs-medium-cloud-pbmm":
		return true
	default:
		return false
	}
}

// Export exports a specific Canadian framework to outDir/<refID>/catalog.json
// (and profile.json for PBMM) with schema validation and SHA-256 sidecars.
func Export(outDir, refID string) error {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return fmt.Errorf("init validator: %w", err)
	}

	type target struct {
		filename string
		kind     schemavalidate.ArtifactKind
		exportFn func() ([]byte, error)
	}

	var targets []target
	canonRef := refID
	if canonRef == "itsp.10.171" {
		canonRef = "cccs-itsp-10-171"
	}

	switch canonRef {
	case "cccs-itsg-33":
		targets = append(targets, target{
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCCCSITSG33Catalog())
			},
		})
	case "cybersecure-canada":
		targets = append(targets, target{
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCyberSecureCanadaCatalog())
			},
		})
	case "cccs-itsp-10-171":
		targets = append(targets, target{
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCCCSITSP10171Catalog())
			},
		})
	case "cccs-medium-cloud-pbmm":
		targets = append(targets, target{
			filename: "profile.json",
			kind:     schemavalidate.KindProfile,
			exportFn: func() ([]byte, error) {
				return oscal.ExportProfileJSON(BuildCCCSMediumCloudPBMMProfile())
			},
		})
		targets = append(targets, target{
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCCCSMediumCloudPBMMCatalog())
			},
		})
	default:
		return fmt.Errorf("unsupported Canadian framework: %s", refID)
	}

	targetDir := filepath.Join(outDir, canonRef)
	// #nosec G301 -- output directory permissions
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("create dir %s: %w", targetDir, err)
	}

	for _, t := range targets {
		data, err := t.exportFn()
		if err != nil {
			return fmt.Errorf("export %s/%s: %w", canonRef, t.filename, err)
		}

		if err := v.Validate(data, t.kind); err != nil {
			return fmt.Errorf("validate %s/%s failed schema preflight: %w", canonRef, t.filename, err)
		}

		targetFile := filepath.Join(targetDir, t.filename)
		// #nosec G306 -- file write permissions
		if err := os.WriteFile(targetFile, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", targetFile, err)
		}

		if _, err := dbutil.WriteDigestSidecar(targetFile); err != nil {
			return fmt.Errorf("write sidecar for %s: %w", targetFile, err)
		}
	}

	return nil
}

// ExportAll exports all Canadian cybersecurity frameworks as OSCAL 1.2.3 JSON files
// with SHA-256 digest sidecars into the specified output directory.
// Each artifact is validated against the official OSCAL JSON schema before writing.
func ExportAll(outDir string) error {
	for _, info := range ListFrameworks() {
		if err := Export(outDir, info.RefID); err != nil {
			return err
		}
	}
	return nil
}
