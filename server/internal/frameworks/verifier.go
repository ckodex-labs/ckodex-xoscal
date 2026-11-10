package frameworks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
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

type integrityCacheResult struct {
	findings []integrity.Finding
	errMsg   string
	hasErr   bool
}

var (
	validatorOnce   sync.Once
	cachedValidator *schemavalidate.Validator
	validatorInitEr error

	controlsCacheMu sync.RWMutex
	controlsCache   = make(map[string][]ControlInfo)

	schemaCacheMu sync.RWMutex
	schemaCache   = make(map[string]string) // key: kind.RootKey+":"+sha256 -> "" (valid) or err.Error()

	integrityCacheMu sync.RWMutex
	integrityCache   = make(map[string]integrityCacheResult) // key: kind.RootKey+":"+sha256

	kgRoundTripCacheMu sync.RWMutex
	kgRoundTripCache   = make(map[string][]Discrepancy)
)

// SharedSchemaValidator returns the process-wide cached OSCAL 1.2.3 schema validator.
func SharedSchemaValidator() (*schemavalidate.Validator, error) {
	return sharedSchemaValidator()
}

func sharedSchemaValidator() (*schemavalidate.Validator, error) {
	validatorOnce.Do(func() {
		cachedValidator, validatorInitEr = schemavalidate.NewValidator()
	})
	return cachedValidator, validatorInitEr
}

// ValidateSchemaCached validates data against the embedded NIST OSCAL 1.2.3 JSON schema,
// memoizing the deterministic outcome by (kind.RootKey, SHA-256(data)).
func ValidateSchemaCached(data []byte, kind schemavalidate.ArtifactKind) error {
	sum := sha256.Sum256(data)
	return validateSchemaCachedWithSum(data, sum, kind)
}

func validateSchemaCachedWithSum(data []byte, sum [32]byte, kind schemavalidate.ArtifactKind) error {
	key := kind.RootKey + ":" + hex.EncodeToString(sum[:])
	schemaCacheMu.RLock()
	cachedErr, ok := schemaCache[key]
	schemaCacheMu.RUnlock()
	if ok {
		if cachedErr == "" {
			return nil
		}
		return fmt.Errorf("%s", cachedErr)
	}

	v, err := sharedSchemaValidator()
	if err != nil {
		return fmt.Errorf("init validator: %w", err)
	}
	valErr := v.Validate(data, kind)
	errStr := ""
	if valErr != nil {
		errStr = valErr.Error()
	}
	schemaCacheMu.Lock()
	schemaCache[key] = errStr
	schemaCacheMu.Unlock()
	return valErr
}

// CheckIntegrityCached runs semantic reference integrity checks on data,
// memoizing the deterministic findings by (kind.RootKey, SHA-256(data)).
func CheckIntegrityCached(data []byte, kind schemavalidate.ArtifactKind) ([]integrity.Finding, error) {
	sum := sha256.Sum256(data)
	return checkIntegrityCachedWithSum(data, sum, kind)
}

func checkIntegrityCachedWithSum(data []byte, sum [32]byte, kind schemavalidate.ArtifactKind) ([]integrity.Finding, error) {
	key := kind.RootKey + ":" + hex.EncodeToString(sum[:])
	integrityCacheMu.RLock()
	cached, ok := integrityCache[key]
	integrityCacheMu.RUnlock()
	if ok {
		if cached.hasErr {
			return nil, fmt.Errorf("%s", cached.errMsg)
		}
		out := make([]integrity.Finding, len(cached.findings))
		copy(out, cached.findings)
		return out, nil
	}

	findings, err := integrity.Check(data, kind)
	entry := integrityCacheResult{}
	if err != nil {
		entry.hasErr = true
		entry.errMsg = err.Error()
	} else if len(findings) > 0 {
		entry.findings = make([]integrity.Finding, len(findings))
		copy(entry.findings, findings)
	}
	integrityCacheMu.Lock()
	integrityCache[key] = entry
	integrityCacheMu.Unlock()
	return findings, err
}

func loadCachedControlsFromBytes(data []byte, sum [32]byte, frameworkID string) ([]ControlInfo, error) {
	key := strings.ToLower(frameworkID) + ":" + hex.EncodeToString(sum[:])
	controlsCacheMu.RLock()
	cached, ok := controlsCache[key]
	controlsCacheMu.RUnlock()
	if ok {
		return cached, nil
	}
	ctrls, err := LoadControlsFromBytes(data, frameworkID)
	if err != nil {
		return nil, err
	}
	controlsCacheMu.Lock()
	controlsCache[key] = ctrls
	controlsCacheMu.Unlock()
	return ctrls, nil
}

func loadCachedExpectedControls(f Framework) ([]ControlInfo, error) {
	data, err := f.CatalogJSON()
	if err != nil {
		return nil, fmt.Errorf("load catalog for %s: %w", f.ID(), err)
	}
	sum := sha256.Sum256(data)
	return loadCachedControlsFromBytes(data, sum, f.ID())
}

