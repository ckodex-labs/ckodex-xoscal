package canadianframeworks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// ExportAll exports all Canadian cybersecurity frameworks as OSCAL 1.2.3 JSON files
// with SHA-256 digest sidecars into the specified output directory.
// Each artifact is validated against the official OSCAL JSON schema before writing.
func ExportAll(outDir string) error {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return fmt.Errorf("init validator: %w", err)
	}

	targets := []struct {
		subdir   string
		filename string
		kind     schemavalidate.ArtifactKind
		exportFn func() ([]byte, error)
	}{
		{
			subdir:   "cccs-itsg-33",
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCCCSITSG33Catalog())
			},
		},
		{
			subdir:   "cccs-medium-cloud-pbmm",
			filename: "profile.json",
			kind:     schemavalidate.KindProfile,
			exportFn: func() ([]byte, error) {
				return oscal.ExportProfileJSON(BuildCCCSMediumCloudPBMMProfile())
			},
		},
		{
			subdir:   "cybersecure-canada",
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCyberSecureCanadaCatalog())
			},
		},
		{
			subdir:   "cccs-itsp-10-171",
			filename: "catalog.json",
			kind:     schemavalidate.KindCatalog,
			exportFn: func() ([]byte, error) {
				return oscal.ExportCatalogJSON(BuildCCCSITSP10171Catalog())
			},
		},
	}

	for _, t := range targets {
		data, err := t.exportFn()
		if err != nil {
			return fmt.Errorf("export %s: %w", t.subdir, err)
		}

		if err := v.Validate(data, t.kind); err != nil {
			return fmt.Errorf("validate %s failed schema preflight: %w", t.subdir, err)
		}

		targetDir := filepath.Join(outDir, t.subdir)
		// #nosec G301 -- output directory permissions
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			return fmt.Errorf("create dir %s: %w", targetDir, err)
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
