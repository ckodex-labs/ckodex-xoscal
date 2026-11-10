package frameworks_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
)

func TestVerifyArtifactParity_ITSP10033(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-itsp-10-033")
	if !ok {
		t.Fatalf("framework cccs-itsp-10-033 not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-itsp-10-033", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
		for _, d := range rep.Discrepancies {
			t.Logf("Discrepancy: [%s] Control %s: %s", d.Aspect, d.ControlID, d.Message)
		}
	}

	if !rep.SidecarMatch {
		t.Errorf("expected sidecar match to be true")
	}
	if !rep.SchemaValid {
		t.Errorf("expected OSCAL 1.2.3 schema valid, got errors: %v", rep.SchemaErrors)
	}
	if !rep.IntegrityValid {
		t.Errorf("expected semantic integrity valid, got errors: %v", rep.IntegrityErrors)
	}
	if rep.TotalExpectedControls != 1248 {
		t.Errorf("expected 1248 controls, got %d", rep.TotalExpectedControls)
	}
	if rep.MatchedControls != 1248 {
		t.Errorf("expected 1248 matched controls, got %d", rep.MatchedControls)
	}
	if rep.ParityScore < 99.9 {
		t.Errorf("expected parity score ~100%%, got %.2f%%", rep.ParityScore)
	}

	// Format outputs
	term := rep.FormatTerminal(false)
	if !strings.Contains(term, "xOSCAL FRAMEWORK VERBATIM PARITY REPORT") {
		t.Errorf("FormatTerminal missing header: %s", term)
	}

	md := rep.FormatMarkdown()
	if !strings.Contains(md, "xOSCAL Framework Verbatim Parity Report") {
		t.Errorf("FormatMarkdown missing header: %s", md)
	}
}

func TestVerifyArtifactParity_CyberSecureCanada(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cybersecure-canada", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
		for _, d := range rep.Discrepancies {
			t.Logf("Discrepancy: [%s] Control %s: %s", d.Aspect, d.ControlID, d.Message)
		}
	}
	if rep.TotalExpectedControls != 46 {
		t.Errorf("expected 46 controls, got %d", rep.TotalExpectedControls)
	}
}

func TestVerifyArtifactParity_CCCS_ITSG33(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-itsg-33")
	if !ok {
		t.Fatalf("framework cccs-itsg-33 not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-itsg-33", "catalog.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
	}
	if rep.TotalExpectedControls != 234 {
		t.Errorf("expected 234 controls, got %d", rep.TotalExpectedControls)
	}
}

func TestVerifyArtifactParity_MediumCloudPBMM_Profile(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cccs-medium-cloud-pbmm")
	if !ok {
		t.Fatalf("framework cccs-medium-cloud-pbmm not found")
	}

	artifactPath := filepath.Join("..", "..", "..", "data", "frameworks", "cccs-medium-cloud-pbmm", "profile.json")
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		t.Skipf("skipping test: artifact %s does not exist", artifactPath)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, artifactPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity failed: %v", err)
	}

	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY for profile, got %s (discrepancies: %d)", rep.Verdict, len(rep.Discrepancies))
	}
	if rep.ArtifactKind != "profile" {
		t.Errorf("expected profile artifact kind, got %s", rep.ArtifactKind)
	}
}

