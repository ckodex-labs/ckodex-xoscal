package frameworks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// Export exports all 7 layers of OSCAL 1.2.3 artifacts for a given framework
// to outDir/<framework-id>/, validating each document against the official
// NIST OSCAL 1.2.3 JSON schema and writing cryptographic SHA-256 sidecars.
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

	layers, err := ProjectAllLayers(f)
	if err != nil {
		return fmt.Errorf("project all layers for %s: %w", f.ID(), err)
	}

	type layerArtifact struct {
		name string
		data []byte
		kind schemavalidate.ArtifactKind
	}

	artifacts := []layerArtifact{
		{"catalog.json", layers.Catalog, schemavalidate.KindCatalog},
		{"profile.json", layers.Profile, schemavalidate.KindProfile},
		{"component-definition.json", layers.ComponentDefinition, schemavalidate.KindComponentDefinition},
		{"system-security-plan.json", layers.SystemSecurityPlan, schemavalidate.KindSSP},
		{"assessment-plan.json", layers.AssessmentPlan, schemavalidate.KindAssessmentPlan},
		{"assessment-results.json", layers.AssessmentResults, schemavalidate.KindAssessmentResults},
		{"plan-of-action-and-milestones.json", layers.POAM, schemavalidate.KindPOAM},
	}

	for _, art := range artifacts {
		if err := v.Validate(art.data, art.kind); err != nil {
			return fmt.Errorf("validate %s for %s against OSCAL 1.2.3 schema: %w", art.name, f.ID(), err)
		}

		filePath := filepath.Join(targetDir, art.name)
		// #nosec G703,G304,G306 -- filePath is beneath operator-selected output directory
		if err := os.WriteFile(filePath, art.data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", filePath, err)
		}

		if _, err := dbutil.WriteDigestSidecar(filePath); err != nil {
			return fmt.Errorf("write digest sidecar for %s: %w", filePath, err)
		}
	}

	return nil
}