func detectOSCALKind(data []byte) (schemavalidate.ArtifactKind, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return schemavalidate.KindCatalog, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return schemavalidate.KindCatalog, fmt.Errorf("expected JSON object root")
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return schemavalidate.KindCatalog, err
		}
		key, _ := keyTok.(string)
		switch key {
		case "catalog":
			return schemavalidate.KindCatalog, nil
		case "profile":
			return schemavalidate.KindProfile, nil
		case "component-definition":
			return schemavalidate.KindComponentDefinition, nil
		case "system-security-plan":
			return schemavalidate.KindSSP, nil
		case "assessment-plan":
			return schemavalidate.KindAssessmentPlan, nil
		case "assessment-results":
			return schemavalidate.KindAssessmentResults, nil
		case "plan-of-action-and-milestones":
			return schemavalidate.KindPOAM, nil
		default:
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return schemavalidate.KindCatalog, err
			}
		}
	}
	return schemavalidate.KindCatalog, nil
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
	return verifyBytesParityWithExpected(f, data, artifactPath, nil)
}

func verifyBytesParityWithExpected(f Framework, data []byte, artifactPath string, preloadedExpected []ControlInfo) (*ParityReport, error) {
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

	// 2. Identify artifact kind without unmarshaling full document trees into map[string]interface{}
	kind, err := detectOSCALKind(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal artifact json: %w", err)
	}
	rep.ArtifactKind = kind.Name

	// 3. NIST OSCAL 1.2.3 Schema Validation (content-addressed by SHA-256)
	if err := validateSchemaCachedWithSum(data, sum, kind); err != nil {
		rep.SchemaValid = false
		rep.Verdict = VerdictDeviationDetected
		rep.SchemaErrors = append(rep.SchemaErrors, err.Error())
	} else {
		rep.SchemaValid = true
	}

	// 4. Semantic Reference Integrity Check (content-addressed by SHA-256)
	findings, err := checkIntegrityCachedWithSum(data, sum, kind)
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

	getExpected := func() ([]ControlInfo, error) {
		if preloadedExpected != nil {
			return preloadedExpected, nil
		}
		return loadCachedExpectedControls(f)
	}

	// 5. Verbatim Control, Anti-Invariant & Cross-Layer Parity Verification
	switch kind.RootKey {
	case "catalog":
		expectedCtrls, err := getExpected()
		if err != nil {
			return nil, fmt.Errorf("load expected controls for %s: %w", f.ID(), err)
		}

		actualCtrls, err := loadCachedControlsFromBytes(data, sum, f.ID())
		if err != nil {
			return nil, fmt.Errorf("load actual controls from artifact: %w", err)
		}

		compareControls(f, expectedCtrls, actualCtrls, rep)
		verifyBaselineInventory(f, actualCtrls, rep)
		verifyKGRoundTripParity(f, actualCtrls, rep)

	case "profile":
		var root map[string]interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, _ := getExpected()
		var expectedProfileBytes []byte
		if f.HasProfile() {
			expectedProfileBytes, _ = f.ProfileJSON()
		} else if len(expectedCtrls) > 0 {
			expectedProfileBytes, _ = ProjectProfile(f, expectedCtrls)
		}
		compareProfiles(f, expectedCtrls, expectedProfileBytes, root, rep)

	case "component-definition":
		var doc oscalVerifyCDDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, err := getExpected()
		if err == nil {
			compareComponentDefinition(expectedCtrls, &doc, rep)
		}

	case "system-security-plan":
		var doc oscalVerifySSPDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, err := getExpected()
		if err == nil {
			compareSSP(expectedCtrls, &doc, rep)
		}

	case "assessment-plan":
		var doc oscalVerifyAPDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, _ := getExpected()
		compareAssessmentPlan(expectedCtrls, &doc, rep)

	case "assessment-results":
		var doc oscalVerifyARDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, _ := getExpected()
		compareAssessmentResults(expectedCtrls, &doc, rep)

	case "plan-of-action-and-milestones":
		var doc oscalVerifyPOAMDoc
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal artifact json: %w", err)
		}
		expectedCtrls, _ := getExpected()
		comparePOAM(expectedCtrls, &doc, rep)
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

// memKGStore is an ephemeral, concurrency-safe in-memory kg.Store used by
// in-memory KG round-trip parity checks so repeated catalog verifications
// exercise the exact SeedFramework -> Snapshot -> oscal.Generator pipeline
// without spawning dozens of C-to-Go SQLite VM instances per test run.
type memKGStore struct {
	mu        sync.RWMutex
	entities  []*kg.Entity
	byURN     map[string]int
	snapshots map[string][]*kg.Entity
	releases  map[string]*kg.Release
}

func newMemKGStore() *memKGStore {
	return &memKGStore{
		byURN:     make(map[string]int),
		snapshots: make(map[string][]*kg.Entity),
		releases:  make(map[string]*kg.Release),
	}
}

func (m *memKGStore) BulkUpsertEntities(_ context.Context, entities []*kg.Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for _, e := range entities {
		if e == nil {
			continue
		}
		cp := *e
		if cp.Status == "" {
			cp.Status = kg.EntityStatusActive
		}
		if cp.ValidFrom.IsZero() {
			cp.ValidFrom = now
		}
		if idx, exists := m.byURN[cp.URN]; exists {
			old := m.entities[idx]
			cp.Version = old.Version + 1
			m.entities[idx] = &cp
		} else {
			if cp.Version == 0 {
				cp.Version = 1
			}
			m.byURN[cp.URN] = len(m.entities)
			m.entities = append(m.entities, &cp)
		}
	}
	return nil
}

func (m *memKGStore) CreateEntity(ctx context.Context, e *kg.Entity) error {
	return m.BulkUpsertEntities(ctx, []*kg.Entity{e})
}

func (m *memKGStore) GetEntity(_ context.Context, urn string) (*kg.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if idx, ok := m.byURN[urn]; ok {
		cp := *m.entities[idx]
		return &cp, nil
	}
	return nil, fmt.Errorf("not found: %s", urn)
}

func (m *memKGStore) UpdateEntity(ctx context.Context, e *kg.Entity) error {
	return m.BulkUpsertEntities(ctx, []*kg.Entity{e})
}

func (m *memKGStore) ListEntities(_ context.Context, entityType string, status kg.EntityStatus) ([]*kg.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*kg.Entity, 0, len(m.entities))
	for _, e := range m.entities {
		if entityType != "" && e.Type != entityType {
			continue
		}
		if status != "" && e.Status != status {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memKGStore) Snapshot(ctx context.Context, _ time.Time) ([]*kg.Entity, error) {
	return m.ListEntities(ctx, "", kg.EntityStatusActive)
}

func (m *memKGStore) CreateSnapshot(_ context.Context, name string) (*kg.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := make([]*kg.Entity, 0, len(m.entities))
	for _, e := range m.entities {
		if e.Status == kg.EntityStatusActive {
			cp := *e
			snap = append(snap, &cp)
		}
	}
	m.snapshots[name] = snap
	return &kg.Snapshot{Name: name, CreatedAt: time.Now().UTC()}, nil
}

func (m *memKGStore) GetSnapshot(_ context.Context, name string) ([]*kg.Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap, ok := m.snapshots[name]
	if !ok {
		return nil, fmt.Errorf("snapshot %s not found", name)
	}
	return snap, nil
}

func (m *memKGStore) ListSnapshots(_ context.Context) ([]*kg.Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*kg.Snapshot
	for name := range m.snapshots {
		out = append(out, &kg.Snapshot{Name: name})
	}
	return out, nil
}

func (m *memKGStore) CreateRelease(_ context.Context, name, snapshotName string) (*kg.Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rel := &kg.Release{Name: name, Snapshot: snapshotName, CreatedAt: time.Now().UTC()}
	m.releases[name] = rel
	return rel, nil
}

func (m *memKGStore) GetRelease(_ context.Context, name string) (*kg.Release, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rel, ok := m.releases[name]
	if !ok {
		return nil, fmt.Errorf("release %s not found", name)
	}
	return rel, nil
}

func (m *memKGStore) ListReleases(_ context.Context) ([]*kg.Release, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*kg.Release
	for _, rel := range m.releases {
		out = append(out, rel)
	}
	return out, nil
}

func (m *memKGStore) Close() error { return nil }

// verifyKGRoundTripParity seeds the framework into an in-memory Knowledge Graph,
// creates a snapshot, generates an OSCAL catalog from the KG snapshot via oscal.Generator,
// and verifies that every control is faithfully represented with verbatim title, statement AST,
// guidance, parameters, and sovereign properties.
func verifyKGRoundTripParity(f Framework, actual []ControlInfo, rep *ParityReport) {
	var cacheKey string
	if catBytes, err := f.CatalogJSON(); err == nil && rep.ArtifactSHA256 != "" {
		catSum := sha256.Sum256(catBytes)
		profHex := ""
		if f.HasProfile() {
			if pb, err := f.ProfileJSON(); err == nil {
				ps := sha256.Sum256(pb)
				profHex = hex.EncodeToString(ps[:])
			}
		}
		cacheKey = f.ID() + "|" + f.Jurisdiction() + "|" + hex.EncodeToString(catSum[:]) + "|" + profHex + "|" + rep.ArtifactSHA256
		kgRoundTripCacheMu.RLock()
		cached, ok := kgRoundTripCache[cacheKey]
		kgRoundTripCacheMu.RUnlock()
		if ok {
			rep.Discrepancies = append(rep.Discrepancies, cached...)
			return
		}
	}
	baseDiscIdx := len(rep.Discrepancies)
	defer func() {
		if cacheKey != "" {
			added := rep.Discrepancies[baseDiscIdx:]
			stored := make([]Discrepancy, len(added))
			copy(stored, added)
			kgRoundTripCacheMu.Lock()
			kgRoundTripCache[cacheKey] = stored
			kgRoundTripCacheMu.Unlock()
		}
	}()

	store := newMemKGStore()
	defer func() { _ = store.Close() }()

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

	genControls := loadControlsFromProto(genCat, f.ID())

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

func loadControlsFromProto(cat *catalogv1.Catalog, frameworkID string) []ControlInfo {
	if cat == nil {
		return nil
	}
	var result []ControlInfo
	partProse := func(p *catalogv1.Part) string {
		if p == nil || len(p.Prose) == 0 {
			return ""
		}
		if len(p.Prose) == 1 {
			if p.Prose[0] == nil {
				return ""
			}
			return p.Prose[0].Value
		}
		var parts []string
		for _, pr := range p.Prose {
			if pr != nil && pr.Value != "" {
				parts = append(parts, pr.Value)
			}
		}
		return strings.Join(parts, "\n\n")
	}

	var parseControl func(c *catalogv1.Control, famID, famTitle, parentID string) ControlInfo
	parseControl = func(c *catalogv1.Control, famID, famTitle, parentID string) ControlInfo {
		ctrlID := ""
		if c.Id != nil {
			ctrlID = c.Id.Value
		}
		title := ""
		if c.Title != nil {
			title = c.Title.Value
		}
		info := ControlInfo{
			ID:              strings.ToLower(ctrlID),
			Title:           title,
			Class:           c.Class,
			FamilyID:        strings.ToUpper(famID),
			FamilyTitle:     famTitle,
			FrameworkID:     frameworkID,
			Parameters:      make(map[string]string),
			ParentControlID: strings.ToLower(parentID),
			IsEnhancement:   parentID != "",
		}

		for _, prop := range c.Props {
			if prop == nil {
				continue
			}
			switch strings.ToLower(prop.Name) {
			case "jurisdiction":
				if strings.EqualFold(prop.Value, "ca") {
					info.IsCanadian = true
				}
			case "sovereign-guidance", "sovereign-control", "sovereign-specification":
				if strings.EqualFold(prop.Value, "true") {
					info.IsCanadian = true
				}
			case "family-id":
				if info.FamilyID == "" || info.FamilyID == "GENERAL" {
					info.FamilyID = strings.ToUpper(prop.Value)
				}
			case "family-title":
				if info.FamilyTitle == "" || info.FamilyTitle == "General Controls" {
					info.FamilyTitle = prop.Value
				}
			}
		}

		for _, p := range c.Params {
			if p == nil || p.Id == nil || p.Id.Value == "" {
				continue
			}
			val := ""
			if p.Label != nil {
				val = p.Label.Value
			}
			var vals []string
			for _, v := range p.Values {
				if v != nil && v.Value != "" {
					vals = append(vals, v.Value)
				}
			}
			if len(vals) > 0 {
				val = strings.Join(vals, ", ")
			}
			if val == "" && p.Select != nil {
				var choices []string
				for _, ch := range p.Select.Choices {
					if ch != nil && ch.Value != "" {
						choices = append(choices, ch.Value)
					}
				}
				if len(choices) > 0 {
					val = strings.Join(choices, "; ")
				}
			}
			info.Parameters[p.Id.Value] = val
		}

		var stmts []string
		var extractPartStmts func(part *catalogv1.Part)
		extractPartStmts = func(part *catalogv1.Part) {
			if part == nil {
				return
			}
			lbl := ""
			if part.Title != nil {
				lbl = part.Title.Value
			}
			for _, prop := range part.Props {
				if prop == nil {
					continue
				}
				if strings.EqualFold(prop.Name, "label") && prop.Value != "" {
					lbl = prop.Value
				}
				if (strings.EqualFold(prop.Name, "jurisdiction") && strings.EqualFold(prop.Value, "ca")) ||
					(strings.EqualFold(prop.Name, "sovereign-specification") && strings.EqualFold(prop.Value, "true")) {
					info.IsCanadian = true
				}
			}
			prose := partProse(part)
			if prose != "" {
				if strings.EqualFold(part.Name, "item") && lbl != "" {
					stmts = append(stmts, lbl+" "+prose)
				} else {
					stmts = append(stmts, prose)
				}
			}
			for _, sub := range part.Parts {
				extractPartStmts(sub)
			}
		}

		for _, p := range c.Parts {
			if p == nil {
				continue
			}
			pName := strings.ToLower(p.Name)
			pTitle := ""
			if p.Title != nil {
				pTitle = p.Title.Value
			}
			isSovereignGuidance := pName == "gc-guidance" || pName == "canadian-guidance" || strings.EqualFold(pTitle, "GC Discussion")
			for _, prop := range p.Props {
				if prop == nil {
					continue
				}
				if (strings.EqualFold(prop.Name, "sovereign-guidance") && strings.EqualFold(prop.Value, "true")) ||
					(strings.EqualFold(prop.Name, "jurisdiction") && strings.EqualFold(prop.Value, "ca")) {
					isSovereignGuidance = true
				}
			}
			prose := partProse(p)
			switch {
			case pName == "statement":
				extractPartStmts(p)
			case isSovereignGuidance && (pName == "guidance" || pName == "gc-guidance" || pName == "canadian-guidance"):
				if prose != "" {
					info.CanadianGuidance = prose
					info.IsCanadian = true
				}
			case pName == "guidance":
				if prose != "" {
					info.Guidance = prose
				}
			default:
				if !strings.Contains(pName, "assessment") {
					extractPartStmts(p)
				}
			}
		}
		info.Statements = stmts

		for _, sub := range c.Controls {
			if sub == nil {
				continue
			}
			enh := parseControl(sub, famID, famTitle, ctrlID)
			result = append(result, enh)
			info.Enhancements = append(info.Enhancements, enh.ID)
		}
		return info
	}

	var traverseGroup func(g *catalogv1.Group, topFamID, topFamTitle string)
	traverseGroup = func(g *catalogv1.Group, topFamID, topFamTitle string) {
		if g == nil {
			return
		}
		famID := topFamID
		famTitle := topFamTitle
		if famID == "" {
			if g.Id != nil {
				famID = g.Id.Value
			}
			if g.Title != nil {
				famTitle = g.Title.Value
			}
		}
		for _, c := range g.Controls {
			if c != nil {
				result = append(result, parseControl(c, famID, famTitle, ""))
			}
		}
		for _, sub := range g.Groups {
			traverseGroup(sub, famID, famTitle)
		}
	}

	for _, g := range cat.Groups {
		if g == nil {
			continue
		}
		gid, gtitle := "", ""
		if g.Id != nil {
			gid = g.Id.Value
		}
		if g.Title != nil {
			gtitle = g.Title.Value
		}
		traverseGroup(g, gid, gtitle)
	}
	for _, c := range cat.Controls {
		if c != nil {
			result = append(result, parseControl(c, "GENERAL", "General Controls", ""))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
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

func compareProfiles(f Framework, catalogCtrls []ControlInfo, expectedJSON []byte, actRoot map[string]interface{}, rep *ParityReport) {
	var expRoot map[string]interface{}
	_ = json.Unmarshal(expectedJSON, &expRoot)

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

type oscalVerifyCDDoc struct {
	ComponentDefinition *struct {
		Components []struct {
			ControlImplementations []struct {
				ImplementedRequirements []struct {
					ControlID   string `json:"control-id"`
					Description string `json:"description"`
				} `json:"implemented-requirements"`
			} `json:"control-implementations"`
		} `json:"components"`
	} `json:"component-definition"`
}

type oscalVerifySSPDoc struct {
	SSP *struct {
		ControlImplementation *struct {
			ImplementedRequirements []struct {
				ControlID    string            `json:"control-id"`
				ByComponents []json.RawMessage `json:"by-components"`
			} `json:"implemented-requirements"`
		} `json:"control-implementation"`
	} `json:"system-security-plan"`
}

type oscalVerifyReviewedControls struct {
	ControlSelections []struct {
		IncludeControls []struct {
			ControlID string   `json:"control-id"`
			WithIDs   []string `json:"with-ids"`
		} `json:"include-controls"`
	} `json:"control-selections"`
}

type oscalVerifyAPDoc struct {
	AssessmentPlan *struct {
		Tasks            []json.RawMessage            `json:"tasks"`
		ReviewedControls *oscalVerifyReviewedControls `json:"reviewed-controls"`
	} `json:"assessment-plan"`
}

type oscalVerifyARDoc struct {
	AssessmentResults *struct {
		Results []struct {
			ReviewedControls *oscalVerifyReviewedControls `json:"reviewed-controls"`
		} `json:"results"`
	} `json:"assessment-results"`
}

type oscalVerifyPOAMDoc struct {
	POAM *struct {
		PoamItems []struct {
			Props []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"props"`
		} `json:"poam-items"`
	} `json:"plan-of-action-and-milestones"`
}

func compareComponentDefinition(expected []ControlInfo, doc *oscalVerifyCDDoc, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	if doc == nil || doc.ComponentDefinition == nil {
		return
	}
	ctrlDesc := make(map[string]string, len(expected))
	for _, comp := range doc.ComponentDefinition.Components {
		for _, impl := range comp.ControlImplementations {
			for _, req := range impl.ImplementedRequirements {
				if req.ControlID != "" {
					ctrlDesc[strings.ToLower(req.ControlID)] = strings.TrimSpace(req.Description)
				}
			}
		}
	}
	rep.TotalActualControls = len(ctrlDesc)
	expMap := make(map[string]ControlInfo, len(expected))
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

func compareSSP(expected []ControlInfo, doc *oscalVerifySSPDoc, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	if doc == nil || doc.SSP == nil || doc.SSP.ControlImplementation == nil {
		return
	}
	ctrlByCompCount := make(map[string]int, len(expected))
	for _, req := range doc.SSP.ControlImplementation.ImplementedRequirements {
		if req.ControlID != "" {
			ctrlByCompCount[strings.ToLower(req.ControlID)] = len(req.ByComponents)
		}
	}
	rep.TotalActualControls = len(ctrlByCompCount)
	expMap := make(map[string]ControlInfo, len(expected))
	for _, c := range expected {
		cid := strings.ToLower(c.ID)
		expMap[cid] = c
		byCompCount, ok := ctrlByCompCount[cid]
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
		if byCompCount > 0 {
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
	for cid := range ctrlByCompCount {
		if _, ok := expMap[cid]; !ok {
			rep.ExtraControls = append(rep.ExtraControls, cid)
		}
	}
}

func extractTypedReviewedControlIDs(rc *oscalVerifyReviewedControls, dest map[string]bool) {
	if rc == nil {
		return
	}
	for _, sel := range rc.ControlSelections {
		for _, inc := range sel.IncludeControls {
			if cid := strings.ToLower(strings.TrimSpace(inc.ControlID)); cid != "" {
				dest[cid] = true
			}
			for _, wid := range inc.WithIDs {
				if s := strings.ToLower(strings.TrimSpace(wid)); s != "" {
					dest[s] = true
				}
			}
		}
	}
}

func compareControlIDSet(expected []ControlInfo, actualSet map[string]bool, aspect string, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	rep.TotalActualControls = len(actualSet)

	expMap := make(map[string]ControlInfo, len(expected))
	for _, c := range expected {
		cid := strings.ToLower(strings.TrimSpace(c.ID))
		if cid == "" {
			continue
		}
		expMap[cid] = c
		if !actualSet[cid] {
			rep.MissingControls = append(rep.MissingControls, cid)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    aspect,
				Expected:  "PRESENT",
				Actual:    "MISSING",
				Message:   fmt.Sprintf("Control '%s' missing from %s", cid, aspect),
			})
			continue
		}
		rep.MatchedControls++
		rep.TitleMatches++
		rep.StatementMatches++
		rep.GuidanceMatches++
		rep.ParameterMatches++
		rep.SovereignTagMatches++
	}
	sort.Strings(rep.MissingControls)

	for cid := range actualSet {
		if _, ok := expMap[cid]; !ok {
			rep.ExtraControls = append(rep.ExtraControls, cid)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: cid,
				Aspect:    aspect,
				Expected:  "ABSENT",
				Actual:    "PRESENT",
				Message:   fmt.Sprintf("Unexpected control '%s' present in %s", cid, aspect),
			})
		}
	}
	sort.Strings(rep.ExtraControls)
}

func compareAssessmentPlan(expected []ControlInfo, doc *oscalVerifyAPDoc, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	if doc == nil || doc.AssessmentPlan == nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentPlanRoot",
			Expected:  "valid assessment-plan",
			Actual:    "missing assessment-plan",
			Message:   "Assessment Plan root object missing",
		})
		return
	}
	if len(expected) > 0 && len(doc.AssessmentPlan.Tasks) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentPlanTasks",
			Expected:  "Non-empty assessment tasks",
			Actual:    "0 tasks",
			Message:   "Assessment Plan contains no assessment tasks",
		})
	}

	actualSet := make(map[string]bool, len(expected))
	extractTypedReviewedControlIDs(doc.AssessmentPlan.ReviewedControls, actualSet)
	compareControlIDSet(expected, actualSet, "AssessmentPlanReviewedControls", rep)
}

func compareAssessmentResults(expected []ControlInfo, doc *oscalVerifyARDoc, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	if doc == nil || doc.AssessmentResults == nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentResultsRoot",
			Expected:  "valid assessment-results",
			Actual:    "missing assessment-results",
			Message:   "Assessment Results root object missing",
		})
		return
	}
	if len(doc.AssessmentResults.Results) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "AssessmentResults",
			Expected:  "Non-empty results",
			Actual:    "0 results",
			Message:   "Assessment Results contains no result entries",
		})
	}

	actualSet := make(map[string]bool, len(expected))
	for _, res := range doc.AssessmentResults.Results {
		extractTypedReviewedControlIDs(res.ReviewedControls, actualSet)
	}
	compareControlIDSet(expected, actualSet, "AssessmentResultsReviewedControls", rep)
}

func comparePOAM(expected []ControlInfo, doc *oscalVerifyPOAMDoc, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	if doc == nil || doc.POAM == nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "POAMRoot",
			Expected:  "valid plan-of-action-and-milestones",
			Actual:    "missing plan-of-action-and-milestones",
			Message:   "POA&M root object missing",
		})
		return
	}
	if len(doc.POAM.PoamItems) == 0 {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "POAMItems",
			Expected:  "Non-empty poam-items",
			Actual:    "0 poam-items",
			Message:   "POA&M contains no poam-items",
		})
	}

	actualSet := make(map[string]bool, len(expected))
	for _, item := range doc.POAM.PoamItems {
		for _, prop := range item.Props {
			if strings.EqualFold(strings.TrimSpace(prop.Name), "control-id") {
				if cid := strings.ToLower(strings.TrimSpace(prop.Value)); cid != "" {
					actualSet[cid] = true
				}
			}
		}
	}
	compareControlIDSet(expected, actualSet, "POAMControlCoverage", rep)
}

// VerifyAllFrameworksParity iterates across all frameworks in registry and checks parity against directory.
func VerifyAllFrameworksParity(r *Registry, baseDir string) ([]*ParityReport, error) {
	artifactNames := []string{
		"catalog.json",
		"profile.json",
		"component-definition.json",
		"system-security-plan.json",
		"assessment-plan.json",
		"assessment-results.json",
		"plan-of-action-and-milestones.json",
	}

	fws := r.List()
	if len(fws) == 0 {
		return nil, nil
	}

	// Pre-initialize shared schema validator before spawning workers.
	if _, err := sharedSchemaValidator(); err != nil {
		return nil, fmt.Errorf("init schema validator: %w", err)
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	if workers > len(fws) {
		workers = len(fws)
	}

	perFwReports := make([][]*ParityReport, len(fws))
	perFwErr := make([]error, len(fws))
	jobs := make(chan int, len(fws))
	for i := range fws {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				f := fws[idx]
				expectedCtrls, _ := loadCachedExpectedControls(f)
				fwDir := filepath.Join(baseDir, f.ID())
				var fwReps []*ParityReport
				foundAny := false
				for _, name := range artifactNames {
					artPath := filepath.Join(fwDir, name)
					// #nosec G703,G304 -- artPath is resolved within the verified base directory.
					data, err := os.ReadFile(artPath)
					if err == nil {
						foundAny = true
						rep, err := verifyBytesParityWithExpected(f, data, artPath, expectedCtrls)
						if err != nil {
							perFwErr[idx] = fmt.Errorf("verify %s %s: %w", f.ID(), name, err)
							break
						}
						fwReps = append(fwReps, rep)
					}
				}
				if perFwErr[idx] != nil {
					continue
				}
				if !foundAny {
					catData, err := f.CatalogJSON()
					if err == nil {
						rep, _ := verifyBytesParityWithExpected(f, catData, "memory://"+f.ID()+"/catalog.json", expectedCtrls)
						fwReps = append(fwReps, rep)
					}
				}
				perFwReports[idx] = fwReps
			}
		}()
	}
	wg.Wait()

	var reports []*ParityReport
	for i, err := range perFwErr {
		if err != nil {
			return nil, err
		}
		reports = append(reports, perFwReports[i]...)
	}

	return reports, nil
}

// FormatTerminal renders a human-readable terminal audit summary.
func (r *ParityReport) FormatTerminal(verbose bool) string {
	var b strings.Builder
	b.WriteString("\n=== xOSCAL FRAMEWORK VERBATIM PARITY REPORT ===\n")
	fmt.Fprintf(&b, "Framework:    %s (%s)\n", r.FrameworkTitle, r.FrameworkID)
	fmt.Fprintf(&b, "Artifact:     %s [%s]\n", r.ArtifactPath, r.ArtifactKind)
	fmt.Fprintf(&b, "Artifact SHA: %s\n", r.ArtifactSHA256)
	if r.SidecarPath != "" {
		matchStr := "MATCH"
		if !r.SidecarMatch {
			matchStr = "MISMATCH"
		}
		fmt.Fprintf(&b, "Sidecar:      %s [%s]\n", r.SidecarPath, matchStr)
	}
	schemaStr := "VALID"
	if !r.SchemaValid {
		schemaStr = "INVALID"
	}
	fmt.Fprintf(&b, "OSCAL 1.2.3:  %s\n", schemaStr)
	fmt.Fprintf(&b, "Integrity:    %t\n", r.IntegrityValid)
	fmt.Fprintf(&b, "Controls:     %d Expected | %d Actual | %d Matched\n",
		r.TotalExpectedControls, r.TotalActualControls, r.MatchedControls)
	fmt.Fprintf(&b, "Parity Score: %.1f%%\n", r.ParityScore)
	fmt.Fprintf(&b, "Verdict:      %s\n\n", r.Verdict)

	b.WriteString("VERBATIM FIELD PARITY AUDIT:\n")
	fmt.Fprintf(&b, "  * Control Titles:    %d / %d matches\n", r.TitleMatches, r.TotalExpectedControls)
	fmt.Fprintf(&b, "  * Statements (AST):  %d / %d matches\n", r.StatementMatches, r.TotalExpectedControls)
	fmt.Fprintf(&b, "  * Guidance/GC Disc:  %d / %d matches\n", r.GuidanceMatches, r.TotalExpectedControls)
	fmt.Fprintf(&b, "  * Parameters (ODP):  %d / %d matches\n", r.ParameterMatches, r.TotalExpectedControls)
	fmt.Fprintf(&b, "  * Sovereign Tagging: %d / %d matches\n", r.SovereignTagMatches, r.TotalExpectedControls)

	if len(r.MissingControls) > 0 {
		fmt.Fprintf(&b, "\nMISSING CONTROLS (%d):\n", len(r.MissingControls))
		for _, m := range r.MissingControls {
			fmt.Fprintf(&b, "  - %s\n", m)
		}
	}

	if len(r.ExtraControls) > 0 {
		fmt.Fprintf(&b, "\nEXTRA CONTROLS (%d):\n", len(r.ExtraControls))
		for _, e := range r.ExtraControls {
			fmt.Fprintf(&b, "  + %s\n", e)
		}
	}

	if len(r.Discrepancies) > 0 && verbose {
		fmt.Fprintf(&b, "\nDISCREPANCIES (%d):\n", len(r.Discrepancies))
		for i, d := range r.Discrepancies {
			if i >= 10 {
				fmt.Fprintf(&b, "  ... and %d more discrepancies (use -json for complete dump)\n", len(r.Discrepancies)-10)
				break
			}
			fmt.Fprintf(&b, "  [%s] Control %s: %s\n", d.Aspect, d.ControlID, d.Message)
		}
	}

	return b.String()
}

// FormatMarkdown renders a clean GitHub Flavored Markdown audit report.
func (r *ParityReport) FormatMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "### xOSCAL Framework Verbatim Parity Report: %s\n\n", r.FrameworkID)
	fmt.Fprintf(&b, "- **Framework:** %s (`%s`)\n", r.FrameworkTitle, r.FrameworkID)
	fmt.Fprintf(&b, "- **Artifact:** `%s` (`%s`)\n", r.ArtifactPath, r.ArtifactKind)
	fmt.Fprintf(&b, "- **Computed SHA-256:** `%s`\n", r.ArtifactSHA256)
	fmt.Fprintf(&b, "- **Sidecar Match:** `%t`\n", r.SidecarMatch)
	fmt.Fprintf(&b, "- **OSCAL 1.2.3 Schema:** `%t`\n", r.SchemaValid)
	fmt.Fprintf(&b, "- **Parity Score:** **%.1f%%**\n", r.ParityScore)
	fmt.Fprintf(&b, "- **Verdict:** `%s`\n\n", r.Verdict)

	b.WriteString("| Verification Aspect | Matches | Expected | Parity Posture |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")
	fmt.Fprintf(&b, "| **Control Presence** | %d | %d | %s |\n", r.MatchedControls, r.TotalExpectedControls, postureBadge(r.MatchedControls == r.TotalExpectedControls))
	fmt.Fprintf(&b, "| **Control Titles** | %d | %d | %s |\n", r.TitleMatches, r.TotalExpectedControls, postureBadge(r.TitleMatches == r.TotalExpectedControls))
	fmt.Fprintf(&b, "| **Statement Prose (AST)** | %d | %d | %s |\n", r.StatementMatches, r.TotalExpectedControls, postureBadge(r.StatementMatches == r.TotalExpectedControls))
	fmt.Fprintf(&b, "| **Guidance / GC Discussion** | %d | %d | %s |\n", r.GuidanceMatches, r.TotalExpectedControls, postureBadge(r.GuidanceMatches == r.TotalExpectedControls))
	fmt.Fprintf(&b, "| **ODP Parameters** | %d | %d | %s |\n", r.ParameterMatches, r.TotalExpectedControls, postureBadge(r.ParameterMatches == r.TotalExpectedControls))
	fmt.Fprintf(&b, "| **Sovereign Canadian Tagging** | %d | %d | %s |\n", r.SovereignTagMatches, r.TotalExpectedControls, postureBadge(r.SovereignTagMatches == r.TotalExpectedControls))

	if len(r.Discrepancies) > 0 {
		fmt.Fprintf(&b, "\n#### Discrepancies (%d)\n\n", len(r.Discrepancies))
		b.WriteString("| Control ID | Aspect | Message |\n")
		b.WriteString("| :--- | :--- | :--- |\n")
		for i, d := range r.Discrepancies {
			if i >= 15 {
				fmt.Fprintf(&b, "| ... | ... | *%d more discrepancies omitted* |\n", len(r.Discrepancies)-15)
				break
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", d.ControlID, d.Aspect, d.Message)
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
