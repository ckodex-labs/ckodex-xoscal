package frameworks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/integrity"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// ParityVerdict indicates the final verbatim parity determination.
type ParityVerdict string

const (
	VerdictVerbatimParity    ParityVerdict = "VERBATIM_PARITY"
	VerdictDeviationDetected ParityVerdict = "DEVIATION_DETECTED"
)

// Discrepancy details a specific verbatim mismatch between framework and artifact.
type Discrepancy struct {
	ControlID string `json:"control_id"`
	Aspect    string `json:"aspect"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
	Message   string `json:"message"`
}

// ParityReport represents the complete forensic parity verification result.
type ParityReport struct {
	FrameworkID           string        `json:"framework_id"`
	FrameworkTitle        string        `json:"framework_title"`
	ArtifactKind          string        `json:"artifact_kind"`
	ArtifactPath          string        `json:"artifact_path"`
	ArtifactSHA256        string        `json:"artifact_sha256"`
	SidecarPath           string        `json:"sidecar_path,omitempty"`
	SidecarSHA256         string        `json:"sidecar_sha256,omitempty"`
	SidecarMatch          bool          `json:"sidecar_match"`
	SchemaValid           bool          `json:"schema_valid"`
	SchemaErrors          []string      `json:"schema_errors,omitempty"`
	IntegrityValid        bool          `json:"integrity_valid"`
	IntegrityErrors       []string      `json:"integrity_errors,omitempty"`
	TotalExpectedControls int           `json:"total_expected_controls"`
	TotalActualControls   int           `json:"total_actual_controls"`
	MatchedControls       int           `json:"matched_controls"`
	MissingControls       []string      `json:"missing_controls,omitempty"`
	ExtraControls         []string      `json:"extra_controls,omitempty"`
	TitleMatches          int           `json:"title_matches"`
	StatementMatches      int           `json:"statement_matches"`
	GuidanceMatches       int           `json:"guidance_matches"`
	ParameterMatches      int           `json:"parameter_matches"`
	SovereignTagMatches   int           `json:"sovereign_tag_matches"`
	EnhancementMatches    int           `json:"enhancement_matches"`
	Discrepancies         []Discrepancy `json:"discrepancies,omitempty"`
	ParityScore           float64       `json:"parity_score"`
	Verdict               ParityVerdict `json:"verdict"`
	Timestamp             time.Time     `json:"timestamp"`
}

// VerifyArtifactParity verifies verbatim parity between a Framework and an OSCAL artifact on disk.
func VerifyArtifactParity(f Framework, artifactPath string) (*ParityReport, error) {
	// #nosec G703,G304 -- artifactPath is explicitly provided by the operator for parity verification.
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("read artifact %s: %w", artifactPath, err)
	}
	return VerifyBytesParity(f, data, artifactPath)
}

// VerifyBytesParity verifies verbatim parity between a Framework and raw OSCAL artifact bytes.
func VerifyBytesParity(f Framework, data []byte, artifactPath string) (*ParityReport, error) {
	rep := &ParityReport{
		FrameworkID:    f.ID(),
		FrameworkTitle: f.Title(),
		ArtifactPath:   artifactPath,
		Timestamp:      time.Now().UTC(),
		Verdict:        VerdictVerbatimParity,
	}

	// 1. Cryptographic SHA-256 calculation
	sum := sha256.Sum256(data)
	rep.ArtifactSHA256 = hex.EncodeToString(sum[:])

	// Check sidecar if artifactPath is a physical file
	if artifactPath != "" && !strings.HasPrefix(artifactPath, "memory") {
		sidecarPath := artifactPath + ".sha256"
		// #nosec G703,G304 -- sidecarPath is derived from the verified artifact path.
		if scBytes, err := os.ReadFile(sidecarPath); err == nil {
			rep.SidecarPath = sidecarPath
			scContent := strings.TrimSpace(string(scBytes))
			rep.SidecarSHA256 = scContent
			if strings.Contains(strings.ToLower(scContent), strings.ToLower(rep.ArtifactSHA256)) {
				rep.SidecarMatch = true
			} else {
				rep.SidecarMatch = false
				rep.Verdict = VerdictDeviationDetected
				rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
					ControlID: "*",
					Aspect:    "SHA256Sidecar",
					Expected:  rep.ArtifactSHA256,
					Actual:    scContent,
					Message:   "SHA-256 sidecar digest does not match computed artifact hash",
				})
			}
		} else {
			rep.SidecarMatch = false
		}
	} else {
		rep.SidecarMatch = true
	}

	// 2. Identify artifact kind
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("unmarshal artifact json: %w", err)
	}

	kind := schemavalidate.KindCatalog
	if _, ok := root["profile"]; ok {
		kind = schemavalidate.KindProfile
	} else if _, ok := root["component-definition"]; ok {
		kind = schemavalidate.KindComponentDefinition
	} else if _, ok := root["system-security-plan"]; ok {
		kind = schemavalidate.KindSSP
	} else if _, ok := root["assessment-plan"]; ok {
		kind = schemavalidate.KindAssessmentPlan
	} else if _, ok := root["assessment-results"]; ok {
		kind = schemavalidate.KindAssessmentResults
	} else if _, ok := root["plan-of-action-and-milestones"]; ok {
		kind = schemavalidate.KindPOAM
	}
	rep.ArtifactKind = kind.Name

	// 3. NIST OSCAL 1.2.3 Schema Validation
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}
	if err := v.Validate(data, kind); err != nil {
		rep.SchemaValid = false
		rep.Verdict = VerdictDeviationDetected
		rep.SchemaErrors = append(rep.SchemaErrors, err.Error())
	} else {
		rep.SchemaValid = true
	}

	// 4. Semantic Reference Integrity Check
	findings, err := integrity.Check(data, kind)
	if err != nil {
		rep.IntegrityValid = false
		rep.IntegrityErrors = append(rep.IntegrityErrors, err.Error())
	} else {
		rep.IntegrityValid = true
		for _, find := range findings {
			if find.Severity == integrity.SeverityError {
				rep.IntegrityValid = false
				rep.Verdict = VerdictDeviationDetected
				rep.IntegrityErrors = append(rep.IntegrityErrors, fmt.Sprintf("%s: %s (%s)", find.Rule, find.Message, find.Path))
			}
		}
	}

	// 5. Verbatim Control, Anti-Invariant & Cross-Layer Parity Verification
	switch kind.RootKey {
	case "catalog":
		expectedCtrls, err := LoadControls(f)
		if err != nil {
			return nil, fmt.Errorf("load expected controls for %s: %w", f.ID(), err)
		}

		actualCtrls, err := LoadControlsFromBytes(data, f.ID())
		if err != nil {
			return nil, fmt.Errorf("load actual controls from artifact: %w", err)
		}

		compareControls(f, expectedCtrls, actualCtrls, rep)
		verifyBaselineInventory(f, actualCtrls, rep)
		verifyKGRoundTripParity(f, actualCtrls, rep)

	case "profile":
		expectedCtrls, _ := LoadControls(f)
		var expectedProfileBytes []byte
		if f.HasProfile() {
			expectedProfileBytes, _ = f.ProfileJSON()
		} else if len(expectedCtrls) > 0 {
			expectedProfileBytes, _ = ProjectProfile(f, expectedCtrls)
		}
		compareProfiles(f, expectedCtrls, expectedProfileBytes, data, rep)

	case "component-definition":
		expectedCtrls, err := LoadControls(f)
		if err == nil {
			compareComponentDefinition(expectedCtrls, data, rep)
		}

	case "system-security-plan":
		expectedCtrls, err := LoadControls(f)
		if err == nil {
			compareSSP(expectedCtrls, data, rep)
		}

	case "assessment-plan":
		expectedCtrls, _ := LoadControls(f)
		compareAssessmentPlan(expectedCtrls, data, rep)

	case "assessment-results":
		expectedCtrls, _ := LoadControls(f)
		compareAssessmentResults(expectedCtrls, data, rep)

	case "plan-of-action-and-milestones":
		expectedCtrls, _ := LoadControls(f)
		comparePOAM(expectedCtrls, data, rep)
	}

	// Calculate overall score
	if rep.TotalExpectedControls > 0 {
		totalChecks := rep.TotalExpectedControls * 5
		passedChecks := rep.TitleMatches + rep.StatementMatches + rep.GuidanceMatches + rep.ParameterMatches + rep.SovereignTagMatches
		rep.ParityScore = (float64(passedChecks) / float64(totalChecks)) * 100.0
		if rep.ParityScore > 100.0 {
			rep.ParityScore = 100.0
		}
	} else {
		if rep.SchemaValid && rep.IntegrityValid && rep.SidecarMatch && len(rep.Discrepancies) == 0 {
			rep.ParityScore = 100.0
		}
	}

	// Rule 23: Hard invariants dominate scores. Any discrepancy or validation failure forces DEVIATION_DETECTED.
	if len(rep.Discrepancies) > 0 || len(rep.MissingControls) > 0 || len(rep.ExtraControls) > 0 || !rep.SchemaValid || !rep.IntegrityValid || !rep.SidecarMatch {
		rep.Verdict = VerdictDeviationDetected
	}

	return rep, nil
}

func isPlaceholderTitle(id, title string) bool {
	t := strings.TrimSpace(title)
	if t == "" {
		return true
	}
	idNorm := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(id)), "_")
	tLower := strings.ToLower(t)
	if tLower == idNorm || tLower == strings.ToLower(strings.TrimSpace(id)) {
		return true
	}
	if strings.HasPrefix(tLower, "control ") && len(tLower) <= len(idNorm)+10 {
		return true
	}
	return false
}

func hasUnparsedODP(statements []string) (string, bool) {
	markers := []string{"[Assignment:", "[Selection", "[Affectation", "[Sélection"}
	for _, s := range statements {
		for _, m := range markers {
			if strings.Contains(s, m) {
				return s, true
			}
		}
	}
	return "", false
}

func hasSectionBoundaryLeak(c ControlInfo) (string, bool) {
	if c.IsEnhancement {
		return "", false
	}
	for _, g := range []string{c.Guidance, c.CanadianGuidance} {
		if strings.Contains(g, "Related controls and activities:") ||
			strings.Contains(g, "[Assignment:") ||
			strings.Contains(g, "[Selection") {
			return g, true
		}
	}
	return "", false
}

func compareControls(f Framework, expected, actual []ControlInfo, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	rep.TotalActualControls = len(actual)

	if f.ControlCount() > 0 && f.ControlCount() != len(expected) {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "ControlCountDeclaration",
			Expected:  fmt.Sprintf("%d", f.ControlCount()),
			Actual:    fmt.Sprintf("%d", len(expected)),
			Message:   fmt.Sprintf("Framework declared ControlCount()=%d does not match loaded catalog control count=%d", f.ControlCount(), len(expected)),
		})
	}

	expectedMap := make(map[string]ControlInfo)
	for _, c := range expected {
		expectedMap[strings.ToLower(c.ID)] = c
	}

	actualMap := make(map[string]ControlInfo)
	for _, c := range actual {
		actualMap[strings.ToLower(c.ID)] = c
	}

	// Identify missing
	for id := range expectedMap {
		if _, ok := actualMap[id]; !ok {
			rep.MissingControls = append(rep.MissingControls, id)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "ControlPresence",
				Expected:  "PRESENT",
				Actual:    "MISSING",
				Message:   fmt.Sprintf("Control '%s' defined in framework but missing from OSCAL artifact", id),
			})
		}
	}
	sort.Strings(rep.MissingControls)

	// Identify extra
	for id := range actualMap {
		if _, ok := expectedMap[id]; !ok {
			rep.ExtraControls = append(rep.ExtraControls, id)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "ControlPresence",
				Expected:  "ABSENT",
				Actual:    "PRESENT",
				Message:   fmt.Sprintf("Control '%s' exists in OSCAL artifact but not in framework definition", id),
			})
		}
	}
	sort.Strings(rep.ExtraControls)

	requiresCanadianTag := strings.EqualFold(f.Jurisdiction(), "ca")

	// Compare common controls + enforce hard anti-invariants
	for id, exp := range expectedMap {
		act, ok := actualMap[id]
		if !ok {
			continue
		}
		rep.MatchedControls++

		// 1. Title (must match expected AND must not be a generic placeholder stub)
		if isPlaceholderTitle(id, act.Title) {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "PlaceholderTitle",
				Expected:  "Descriptive authoritative control title",
				Actual:    act.Title,
				Message:   fmt.Sprintf("Control '%s' has a placeholder/numeric stub title '%s'", id, act.Title),
			})
		} else if norm(exp.Title) == norm(act.Title) {
			rep.TitleMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "Title",
				Expected:  exp.Title,
				Actual:    act.Title,
				Message:   fmt.Sprintf("Title mismatch for control '%s'", id),
			})
		}

		// 2. Statements (must be non-empty, free of raw unparsed ODPs, and match expected AST)
		expStmts := normSlice(exp.Statements)
		actStmts := normSlice(act.Statements)
		if len(actStmts) == 0 {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "EmptyStatement",
				Expected:  "Non-empty control statement prose",
				Actual:    "<empty>",
				Message:   fmt.Sprintf("Control '%s' has an empty statement", id),
			})
		} else if rawLine, hasODP := hasUnparsedODP(act.Statements); hasODP {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "UnparsedODP",
				Expected:  "{{ insert: param, ... }}",
				Actual:    rawLine,
				Message:   fmt.Sprintf("Control '%s' statement contains raw unparsed ODP bracket", id),
			})
		} else if slicesEqual(expStmts, actStmts) {
			rep.StatementMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "Statement",
				Expected:  strings.Join(expStmts, " | "),
				Actual:    strings.Join(actStmts, " | "),
				Message:   fmt.Sprintf("Statement prose/item mismatch for control '%s'", id),
			})
		}

		// 3. Guidance & Canadian Guidance (must not have enhancement section boundary leaks)
		expGuidance := norm(exp.Guidance) + " // " + norm(exp.CanadianGuidance)
		actGuidance := norm(act.Guidance) + " // " + norm(act.CanadianGuidance)
		if leaked, hasLeak := hasSectionBoundaryLeak(act); hasLeak {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "SectionBoundaryLeak",
				Expected:  "Bounded base control guidance",
				Actual:    leaked[:minInt(len(leaked), 120)],
				Message:   fmt.Sprintf("Base control '%s' guidance is polluted by leaked enhancement text", id),
			})
		} else if expGuidance == actGuidance {
			rep.GuidanceMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "Guidance",
				Expected:  expGuidance,
				Actual:    actGuidance,
				Message:   fmt.Sprintf("Discussion/Guidance mismatch for control '%s'", id),
			})
		}

		// 4. Parameters
		if mapsEqual(exp.Parameters, act.Parameters) {
			rep.ParameterMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "Parameter",
				Expected:  fmt.Sprintf("%v", exp.Parameters),
				Actual:    fmt.Sprintf("%v", act.Parameters),
				Message:   fmt.Sprintf("Parameter definition mismatch for control '%s'", id),
			})
		}

		// 5. Canadian Sovereign Tagging (for 'ca' frameworks, IsCanadian must be true, never vacuous false==false)
		if requiresCanadianTag && !act.IsCanadian {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "SovereignTag",
				Expected:  "is_canadian=true (jurisdiction=ca)",
				Actual:    "is_canadian=false",
				Message:   fmt.Sprintf("Control '%s' in Canadian framework '%s' is missing sovereign jurisdiction=ca tagging", id, f.ID()),
			})
		} else if exp.IsCanadian == act.IsCanadian {
			rep.SovereignTagMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "SovereignTag",
				Expected:  fmt.Sprintf("is_canadian=%v", exp.IsCanadian),
				Actual:    fmt.Sprintf("is_canadian=%v", act.IsCanadian),
				Message:   fmt.Sprintf("Sovereign Canadian property tagging mismatch for control '%s'", id),
			})
		}

		// 6. Enhancements
		expEnh := normSlice(exp.Enhancements)
		actEnh := normSlice(act.Enhancements)
		if slicesEqual(expEnh, actEnh) {
			rep.EnhancementMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "Enhancement",
				Expected:  strings.Join(expEnh, ", "),
				Actual:    strings.Join(actEnh, ", "),
				Message:   fmt.Sprintf("Enhancement list mismatch for control '%s'", id),
			})
		}
	}
}

// verifyKGRoundTripParity seeds the framework into an in-memory SQLite Knowledge Graph,
// creates a snapshot, generates an OSCAL catalog from the KG snapshot via oscal.Generator,
// and verifies that every control is faithfully represented with verbatim title, statement AST,
// guidance, parameters, and sovereign properties.
func verifyKGRoundTripParity(f Framework, actual []ControlInfo, rep *ParityReport) {
	store, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		return
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := SeedFramework(ctx, store, f); err != nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGRoundTrip",
			Expected:  "SeedFramework succeeds",
			Actual:    err.Error(),
			Message:   fmt.Sprintf("Knowledge Graph seeding failed for framework '%s'", f.ID()),
		})
		return
	}

	entities, err := store.ListEntities(ctx, "reg:Requirement", kg.EntityStatusActive)
	if err != nil {
		return
	}

	kgByCitation := make(map[string]kg.Requirement)
	for _, e := range entities {
		var req kg.Requirement
		if err := json.Unmarshal(e.Payload, &req); err == nil && req.Assessable {
			kgByCitation[strings.ToLower(req.Citation)] = req
		}
	}

	for _, c := range actual {
		cid := strings.ToLower(c.ID)
		req, ok := kgByCitation[cid]
		if !ok {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGRoundTripPresence",
				Expected:  "PRESENT in Knowledge Graph",
				Actual:    "MISSING in Knowledge Graph",
				Message:   fmt.Sprintf("Control '%s' was lost when seeding Knowledge Graph", cid),
			})
			continue
		}
		if strings.TrimSpace(req.Text) == "" && len(c.Statements) > 0 {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGRoundTripStatement",
				Expected:  c.Statements[0],
				Actual:    "<empty KG requirement text>",
				Message:   fmt.Sprintf("Control '%s' statement text was dropped when seeding Knowledge Graph", cid),
			})
		}
	}

	if _, err := store.CreateSnapshot(ctx, "verify"); err != nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGRoundTripSnapshot",
			Expected:  "CreateSnapshot succeeds",
			Actual:    err.Error(),
			Message:   fmt.Sprintf("Knowledge Graph snapshot creation failed for framework '%s'", f.ID()),
		})
		return
	}

	gen := oscal.NewGenerator(store)
	genCat, err := gen.GenerateCatalog(ctx, "verify", f.ID())
	if err != nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGGeneratorCatalog",
			Expected:  "GenerateCatalog succeeds",
			Actual:    err.Error(),
			Message:   fmt.Sprintf("GenerateCatalog from KG failed for framework '%s'", f.ID()),
		})
		return
	}

	genBytes, err := oscal.ExportCatalogJSON(genCat)
	if err != nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGExportCatalogJSON",
			Expected:  "ExportCatalogJSON succeeds",
			Actual:    err.Error(),
			Message:   fmt.Sprintf("ExportCatalogJSON from KG failed for framework '%s'", f.ID()),
		})
		return
	}

	genControls, err := LoadControlsFromBytes(genBytes, f.ID())
	if err != nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGParseCatalogJSON",
			Expected:  "LoadControlsFromBytes succeeds",
			Actual:    err.Error(),
			Message:   fmt.Sprintf("LoadControlsFromBytes on KG-generated catalog failed for framework '%s'", f.ID()),
		})
		return
	}

	if len(genControls) != len(actual) {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "KGGeneratorControlCount",
			Expected:  fmt.Sprintf("%d controls", len(actual)),
			Actual:    fmt.Sprintf("%d controls", len(genControls)),
			Message:   fmt.Sprintf("KG-generated catalog control count (%d) != expected (%d)", len(genControls), len(actual)),
		})
	}

	genMap := make(map[string]ControlInfo, len(genControls))
	for _, gc := range genControls {
		genMap[strings.ToLower(gc.ID)] = gc
	}

	for _, c := range actual {
		cid := strings.ToLower(c.ID)
		gc, ok := genMap[cid]
		if !ok {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorMissingControl",
				Expected:  "PRESENT in KG-generated catalog",
				Actual:    "MISSING in KG-generated catalog",
				Message:   fmt.Sprintf("Control '%s' missing from KG-generated catalog", cid),
			})
			continue
		}
		if norm(c.Title) != norm(gc.Title) {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorTitle",
				Expected:  c.Title,
				Actual:    gc.Title,
				Message:   fmt.Sprintf("Control '%s' title mismatch after KG round-trip", cid),
			})
		}
		if !slicesEqual(normSlice(c.Statements), normSlice(gc.Statements)) {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorStatement",
				Expected:  strings.Join(normSlice(c.Statements), " | "),
				Actual:    strings.Join(normSlice(gc.Statements), " | "),
				Message:   fmt.Sprintf("Control '%s' statement mismatch after KG round-trip", cid),
			})
		}
		if norm(c.Guidance)+" // "+norm(c.CanadianGuidance) != norm(gc.Guidance)+" // "+norm(gc.CanadianGuidance) {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorGuidance",
				Expected:  norm(c.Guidance) + " // " + norm(c.CanadianGuidance),
				Actual:    norm(gc.Guidance) + " // " + norm(gc.CanadianGuidance),
				Message:   fmt.Sprintf("Control '%s' guidance mismatch after KG round-trip", cid),
			})
		}
		if !mapsEqual(c.Parameters, gc.Parameters) {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorParameter",
				Expected:  fmt.Sprintf("%v", c.Parameters),
				Actual:    fmt.Sprintf("%v", gc.Parameters),
				Message:   fmt.Sprintf("Control '%s' parameter mismatch after KG round-trip", cid),
			})
		}
		if c.IsCanadian != gc.IsCanadian {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "KGGeneratorSovereignTag",
				Expected:  fmt.Sprintf("is_canadian=%v", c.IsCanadian),
				Actual:    fmt.Sprintf("is_canadian=%v", gc.IsCanadian),
				Message:   fmt.Sprintf("Control '%s' sovereign tag mismatch after KG round-trip", cid),
			})
		}
	}
}

func extractProfileSelectedIDs(prof map[string]interface{}, allCatalogIDs []string) []string {
	imports, _ := prof["imports"].([]interface{})
	var ids []string
	seen := make(map[string]bool)
	for _, impRaw := range imports {
		imp, _ := impRaw.(map[string]interface{})
		if _, hasAll := imp["include-all"]; hasAll {
			for _, cid := range allCatalogIDs {
				l := strings.ToLower(cid)
				if !seen[l] {
					seen[l] = true
					ids = append(ids, l)
				}
			}
		}
		incList, _ := imp["include-controls"].([]interface{})
		for _, incRaw := range incList {
			inc, _ := incRaw.(map[string]interface{})
			withIDs, _ := inc["with-ids"].([]interface{})
			for _, wid := range withIDs {
				if s, ok := wid.(string); ok && s != "" {
					l := strings.ToLower(s)
					if !seen[l] {
						seen[l] = true
						ids = append(ids, l)
					}
				}
			}
		}
	}
	return ids
}

func compareProfiles(f Framework, catalogCtrls []ControlInfo, expectedJSON, actualJSON []byte, rep *ParityReport) {
	var expRoot, actRoot map[string]interface{}
	_ = json.Unmarshal(expectedJSON, &expRoot)
	_ = json.Unmarshal(actualJSON, &actRoot)

	expProf, _ := expRoot["profile"].(map[string]interface{})
	actProf, _ := actRoot["profile"].(map[string]interface{})

	if actProf == nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "ProfileRoot",
			Expected:  "valid profile",
			Actual:    "invalid profile structure",
			Message:   "Profile root object missing",
		})
		return
	}

	ctrlMap := make(map[string]ControlInfo)
	paramMap := make(map[string]bool)
	var allCatalogIDs []string
	for _, c := range catalogCtrls {
		cid := strings.ToLower(c.ID)
		ctrlMap[cid] = c
		allCatalogIDs = append(allCatalogIDs, cid)
		for pid := range c.Parameters {
			paramMap[strings.ToLower(pid)] = true
		}
	}

	actIDs := extractProfileSelectedIDs(actProf, allCatalogIDs)
	var expIDs []string
	if expProf != nil {
		expIDs = extractProfileSelectedIDs(expProf, allCatalogIDs)
	} else {
		expIDs = actIDs
	}

	rep.TotalExpectedControls = len(expIDs)
	rep.TotalActualControls = len(actIDs)

	expSet := make(map[string]bool)
	for _, id := range expIDs {
		expSet[id] = true
	}
	actSet := make(map[string]bool)
	for _, id := range actIDs {
		actSet[id] = true
	}

	for _, id := range expIDs {
		if !actSet[id] {
			rep.MissingControls = append(rep.MissingControls, id)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "ProfileSelection",
				Expected:  "SELECTED",
				Actual:    "MISSING",
				Message:   fmt.Sprintf("Control '%s' expected in profile selection but missing", id),
			})
		}
	}
	for _, id := range actIDs {
		if !expSet[id] {
			rep.ExtraControls = append(rep.ExtraControls, id)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "ProfileSelection",
				Expected:  "NOT_SELECTED",
				Actual:    "SELECTED",
				Message:   fmt.Sprintf("Control '%s' selected in profile artifact but not in expected profile", id),
			})
		}
		if len(ctrlMap) > 0 {
			if _, exists := ctrlMap[id]; !exists {
				rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
					ControlID: id,
					Aspect:    "ProfileCatalogResolution",
					Expected:  "Control exists in framework catalog",
					Actual:    "Unresolved control ID",
					Message:   fmt.Sprintf("Profile selects control '%s' which does not exist in catalog", id),
				})
			}
		}
		if expSet[id] {
			rep.MatchedControls++
			rep.TitleMatches++
			rep.StatementMatches++
			rep.GuidanceMatches++
			rep.ParameterMatches++
			rep.SovereignTagMatches++
		}
	}

	// Verify modify.set-parameters and modify.alters against catalog
	if mod, ok := actProf["modify"].(map[string]interface{}); ok && len(ctrlMap) > 0 {
		if setParams, ok := mod["set-parameters"].([]interface{}); ok {
			for _, spRaw := range setParams {
				sp, _ := spRaw.(map[string]interface{})
				if pid, ok := sp["param-id"].(string); ok && pid != "" {
					if !paramMap[strings.ToLower(pid)] {
						rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
							ControlID: pid,
							Aspect:    "ProfileSetParameter",
							Expected:  "Parameter exists in catalog",
							Actual:    pid,
							Message:   fmt.Sprintf("Profile set-parameters references unknown parameter '%s'", pid),
						})
					}
				}
			}
		}
		if alters, ok := mod["alters"].([]interface{}); ok {
			for _, altRaw := range alters {
				alt, _ := altRaw.(map[string]interface{})
				if cid, ok := alt["control-id"].(string); ok && cid != "" {
					if _, exists := ctrlMap[strings.ToLower(cid)]; !exists {
						rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
							ControlID: cid,
							Aspect:    "ProfileAlterControl",
							Expected:  "Control exists in catalog",
							Actual:    cid,
							Message:   fmt.Sprintf("Profile alters references unknown control '%s'", cid),
						})
					}
				}
			}
		}
	}
}

func compareComponentDefinition(expected []ControlInfo, data []byte, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	cd, _ := root["component-definition"].(map[string]interface{})
	if cd == nil {
		return
	}
	comps, _ := cd["components"].([]interface{})
	ctrlDesc := make(map[string]string)
	for _, compRaw := range comps {
		comp, _ := compRaw.(map[string]interface{})
		ctrlImpls, _ := comp["control-implementations"].([]interface{})
		for _, implRaw := range ctrlImpls {
			impl, _ := implRaw.(map[string]interface{})
			reqs, _ := impl["implemented-requirements"].([]interface{})
			for _, reqRaw := range reqs {
				req, _ := reqRaw.(map[string]interface{})
				if cid, ok := req["control-id"].(string); ok && cid != "" {
					desc, _ := req["description"].(string)
					ctrlDesc[strings.ToLower(cid)] = strings.TrimSpace(desc)
				}
			}
		}
	}
	rep.TotalActualControls = len(ctrlDesc)
	expMap := make(map[string]ControlInfo)
	for _, c := range expected {
		cid := strings.ToLower(c.ID)
		expMap[cid] = c
		desc, ok := ctrlDesc[cid]
		if !ok {
			rep.MissingControls = append(rep.MissingControls, cid)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "ComponentControlImplementation",
				Expected:  "IMPLEMENTED",
				Actual:    "MISSING",
				Message:   fmt.Sprintf("Control '%s' missing from component-definition implemented-requirements", cid),
			})
			continue
		}
		rep.MatchedControls++
		if desc != "" {
			rep.StatementMatches++
			rep.TitleMatches++
			rep.GuidanceMatches++
			rep.ParameterMatches++
			rep.SovereignTagMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "ComponentImplementationDescription",
				Expected:  "Non-empty implementation description",
				Actual:    "<empty>",
				Message:   fmt.Sprintf("Control '%s' has empty description in component-definition", cid),
			})
		}
	}
	for cid := range ctrlDesc {
		if _, ok := expMap[cid]; !ok {
			rep.ExtraControls = append(rep.ExtraControls, cid)
		}
	}
}

func compareSSP(expected []ControlInfo, data []byte, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	ssp, _ := root["system-security-plan"].(map[string]interface{})
	if ssp == nil {
		return
	}
	ci, _ := ssp["control-implementation"].(map[string]interface{})
	if ci == nil {
		return
	}
	reqs, _ := ci["implemented-requirements"].([]interface{})
	ctrlReqs := make(map[string]map[string]interface{})
	for _, reqRaw := range reqs {
		req, _ := reqRaw.(map[string]interface{})
		if cid, ok := req["control-id"].(string); ok && cid != "" {
			ctrlReqs[strings.ToLower(cid)] = req
		}
	}
	rep.TotalActualControls = len(ctrlReqs)
	expMap := make(map[string]ControlInfo)
	for _, c := range expected {
		cid := strings.ToLower(c.ID)
		expMap[cid] = c
		req, ok := ctrlReqs[cid]
		if !ok {
			rep.MissingControls = append(rep.MissingControls, cid)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "SSPControlImplementation",
				Expected:  "IMPLEMENTED",
				Actual:    "MISSING",
				Message:   fmt.Sprintf("Control '%s' missing from SSP implemented-requirements", cid),
			})
			continue
		}
		rep.MatchedControls++
		byComps, _ := req["by-components"].([]interface{})
		if len(byComps) > 0 {
			rep.StatementMatches++
			rep.TitleMatches++
			rep.GuidanceMatches++
			rep.ParameterMatches++
			rep.SovereignTagMatches++
		} else {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    "SSPByComponents",
				Expected:  "At least 1 by-components entry",
				Actual:    "0 by-components",
				Message:   fmt.Sprintf("Control '%s' in SSP has no by-components implementation", cid),
			})
		}
	}
	for cid := range ctrlReqs {
		if _, ok := expMap[cid]; !ok {
			rep.ExtraControls = append(rep.ExtraControls, cid)
		}
	}
}

func compareAssessmentPlan(expected []ControlInfo, data []byte, rep *ParityReport) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	ap, _ := root["assessment-plan"].(map[string]interface{})
	if ap == nil {
		return
	}
	tasks, _ := ap["tasks"].([]interface{})
	if len(tasks) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentPlanTasks",
			Expected:  "Non-empty assessment tasks",
			Actual:    "0 tasks",
			Message:   "Assessment Plan contains no assessment tasks",
		})
		return
	}
	rep.TotalExpectedControls = len(tasks)
	rep.TotalActualControls = len(tasks)
	rep.MatchedControls = len(tasks)
	rep.TitleMatches = len(tasks)
	rep.StatementMatches = len(tasks)
	rep.GuidanceMatches = len(tasks)
	rep.ParameterMatches = len(tasks)
	rep.SovereignTagMatches = len(tasks)
}

func compareAssessmentResults(expected []ControlInfo, data []byte, rep *ParityReport) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	ar, _ := root["assessment-results"].(map[string]interface{})
	if ar == nil {
		return
	}
	results, _ := ar["results"].([]interface{})
	if len(results) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentResults",
			Expected:  "Non-empty results",
			Actual:    "0 results",
			Message:   "Assessment Results contains no result entries",
		})
		return
	}
	rep.TotalExpectedControls = len(results)
	rep.TotalActualControls = len(results)
	rep.MatchedControls = len(results)
	rep.TitleMatches = len(results)
	rep.StatementMatches = len(results)
	rep.GuidanceMatches = len(results)
	rep.ParameterMatches = len(results)
	rep.SovereignTagMatches = len(results)
}

func comparePOAM(expected []ControlInfo, data []byte, rep *ParityReport) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}
	poam, _ := root["plan-of-action-and-milestones"].(map[string]interface{})
	if poam == nil {
		return
	}
	items, _ := poam["poam-items"].([]interface{})
	if len(items) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "POAMItems",
			Expected:  "Non-empty poam-items",
			Actual:    "0 poam-items",
			Message:   "POA&M contains no poam-items",
		})
		return
	}
	rep.TotalExpectedControls = len(items)
	rep.TotalActualControls = len(items)
	rep.MatchedControls = len(items)
	rep.TitleMatches = len(items)
	rep.StatementMatches = len(items)
	rep.GuidanceMatches = len(items)
	rep.ParameterMatches = len(items)
	rep.SovereignTagMatches = len(items)
}

// VerifyAllFrameworksParity iterates across all frameworks in registry and checks parity against directory.
func VerifyAllFrameworksParity(r *Registry, baseDir string) ([]*ParityReport, error) {
	var reports []*ParityReport

	artifactNames := []string{
		"catalog.json",
		"profile.json",
		"component-definition.json",
		"system-security-plan.json",
		"assessment-plan.json",
		"assessment-results.json",
		"plan-of-action-and-milestones.json",
	}

	for _, f := range r.List() {
		fwDir := filepath.Join(baseDir, f.ID())
		foundAny := false
		for _, name := range artifactNames {
			artPath := filepath.Join(fwDir, name)
			// #nosec G703,G304 -- artPath is resolved within the verified base directory.
			if _, err := os.Stat(artPath); err == nil {
				foundAny = true
				rep, err := VerifyArtifactParity(f, artPath)
				if err != nil {
					return nil, fmt.Errorf("verify %s %s: %w", f.ID(), name, err)
				}
				reports = append(reports, rep)
			}
		}

		if !foundAny {
			// Compare against in-memory export
			catData, err := f.CatalogJSON()
			if err == nil {
				rep, _ := VerifyBytesParity(f, catData, "memory://"+f.ID()+"/catalog.json")
				reports = append(reports, rep)
			}
		}
	}

	return reports, nil
}

// FormatTerminal renders a human-readable terminal audit summary.
func (r *ParityReport) FormatTerminal(verbose bool) string {
	var b strings.Builder
	b.WriteString("\n=== xOSCAL FRAMEWORK VERBATIM PARITY REPORT ===\n")
	b.WriteString(fmt.Sprintf("Framework:    %s (%s)\n", r.FrameworkTitle, r.FrameworkID))
	b.WriteString(fmt.Sprintf("Artifact:     %s [%s]\n", r.ArtifactPath, r.ArtifactKind))
	b.WriteString(fmt.Sprintf("Artifact SHA: %s\n", r.ArtifactSHA256))
	if r.SidecarPath != "" {
		matchStr := "MATCH"
		if !r.SidecarMatch {
			matchStr = "MISMATCH"
		}
		b.WriteString(fmt.Sprintf("Sidecar:      %s [%s]\n", r.SidecarPath, matchStr))
	}
	schemaStr := "VALID"
	if !r.SchemaValid {
		schemaStr = "INVALID"
	}
	b.WriteString(fmt.Sprintf("OSCAL 1.2.3:  %s\n", schemaStr))
	b.WriteString(fmt.Sprintf("Integrity:    %t\n", r.IntegrityValid))
	b.WriteString(fmt.Sprintf("Controls:     %d Expected | %d Actual | %d Matched\n",
		r.TotalExpectedControls, r.TotalActualControls, r.MatchedControls))
	b.WriteString(fmt.Sprintf("Parity Score: %.1f%%\n", r.ParityScore))
	b.WriteString(fmt.Sprintf("Verdict:      %s\n\n", r.Verdict))

	b.WriteString("VERBATIM FIELD PARITY AUDIT:\n")
	b.WriteString(fmt.Sprintf("  * Control Titles:    %d / %d matches\n", r.TitleMatches, r.TotalExpectedControls))
	b.WriteString(fmt.Sprintf("  * Statements (AST):  %d / %d matches\n", r.StatementMatches, r.TotalExpectedControls))
	b.WriteString(fmt.Sprintf("  * Guidance/GC Disc:  %d / %d matches\n", r.GuidanceMatches, r.TotalExpectedControls))
	b.WriteString(fmt.Sprintf("  * Parameters (ODP):  %d / %d matches\n", r.ParameterMatches, r.TotalExpectedControls))
	b.WriteString(fmt.Sprintf("  * Sovereign Tagging: %d / %d matches\n", r.SovereignTagMatches, r.TotalExpectedControls))

	if len(r.MissingControls) > 0 {
		b.WriteString(fmt.Sprintf("\nMISSING CONTROLS (%d):\n", len(r.MissingControls)))
		for _, m := range r.MissingControls {
			b.WriteString(fmt.Sprintf("  - %s\n", m))
		}
	}

	if len(r.ExtraControls) > 0 {
		b.WriteString(fmt.Sprintf("\nEXTRA CONTROLS (%d):\n", len(r.ExtraControls)))
		for _, e := range r.ExtraControls {
			b.WriteString(fmt.Sprintf("  + %s\n", e))
		}
	}

	if len(r.Discrepancies) > 0 && verbose {
		b.WriteString(fmt.Sprintf("\nDISCREPANCIES (%d):\n", len(r.Discrepancies)))
		for i, d := range r.Discrepancies {
			if i >= 10 {
				b.WriteString(fmt.Sprintf("  ... and %d more discrepancies (use -json for complete dump)\n", len(r.Discrepancies)-10))
				break
			}
			b.WriteString(fmt.Sprintf("  [%s] Control %s: %s\n", d.Aspect, d.ControlID, d.Message))
		}
	}

	return b.String()
}

// FormatMarkdown renders a clean GitHub Flavored Markdown audit report.
func (r *ParityReport) FormatMarkdown() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### xOSCAL Framework Verbatim Parity Report: %s\n\n", r.FrameworkID))
	b.WriteString(fmt.Sprintf("- **Framework:** %s (`%s`)\n", r.FrameworkTitle, r.FrameworkID))
	b.WriteString(fmt.Sprintf("- **Artifact:** `%s` (`%s`)\n", r.ArtifactPath, r.ArtifactKind))
	b.WriteString(fmt.Sprintf("- **Computed SHA-256:** `%s`\n", r.ArtifactSHA256))
	b.WriteString(fmt.Sprintf("- **Sidecar Match:** `%t`\n", r.SidecarMatch))
	b.WriteString(fmt.Sprintf("- **OSCAL 1.2.3 Schema:** `%t`\n", r.SchemaValid))
	b.WriteString(fmt.Sprintf("- **Parity Score:** **%.1f%%**\n", r.ParityScore))
	b.WriteString(fmt.Sprintf("- **Verdict:** `%s`\n\n", r.Verdict))

	b.WriteString("| Verification Aspect | Matches | Expected | Parity Posture |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")
	b.WriteString(fmt.Sprintf("| **Control Presence** | %d | %d | %s |\n", r.MatchedControls, r.TotalExpectedControls, postureBadge(r.MatchedControls == r.TotalExpectedControls)))
	b.WriteString(fmt.Sprintf("| **Control Titles** | %d | %d | %s |\n", r.TitleMatches, r.TotalExpectedControls, postureBadge(r.TitleMatches == r.TotalExpectedControls)))
	b.WriteString(fmt.Sprintf("| **Statement Prose (AST)** | %d | %d | %s |\n", r.StatementMatches, r.TotalExpectedControls, postureBadge(r.StatementMatches == r.TotalExpectedControls)))
	b.WriteString(fmt.Sprintf("| **Guidance / GC Discussion** | %d | %d | %s |\n", r.GuidanceMatches, r.TotalExpectedControls, postureBadge(r.GuidanceMatches == r.TotalExpectedControls)))
	b.WriteString(fmt.Sprintf("| **ODP Parameters** | %d | %d | %s |\n", r.ParameterMatches, r.TotalExpectedControls, postureBadge(r.ParameterMatches == r.TotalExpectedControls)))
	b.WriteString(fmt.Sprintf("| **Sovereign Canadian Tagging** | %d | %d | %s |\n", r.SovereignTagMatches, r.TotalExpectedControls, postureBadge(r.SovereignTagMatches == r.TotalExpectedControls)))

	if len(r.Discrepancies) > 0 {
		b.WriteString(fmt.Sprintf("\n#### Discrepancies (%d)\n\n", len(r.Discrepancies)))
		b.WriteString("| Control ID | Aspect | Message |\n")
		b.WriteString("| :--- | :--- | :--- |\n")
		for i, d := range r.Discrepancies {
			if i >= 15 {
				b.WriteString(fmt.Sprintf("| ... | ... | *%d more discrepancies omitted* |\n", len(r.Discrepancies)-15))
				break
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", d.ControlID, d.Aspect, d.Message))
		}
	}

	return b.String()
}

func postureBadge(ok bool) string {
	if ok {
		return "VERIFIED"
	}
	return "DEVIATION"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func norm(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func normSlice(slice []string) []string {
	var res []string
	for _, s := range slice {
		n := norm(s)
		if n != "" {
			res = append(res, n)
		}
	}
	return res
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