func TestVerifyBytesParity_SyntheticDiscrepancies(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	baseBytes, err := fw.CatalogJSON()
	if err != nil {
		t.Fatalf("CatalogJSON: %v", err)
	}

	// 1. Verify exact in-memory bytes parity gives VERBATIM_PARITY
	rep, err := frameworks.VerifyBytesParity(fw, baseBytes, "memory://test.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity: %v", err)
	}
	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY, got %s", rep.Verdict)
	}

	// 2. Tampered Title
	var root map[string]interface{}
	if err := json.Unmarshal(baseBytes, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cat := root["catalog"].(map[string]interface{})
	groups := cat["groups"].([]interface{})
	firstGroup := groups[0].(map[string]interface{})
	ctrls := firstGroup["controls"].([]interface{})
	firstCtrl := ctrls[0].(map[string]interface{})
	firstCtrl["title"] = "Tampered Title That Does Not Match"

	tamperedBytes, _ := json.Marshal(root)
	repTampered, err := frameworks.VerifyBytesParity(fw, tamperedBytes, "memory://tampered.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity tampered: %v", err)
	}
	if repTampered.Verdict != frameworks.VerdictDeviationDetected {
		t.Errorf("expected DEVIATION_DETECTED on tampered title, got %s", repTampered.Verdict)
	}
	foundTitleDiscrepancy := false
	for _, d := range repTampered.Discrepancies {
		if d.Aspect == "Title" {
			foundTitleDiscrepancy = true
			break
		}
	}
	if !foundTitleDiscrepancy {
		t.Errorf("expected Title discrepancy, none found")
	}

	// 3. Missing control
	ctrlsRest := ctrls[1:]
	firstGroup["controls"] = ctrlsRest
	missingCtrlBytes, _ := json.Marshal(root)
	repMissing, err := frameworks.VerifyBytesParity(fw, missingCtrlBytes, "memory://missing.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity missing: %v", err)
	}
	if len(repMissing.MissingControls) == 0 {
		t.Errorf("expected MissingControls to have at least 1 item")
	}

	// 4. Placeholder title anti-invariant (e.g. "Control OC.1") and Unparsed ODP anti-invariant
	var rootAnti map[string]interface{}
	if err := json.Unmarshal(baseBytes, &rootAnti); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	catAnti := rootAnti["catalog"].(map[string]interface{})
	groupsAnti := catAnti["groups"].([]interface{})
	firstGroupAnti := groupsAnti[0].(map[string]interface{})
	ctrlsAnti := firstGroupAnti["controls"].([]interface{})
	firstCtrlAnti := ctrlsAnti[0].(map[string]interface{})
	firstCtrlAnti["title"] = "Control OC.1"
	partsAnti := firstCtrlAnti["parts"].([]interface{})
	firstPartAnti := partsAnti[0].(map[string]interface{})
	firstPartAnti["prose"] = "Organization defines [Assignment: organization-defined frequency] for review."

	antiBytes, _ := json.Marshal(rootAnti)
	repAnti, err := frameworks.VerifyBytesParity(fw, antiBytes, "memory://anti.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity anti: %v", err)
	}
	if repAnti.Verdict != frameworks.VerdictDeviationDetected {
		t.Errorf("expected DEVIATION_DETECTED on anti-invariants, got %s", repAnti.Verdict)
	}
	foundPlaceholder, foundODP := false, false
	for _, d := range repAnti.Discrepancies {
		if d.Aspect == "PlaceholderTitle" {
			foundPlaceholder = true
		}
		if d.Aspect == "UnparsedODP" {
			foundODP = true
		}
	}
	if !foundPlaceholder {
		t.Errorf("expected PlaceholderTitle discrepancy, none found")
	}
	if !foundODP {
		t.Errorf("expected UnparsedODP discrepancy, none found")
	}
}

func TestVerifyArtifactParity_SidecarMismatch(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	tmpDir := t.TempDir()
	catPath := filepath.Join(tmpDir, "catalog.json")
	sidecarPath := catPath + ".sha256"

	baseBytes, err := fw.CatalogJSON()
	if err != nil {
		t.Fatalf("CatalogJSON: %v", err)
	}

	if err := os.WriteFile(catPath, baseBytes, 0644); err != nil {
		t.Fatalf("WriteFile catPath: %v", err)
	}
	// Write wrong hash into sidecar
	wrongHash := hex.EncodeToString(sha256.New().Sum([]byte("corrupt")))
	if err := os.WriteFile(sidecarPath, []byte(wrongHash), 0644); err != nil {
		t.Fatalf("WriteFile sidecar: %v", err)
	}

	rep, err := frameworks.VerifyArtifactParity(fw, catPath)
	if err != nil {
		t.Fatalf("VerifyArtifactParity: %v", err)
	}
	if rep.SidecarMatch {
		t.Errorf("expected sidecar match to be false on mismatch")
	}
	if rep.Verdict != frameworks.VerdictDeviationDetected {
		t.Errorf("expected DEVIATION_DETECTED on sidecar mismatch, got %s", rep.Verdict)
	}
}

