package frameworks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// Export exports a given framework to outDir/<framework-id>/catalog.json
// (and profile.json if present), validating each document against the official
// OSCAL 1.2.3 JSON schema and writing cryptographic SHA-256 sidecars.
func Export(outDir string, f Framework) error {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return fmt.Errorf("init schema validator: %w", err)
	}

	targetDir := filepath.Join(outDir, f.ID())
	// #nosec G703 -- targetDir is constructed beneath the operator-selected outDir
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("create destination directory %s: %w", targetDir, err)
	}

	// 1. Export & Validate Catalog
	catData, err := f.CatalogJSON()
	if err != nil {
		return fmt.Errorf("obtain catalog for %s: %w", f.ID(), err)
	}

	if err := v.Validate(catData, schemavalidate.KindCatalog); err != nil {
		return fmt.Errorf("validate catalog for %s against OSCAL 1.2.3 schema: %w", f.ID(), err)
	}

	catPath := filepath.Join(targetDir, "catalog.json")
	// #nosec G703,G304,G306 -- catPath is beneath operator-selected output directory
	if err := os.WriteFile(catPath, catData, 0o600); err != nil {
		return fmt.Errorf("write catalog %s: %w", catPath, err)
	}

	if _, err := dbutil.WriteDigestSidecar(catPath); err != nil {
		return fmt.Errorf("write catalog digest sidecar %s: %w", catPath, err)
	}

	// 2. Export & Validate Profile (if present)
	if f.HasProfile() {
		profData, err := f.ProfileJSON()
		if err != nil {
			return fmt.Errorf("obtain profile for %s: %w", f.ID(), err)
		}

		if err := v.Validate(profData, schemavalidate.KindProfile); err != nil {
			return fmt.Errorf("validate profile for %s against OSCAL 1.2.3 schema: %w", f.ID(), err)
		}

		profPath := filepath.Join(targetDir, "profile.json")
		// #nosec G703,G304,G306 -- profPath is beneath operator-selected output directory
		if err := os.WriteFile(profPath, profData, 0o600); err != nil {
			return fmt.Errorf("write profile %s: %w", profPath, err)
		}

		if _, err := dbutil.WriteDigestSidecar(profPath); err != nil {
			return fmt.Errorf("write profile digest sidecar %s: %w", profPath, err)
		}
	}

	return nil
}
