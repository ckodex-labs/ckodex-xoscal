package frameworks

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedCatalogs_SHA256ParityWithDisk(t *testing.T) {
	if len(manifestFrameworkSpecs) != 31 {
		t.Fatalf("expected 31 manifestFrameworkSpecs, got %d", len(manifestFrameworkSpecs))
	}

	for _, spec := range manifestFrameworkSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			diskPath := filepath.Join("..", "..", "..", "data", "frameworks", spec.ID, "catalog.json")
			diskBytes, err := os.ReadFile(diskPath)
			if err != nil {
				t.Fatalf("read disk catalog %s: %v", diskPath, err)
			}

			embBytes, err := loadEmbeddedCatalog(spec.ID)
			if err != nil {
				t.Fatalf("loadEmbeddedCatalog(%s): %v", spec.ID, err)
			}

			diskSum := sha256.Sum256(diskBytes)
			embSum := sha256.Sum256(embBytes)
			if diskSum != embSum || !bytes.Equal(diskBytes, embBytes) {
				t.Fatalf("embedded catalog SHA-256 mismatch for %s: disk=%x embedded=%x", spec.ID, diskSum, embSum)
			}
		})
	}
}

func TestEmbeddedCatalogs_StandaloneTempDirAll36Frameworks(t *testing.T) {
	t.Setenv("XOSCAL_DATA_DIR", "")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWorkDir := t.TempDir()
	if err := os.Chdir(tmpWorkDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// 1. Verify fresh uncached manifest registration loads all 31 frameworks from embedded archive.
	freshReg := NewRegistry()
	RegisterManifestFrameworks(freshReg)
	if got := len(freshReg.List()); got != 31 {
		t.Fatalf("expected 31 manifest frameworks in fresh registry, got %d", got)
	}
	for _, fw := range freshReg.List() {
		ctrls, err := LoadControls(fw)
		if err != nil {
			t.Fatalf("fresh registry LoadControls(%s) from temp cwd failed: %v", fw.ID(), err)
		}
		if len(ctrls) != fw.ControlCount() {
			t.Errorf("framework %s control count mismatch: LoadControls=%d vs ControlCount()=%d", fw.ID(), len(ctrls), fw.ControlCount())
		}
	}

	// 2. Verify DefaultRegistry() loads all 36 frameworks (31 manifest + 5 Canadian) from temp cwd.
	allFws := DefaultRegistry().List()
	if len(allFws) != 36 {
		t.Fatalf("expected 36 frameworks in DefaultRegistry (ensure canadianframeworks imported in package tests or check count), got %d", len(allFws))
	}
	for _, fw := range allFws {
		ctrls, err := LoadControls(fw)
		if err != nil {
			t.Fatalf("DefaultRegistry LoadControls(%s) from temp cwd failed: %v", fw.ID(), err)
		}
		if len(ctrls) == 0 {
			t.Errorf("expected non-empty controls for %s", fw.ID())
		}
	}
}

func TestEmbeddedCatalogs_FailClosedOnExplicitBrokenDataDir(t *testing.T) {
	emptyDataDir := t.TempDir()
	t.Setenv("XOSCAL_DATA_DIR", emptyDataDir)

	// Must fail closed even if DefaultRegistry() previously cached catalog bytes when XOSCAL_DATA_DIR was unset.
	fw, ok := DefaultRegistry().Get("nist-csf-2.0")
	if !ok {
		t.Fatalf("nist-csf-2.0 not registered")
	}

	if _, err := fw.CatalogJSON(); err == nil {
		t.Fatalf("expected CatalogJSON() to fail closed when XOSCAL_DATA_DIR points to empty directory")
	} else if !strings.Contains(err.Error(), "XOSCAL_DATA_DIR") {
		t.Fatalf("expected error mentioning XOSCAL_DATA_DIR, got: %v", err)
	}

	if _, err := LoadControls(fw); err == nil {
		t.Fatalf("expected LoadControls() to fail closed when XOSCAL_DATA_DIR points to empty directory")
	}
}

func TestJurisdictionMetadataAndNormalization(t *testing.T) {
	reg := DefaultRegistry()

	eits, ok := reg.Get("e-its-2024")
	if !ok {
		t.Fatalf("e-its-2024 not found")
	}
	if eits.Jurisdiction() != "ee" {
		t.Errorf("expected e-its-2024 jurisdiction 'ee', got %q", eits.Jurisdiction())
	}

	iso42001, ok := reg.Get("iso-iec-42001")
	if !ok {
		t.Fatalf("iso-iec-42001 not found")
	}
	if iso42001.Jurisdiction() != "international" {
		t.Errorf("expected iso-iec-42001 jurisdiction 'international', got %q", iso42001.Jurisdiction())
	}

	for _, alias := range []string{"ee", "est", "estonia", "ESTONIA"} {
		matches := reg.FindByJurisdiction(alias)
		if len(matches) != 1 || matches[0].ID() != "e-its-2024" {
			t.Errorf("FindByJurisdiction(%q) expected [e-its-2024], got %d matches", alias, len(matches))
		}
	}

	for _, alias := range []string{"int", "intl", "international", "global", "iso", "ISO"} {
		matches := reg.FindByJurisdiction(alias)
		if len(matches) != 1 || matches[0].ID() != "iso-iec-42001" {
			t.Errorf("FindByJurisdiction(%q) expected [iso-iec-42001], got %d matches", alias, len(matches))
		}
	}
}