func TestVerifyAllFrameworksParity(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	baseDir := filepath.Join("..", "..", "..", "data", "frameworks")

	reports, err := frameworks.VerifyAllFrameworksParity(reg, baseDir)
	if err != nil {
		t.Fatalf("VerifyAllFrameworksParity failed: %v", err)
	}

	if len(reports) == 0 {
		t.Fatalf("expected parity reports, got 0")
	}

	for _, r := range reports {
		t.Logf("Framework %s (%s): Verdict=%s Score=%.1f%% SchemaValid=%t IntegrityValid=%t",
			r.FrameworkID, r.ArtifactKind, r.Verdict, r.ParityScore, r.SchemaValid, r.IntegrityValid)
		if r.Verdict != frameworks.VerdictVerbatimParity {
			t.Errorf("framework %s (%s) failed verbatim parity: %d discrepancies",
				r.FrameworkID, r.ArtifactKind, len(r.Discrepancies))
		}
	}
}

func TestBaselineInventoryAndTamperDetection(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fws := reg.List()
	if len(fws) != 36 {
		t.Fatalf("expected 36 registered frameworks, got %d", len(fws))
	}

	if os.Getenv("GENERATE_BASELINE_INVENTORY") == "1" {
		inv := frameworks.BaselineInventory{
			Frameworks: make(map[string]frameworks.FrameworkBaseline, len(fws)),
		}
		for _, f := range fws {
			b, err := frameworks.ComputeFrameworkBaseline(f)
			if err != nil {
				t.Fatalf("ComputeFrameworkBaseline(%s): %v", f.ID(), err)
			}
			inv.Frameworks[strings.ToLower(f.ID())] = b
		}
		outBytes, err := json.MarshalIndent(inv, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent: %v", err)
		}
		if err := os.WriteFile("baseline_inventory.json", append(outBytes, '\n'), 0o600); err != nil {
			t.Fatalf("WriteFile baseline_inventory.json: %v", err)
		}
		t.Logf("Generated baseline_inventory.json for %d frameworks", len(inv.Frameworks))
		return
	}

	for _, f := range fws {
		base, ok := frameworks.LookupBaseline(f.ID())
		if !ok {
			t.Fatalf("missing baseline inventory entry for %s", f.ID())
		}
		computed, err := frameworks.ComputeFrameworkBaseline(f)
		if err != nil {
			t.Fatalf("ComputeFrameworkBaseline(%s): %v", f.ID(), err)
		}
		if base.TotalControls != computed.TotalControls {
			t.Errorf("%s: baseline TotalControls=%d, computed=%d", f.ID(), base.TotalControls, computed.TotalControls)
		}
		if base.ControlsDigest != computed.ControlsDigest {
			t.Errorf("%s: baseline ControlsDigest=%s, computed=%s", f.ID(), base.ControlsDigest, computed.ControlsDigest)
		}
	}

	// Simulate a self-referential disk tampering attack where a Framework's CatalogJSON()
	// returns the same tampered bytes as the artifact on disk.
	realFw, ok := reg.Get("e-its-2024")
	if !ok {
		t.Fatalf("e-its-2024 not found")
	}
	origBytes, err := realFw.CatalogJSON()
	if err != nil {
		t.Fatalf("CatalogJSON: %v", err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(origBytes, &root); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	cat := root["catalog"].(map[string]interface{})
	if !tamperFirstControlTitle(cat, "Silent Disk Tampered Title") {
		t.Fatalf("could not find a control to tamper in e-its-2024")
	}
	tamperedBytes, err := json.Marshal(root)
	if err != nil {
		t.Fatalf("Marshal tampered: %v", err)
	}

	// Create a BaseFramework with ID "e-its-2024" whose CatalogJSON() returns tamperedBytes
	// (matching the tampered bytes passed to VerifyBytesParity). Without baseline_inventory.json,
	// compareControls would compare tampered vs tampered and pass!
	selfRefFw := frameworks.New(frameworks.Config{
		ID:           "e-its-2024",
		Title:        realFw.Title(),
		Version:      realFw.Version(),
		Jurisdiction: realFw.Jurisdiction(),
		ControlCount: realFw.ControlCount(),
		CatalogFn: func() ([]byte, error) {
			return tamperedBytes, nil
		},
	})

	rep, err := frameworks.VerifyBytesParity(selfRefFw, tamperedBytes, "memory://self-ref-tampered.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity: %v", err)
	}
	if rep.Verdict != frameworks.VerdictDeviationDetected {
		t.Fatalf("expected DEVIATION_DETECTED against independent baseline inventory, got %s", rep.Verdict)
	}
	foundBaselineDigest := false
	for _, d := range rep.Discrepancies {
		if d.Aspect == "BaselineControlDigest" {
			foundBaselineDigest = true
			break
		}
	}
	if !foundBaselineDigest {
		t.Errorf("expected BaselineControlDigest discrepancy, got %+v", rep.Discrepancies)
	}
}

func tamperFirstControlTitle(node map[string]interface{}, newTitle string) bool {
	if ctrls, ok := node["controls"].([]interface{}); ok && len(ctrls) > 0 {
		if c0, ok := ctrls[0].(map[string]interface{}); ok {
			c0["title"] = newTitle
			return true
		}
	}
	if groups, ok := node["groups"].([]interface{}); ok {
		for _, g := range groups {
			if gm, ok := g.(map[string]interface{}); ok {
				if tamperFirstControlTitle(gm, newTitle) {
					return true
				}
			}
		}
	}
	return false
}

func TestKGRoundTripParity(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	ctx := context.Background()

	// Verify full 7-layer Knowledge Graph -> oscal.Generator round-trip parity
	// across representative Canadian, US, European, and international frameworks.
	sampleIDs := []string{
		"cybersecure-canada",
		"cccs-itsg-33",
		"nist-csf-2.0",
		"iso-iec-42001",
		"cyber_essentials",
	}
	for _, id := range sampleIDs {
		id := id
		t.Run(id, func(t *testing.T) {
			fw, ok := reg.Get(id)
			if !ok {
				t.Fatalf("framework %s not found", id)
			}
			layers, err := frameworks.GenerateAllLayersFromKG(ctx, fw)
			if err != nil {
				t.Fatalf("GenerateAllLayersFromKG(%s): %v", id, err)
			}
			layerMap := map[string][]byte{
				"catalog.json":                       layers.Catalog,
				"profile.json":                       layers.Profile,
				"component-definition.json":          layers.ComponentDefinition,
				"system-security-plan.json":          layers.SystemSecurityPlan,
				"assessment-plan.json":               layers.AssessmentPlan,
				"assessment-results.json":            layers.AssessmentResults,
				"plan-of-action-and-milestones.json": layers.POAM,
			}
			for name, data := range layerMap {
				rep, err := frameworks.VerifyBytesParity(fw, data, "memory://"+id+"/"+name)
				if err != nil {
					t.Fatalf("VerifyBytesParity(%s/%s): %v", id, name, err)
				}
				if rep.Verdict != frameworks.VerdictVerbatimParity {
					t.Errorf("%s/%s: expected VERBATIM_PARITY, got %s (discrepancies: %+v, schemaErrors: %v, integrityErrors: %v)",
						id, name, rep.Verdict, rep.Discrepancies, rep.SchemaErrors, rep.IntegrityErrors)
				}
			}
		})
	}
}

func TestVerifyBytesParity_AP_AR_POAM_TamperDetection(t *testing.T) {
	reg := frameworks.DefaultRegistry()
	fw, ok := reg.Get("cybersecure-canada")
	if !ok {
		t.Fatalf("framework cybersecure-canada not found")
	}

	layers, err := frameworks.ProjectAllLayers(fw)
	if err != nil {
		t.Fatalf("ProjectAllLayers: %v", err)
	}

	// Baseline check: AP, AR, and POA&M must match all 46 controls verbatim.
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"assessment-plan.json", layers.AssessmentPlan},
		{"assessment-results.json", layers.AssessmentResults},
		{"plan-of-action-and-milestones.json", layers.POAM},
	} {
		rep, err := frameworks.VerifyBytesParity(fw, tc.data, "memory://"+tc.name)
		if err != nil {
			t.Fatalf("VerifyBytesParity(%s): %v", tc.name, err)
		}
		if rep.Verdict != frameworks.VerdictVerbatimParity {
			t.Fatalf("%s: expected VERBATIM_PARITY, got %s (%+v, schema=%v)", tc.name, rep.Verdict, rep.Discrepancies, rep.SchemaErrors)
		}
		if rep.TotalExpectedControls != 46 || rep.TotalActualControls != 46 || rep.MatchedControls != 46 {
			t.Fatalf("%s: expected 46/46/46 controls, got exp=%d act=%d matched=%d",
				tc.name, rep.TotalExpectedControls, rep.TotalActualControls, rep.MatchedControls)
		}
	}

	// 1. Assessment Plan: omit one control + tamper another control ID
	var apRoot map[string]interface{}
	if err := json.Unmarshal(layers.AssessmentPlan, &apRoot); err != nil {
		t.Fatalf("unmarshal AP: %v", err)
	}
	ap := apRoot["assessment-plan"].(map[string]interface{})
	apRC := ap["reviewed-controls"].(map[string]interface{})
	apSels := apRC["control-selections"].([]interface{})
	apSel0 := apSels[0].(map[string]interface{})
	apInc := apSel0["include-controls"].([]interface{})
	apInc[0].(map[string]interface{})["control-id"] = "tampered-ap-ctrl"
	apSel0["include-controls"] = apInc[:len(apInc)-1]
	apTamperedBytes, _ := json.Marshal(apRoot)

	apRep, err := frameworks.VerifyBytesParity(fw, apTamperedBytes, "memory://tampered-ap.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity tampered AP: %v", err)
	}
	if apRep.Verdict != frameworks.VerdictDeviationDetected {
		t.Fatalf("expected DEVIATION_DETECTED for tampered AP, got %s", apRep.Verdict)
	}
	if len(apRep.MissingControls) != 2 || len(apRep.ExtraControls) != 1 {
		t.Fatalf("expected 2 missing and 1 extra control in tampered AP, got missing=%v extra=%v", apRep.MissingControls, apRep.ExtraControls)
	}

	// 2. Assessment Results: omit one control from reviewed-controls
	var arRoot map[string]interface{}
	if err := json.Unmarshal(layers.AssessmentResults, &arRoot); err != nil {
		t.Fatalf("unmarshal AR: %v", err)
	}
	ar := arRoot["assessment-results"].(map[string]interface{})
	arRes := ar["results"].([]interface{})[0].(map[string]interface{})
	arRC := arRes["reviewed-controls"].(map[string]interface{})
	arSel0 := arRC["control-selections"].([]interface{})[0].(map[string]interface{})
	arInc := arSel0["include-controls"].([]interface{})
	arSel0["include-controls"] = arInc[1:]
	arTamperedBytes, _ := json.Marshal(arRoot)

	arRep, err := frameworks.VerifyBytesParity(fw, arTamperedBytes, "memory://tampered-ar.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity tampered AR: %v", err)
	}
	if arRep.Verdict != frameworks.VerdictDeviationDetected {
		t.Fatalf("expected DEVIATION_DETECTED for tampered AR, got %s", arRep.Verdict)
	}
	if len(arRep.MissingControls) != 1 {
		t.Fatalf("expected 1 missing control in tampered AR, got %v", arRep.MissingControls)
	}

	// 3. POA&M: omit one poam-item and tamper control-id prop on another
	var poamRoot map[string]interface{}
	if err := json.Unmarshal(layers.POAM, &poamRoot); err != nil {
		t.Fatalf("unmarshal POAM: %v", err)
	}
	poam := poamRoot["plan-of-action-and-milestones"].(map[string]interface{})
	items := poam["poam-items"].([]interface{})
	item0Props := items[0].(map[string]interface{})["props"].([]interface{})
	item0Props[0].(map[string]interface{})["value"] = "tampered-poam-ctrl"
	poam["poam-items"] = items[:len(items)-1]
	poamTamperedBytes, _ := json.Marshal(poamRoot)

	poamRep, err := frameworks.VerifyBytesParity(fw, poamTamperedBytes, "memory://tampered-poam.json")
	if err != nil {
		t.Fatalf("VerifyBytesParity tampered POAM: %v", err)
	}
	if poamRep.Verdict != frameworks.VerdictDeviationDetected {
		t.Fatalf("expected DEVIATION_DETECTED for tampered POAM, got %s", poamRep.Verdict)
	}
	if len(poamRep.MissingControls) != 2 || len(poamRep.ExtraControls) != 1 {
		t.Fatalf("expected 2 missing and 1 extra control in tampered POAM, got missing=%v extra=%v", poamRep.MissingControls, poamRep.ExtraControls)
	}
}
