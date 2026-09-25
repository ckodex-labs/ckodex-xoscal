package canadianframeworks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func TestCanadianFrameworksSchemaConformance(t *testing.T) {
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("schemavalidate.NewValidator() failed: %v", err)
	}

	t.Run("CCCS_ITSG_33_Catalog", func(t *testing.T) {
		cat := BuildCCCSITSG33Catalog()
		data, err := oscal.ExportCatalogJSON(cat)
		if err != nil {
			t.Fatalf("ExportCatalogJSON failed: %v", err)
		}
		if err := v.Validate(data, schemavalidate.KindCatalog); err != nil {
			t.Fatalf("CCCS ITSG-33 Catalog failed OSCAL 1.2.3 schema validation: %v", err)
		}
		if len(cat.Groups) == 0 {
			t.Fatalf("expected groups in ITSG-33 catalog, got 0")
		}
	})

	t.Run("CCCS_Medium_Cloud_PBMM_Profile", func(t *testing.T) {
		prof := BuildCCCSMediumCloudPBMMProfile()
		data, err := oscal.ExportProfileJSON(prof)
		if err != nil {
			t.Fatalf("ExportProfileJSON failed: %v", err)
		}
		if err := v.Validate(data, schemavalidate.KindProfile); err != nil {
			t.Fatalf("CCCS Medium Cloud PBMM Profile failed OSCAL 1.2.3 schema validation: %v", err)
		}
		if len(prof.Imports) == 0 {
			t.Fatalf("expected imports in PBMM profile, got 0")
		}
	})

	t.Run("CyberSecure_Canada_Catalog", func(t *testing.T) {
		cat := BuildCyberSecureCanadaCatalog()
		data, err := oscal.ExportCatalogJSON(cat)
		if err != nil {
			t.Fatalf("ExportCatalogJSON failed: %v", err)
		}
		if err := v.Validate(data, schemavalidate.KindCatalog); err != nil {
			t.Fatalf("CyberSecure Canada Catalog failed OSCAL 1.2.3 schema validation: %v", err)
		}
		if len(cat.Controls) != 13 {
			t.Fatalf("expected 13 baseline controls in CyberSecure Canada catalog, got %d", len(cat.Controls))
		}
	})

	t.Run("CCCS_ITSP_10_171_Catalog", func(t *testing.T) {
		cat := BuildCCCSITSP10171Catalog()
		data, err := oscal.ExportCatalogJSON(cat)
		if err != nil {
			t.Fatalf("ExportCatalogJSON failed: %v", err)
		}
		if err := v.Validate(data, schemavalidate.KindCatalog); err != nil {
			t.Fatalf("CCCS ITSP.10.171 Catalog failed OSCAL 1.2.3 schema validation: %v", err)
		}
		if len(cat.Controls) == 0 {
			t.Fatalf("expected controls in ITSP.10.171 catalog, got 0")
		}
	})
}

func TestExportAll(t *testing.T) {
	tmpDir := t.TempDir()

	if err := ExportAll(tmpDir); err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	targets := []struct {
		subdir   string
		filename string
	}{
		{"cccs-itsg-33", "catalog.json"},
		{"cccs-medium-cloud-pbmm", "profile.json"},
		{"cybersecure-canada", "catalog.json"},
		{"cccs-itsp-10-171", "catalog.json"},
	}

	for _, tgt := range targets {
		fPath := filepath.Join(tmpDir, tgt.subdir, tgt.filename)
		if _, err := os.Stat(fPath); err != nil {
			t.Errorf("file missing: %s", fPath)
		}
		if err := dbutil.CheckDigestSidecar(fPath); err != nil {
			t.Errorf("digest check failed for %s: %v", fPath, err)
		}
	}
}

