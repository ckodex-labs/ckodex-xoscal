package vectorstate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/scaffold"
)

func TestEvaluateArtifact_ValidScaffold(t *testing.T) {
	tmpDir := t.TempDir()

	scan := &scaffold.ProjectScan{
		Name:    "demo-service",
		Version: "1.0.0",
		Components: []scaffold.ComponentCandidate{
			{
				Name:        "core",
				Type:        "software",
				Title:       "Demo Core",
				Description: "Core demo application",
				Controls: []scaffold.SuggestedControl{
					{ControlID: "ac-2", Description: "Account management configured via policy engine."},
					{ControlID: "sc-13", Description: "Cryptographic protection implemented via AES-256-GCM."},
				},
			},
		},
	}

	compDef := scaffold.GenerateComponentDefinition(scan, "nist-sp-800-53-rev5")
	jsonPath := filepath.Join(tmpDir, "component-definition.json")
	_, jsonBytes, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	_ = compDef
	_ = jsonBytes

	// Evaluate without evidence
	state, err := EvaluateArtifact(jsonPath, "")
	if err != nil {
		t.Fatalf("EvaluateArtifact failed: %v", err)
	}

	if state.Lifecycle != ModeNormal {
		t.Errorf("expected ModeNormal, got %s", state.Lifecycle)
	}
	if state.Coherence != Coherent {
		t.Errorf("expected Coherent, got %s", state.Coherence)
	}
	if state.TotalControls == 0 {
		t.Errorf("expected controls evaluated")
	}

	table := state.FormatTable(false)
	if table == "" {
		t.Errorf("expected non-empty table output")
	}
}

func TestEvaluateArtifact_EvidenceTamperDetection(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	evidenceDir := filepath.Join(tmpDir, "evidence")
	_ = os.MkdirAll(evidenceDir, 0750)

	evPath := filepath.Join(evidenceDir, "evidence-ac-2.log")
	_ = os.WriteFile(evPath, []byte("valid audit log bytes"), 0600)

	tamperedHash := hex.EncodeToString([]byte("00000000000000000000000000000000"))
	_ = os.WriteFile(evPath+".sha256", []byte(tamperedHash), 0600)

	jsonPath := filepath.Join(tmpDir, "component-definition.json")
	state, err := EvaluateArtifact(jsonPath, evidenceDir)
	if err != nil {
		t.Fatalf("EvaluateArtifact failed: %v", err)
	}

	if state.AntiCount == 0 {
		t.Fatalf("expected tamper detection to increment AntiCount")
	}
	if state.Lifecycle != ModeQuarantined {
		t.Errorf("expected ModeQuarantined on tamper, got %s", state.Lifecycle)
	}
	if state.Evidence != EvidenceTampered {
		t.Errorf("expected EvidenceTampered, got %s", state.Evidence)
	}

	realSum := sha256.Sum256([]byte("valid audit log bytes"))
	_ = os.WriteFile(evPath+".sha256", []byte(hex.EncodeToString(realSum[:])), 0600)

	cleanState, err := EvaluateArtifact(jsonPath, evidenceDir)
	if err != nil {
		t.Fatalf("clean EvaluateArtifact: %v", err)
	}
	if cleanState.AntiCount != 0 {
		t.Errorf("expected 0 anti count with valid hash, got %d", cleanState.AntiCount)
	}
}

func TestEvaluateArtifact_DerogationLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	_, _, err := scaffold.ScaffoldWorkspace(tmpDir, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, "component-definition.json")
	derogPath := filepath.Join(tmpDir, ".xoscal", "derogations.json")

	// Create an active derogation for an incomplete/degraded control
	store, _ := derogation.Load(derogPath)
	_ = store.Add(derogation.DerogationEntry{
		ControlID:            "ac-2",
		Scope:                "service:core",
		Authority:            "urn:xoscal:authority:ciso-alice",
		Justification:        "Pending hardware security token rollout",
		CompensatingControls: []string{"sc-7 boundary enforcement"},
		ExpiresAt:            time.Now().Add(7 * 24 * time.Hour),
	})
	if err := store.Save(derogPath); err != nil {
		t.Fatalf("save derogations: %v", err)
	}

	state, err := EvaluateArtifact(jsonPath, "")
	if err != nil {
		t.Fatalf("EvaluateArtifact failed: %v", err)
	}

	if state.DerogatedCount != 1 {
		t.Errorf("expected 1 derogated control, got %d", state.DerogatedCount)
	}
	if state.Lifecycle != ModeSafeHold {
		t.Errorf("expected ModeSafeHold for active derogation, got %s", state.Lifecycle)
	}

	// Now expire the derogation
	store.Entries[0].ExpiresAt = time.Now().Add(-1 * time.Hour)
	_ = store.Save(derogPath)

	expiredState, err := EvaluateArtifact(jsonPath, "")
	if err != nil {
		t.Fatalf("expired EvaluateArtifact failed: %v", err)
	}

	if expiredState.AntiCount == 0 {
		t.Errorf("expected anti-conflict for expired derogation")
	}
	if expiredState.Lifecycle != ModeFailed {
		t.Errorf("expected ModeFailed on expired derogation, got %s", expiredState.Lifecycle)
	}
}

func TestEvaluateArtifact_Catalog(t *testing.T) {
	tmpDir := t.TempDir()
	catPath := filepath.Join(tmpDir, "catalog.json")

	cat := canadianframeworks.BuildCyberSecureCanadaCatalog()
	catData, err := oscal.ExportCatalogJSON(cat)
	if err != nil {
		t.Fatalf("export catalog: %v", err)
	}

	if err := os.WriteFile(catPath, catData, 0600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	state, err := EvaluateArtifact(catPath, "")
	if err != nil {
		t.Fatalf("EvaluateArtifact(catalog) failed: %v", err)
	}

	if state.ArtifactKind != "catalog" {
		t.Errorf("expected ArtifactKind 'catalog', got %s", state.ArtifactKind)
	}
	if state.TotalControls != 46 {
		t.Errorf("expected 46 controls evaluated, got %d", state.TotalControls)
	}
	if state.Lifecycle != ModeNormal {
		t.Errorf("expected ModeNormal, got %s", state.Lifecycle)
	}
}

func TestEvaluateArtifact_SSP(t *testing.T) {
	tmpDir := t.TempDir()
	sspPath := filepath.Join(tmpDir, "ssp.json")

	validUUID := "6ba7b810-9dad-51d1-80b4-00c04fd430c8"
	ssp := &sspv1.SystemSecurityPlan{
		Uuid: &commonv1.UUID{Value: validUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Conformance SSP",
			Version: "1.0",
		},
		ImportProfile: &sspv1.ImportProfile{
			Href: &commonv1.URIReference{Value: "https://example.gov/profile.json"},
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds:   []*sspv1.SystemId{{Id: "test-system"}},
			SystemName:  "Test System",
			Description: &commonv1.MarkupMultiline{Value: "A test system description."},
			SystemInformation: &sspv1.SystemInformation{
				InformationTypes: []*sspv1.InformationType{{
					Uuid:        &commonv1.UUID{Value: validUUID},
					Title:       &commonv1.MarkupLine{Value: "Test info type"},
					Description: &commonv1.MarkupMultiline{Value: "Test information type"},
				}},
			},
			Status:                &sspv1.Status{State: "operational"},
			AuthorizationBoundary: &sspv1.AuthorizationBoundary{Description: &commonv1.MarkupMultiline{Value: "Boundary"}},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Users: []*sspv1.SystemUser{{
				Uuid:  &commonv1.UUID{Value: validUUID},
				Title: "Admin",
			}},
			Components: []*sspv1.SystemComponent{{
				Uuid:        &commonv1.UUID{Value: validUUID},
				Type:        "software",
				Title:       "App",
				Description: "App component",
				Status:      &sspv1.Status{State: "operational"},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			Description: &commonv1.MarkupMultiline{Value: "Implementation description"},
			ImplementedRequirements: []*sspv1.ImplementedRequirement{{
				Uuid:      &commonv1.UUID{Value: validUUID},
				ControlId: &commonv1.Token{Value: "ac-1"},
				Remarks:   []*commonv1.MarkupMultiline{{Value: "Access control policy is fully documented and maintained."}},
			}},
		},
	}

	sspData, err := oscal.ExportSSPJSON(ssp)
	if err != nil {
		t.Fatalf("ExportSSPJSON: %v", err)
	}

	if err := os.WriteFile(sspPath, sspData, 0600); err != nil {
		t.Fatalf("write ssp: %v", err)
	}

	state, err := EvaluateArtifact(sspPath, "")
	if err != nil {
		t.Fatalf("EvaluateArtifact(ssp) failed: %v", err)
	}

	if state.ArtifactKind != "ssp" {
		t.Errorf("expected ArtifactKind 'ssp', got %s", state.ArtifactKind)
	}
	if state.TotalControls != 1 {
		t.Errorf("expected 1 control evaluated, got %d", state.TotalControls)
	}
	if state.PassingCount != 1 {
		t.Errorf("expected 1 passing control, got %d", state.PassingCount)
	}
	if state.Lifecycle != ModeNormal {
		t.Errorf("expected ModeNormal, got %s", state.Lifecycle)
	}
}
