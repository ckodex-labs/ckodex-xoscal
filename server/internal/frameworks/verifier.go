package frameworks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/integrity"
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

	// 5. Verbatim Control Parity Verification (Catalog only)
	if kind.RootKey == "catalog" {
		expectedCtrls, err := LoadControls(f)
		if err != nil {
			return nil, fmt.Errorf("load expected controls for %s: %w", f.ID(), err)
		}

		actualCtrls, err := LoadControlsFromBytes(data, f.ID())
		if err != nil {
			return nil, fmt.Errorf("load actual controls from artifact: %w", err)
		}

		compareControls(expectedCtrls, actualCtrls, rep)
	} else if kind.RootKey == "profile" {
		// Profile verification
		expectedProfileBytes, err := f.ProfileJSON()
		if err != nil {
			return nil, fmt.Errorf("load expected profile for %s: %w", f.ID(), err)
		}
		compareProfiles(expectedProfileBytes, data, rep)
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
		if rep.SchemaValid && rep.IntegrityValid && rep.SidecarMatch {
			rep.ParityScore = 100.0
		}
	}

	if len(rep.Discrepancies) > 0 || len(rep.MissingControls) > 0 || len(rep.ExtraControls) > 0 || !rep.SchemaValid || !rep.IntegrityValid || !rep.SidecarMatch {
		rep.Verdict = VerdictDeviationDetected
	}

	return rep, nil
}

func compareControls(expected, actual []ControlInfo, rep *ParityReport) {
	rep.TotalExpectedControls = len(expected)
	rep.TotalActualControls = len(actual)

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

	// Compare common controls
	for id, exp := range expectedMap {
		act, ok := actualMap[id]
		if !ok {
			continue
		}
		rep.MatchedControls++

		// 1. Title
		if norm(exp.Title) == norm(act.Title) {
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

		// 2. Statements
		expStmts := normSlice(exp.Statements)
		actStmts := normSlice(act.Statements)
		if slicesEqual(expStmts, actStmts) {
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

		// 3. Guidance & Canadian Guidance
		expGuidance := norm(exp.Guidance) + " // " + norm(exp.CanadianGuidance)
		actGuidance := norm(act.Guidance) + " // " + norm(act.CanadianGuidance)
		if expGuidance == actGuidance {
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

		// 5. Canadian Sovereign Tagging
		if exp.IsCanadian == act.IsCanadian {
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

func compareProfiles(expectedJSON, actualJSON []byte, rep *ParityReport) {
	var expRoot, actRoot map[string]interface{}
	_ = json.Unmarshal(expectedJSON, &expRoot)
	_ = json.Unmarshal(actualJSON, &actRoot)

	expProf, _ := expRoot["profile"].(map[string]interface{})
	actProf, _ := actRoot["profile"].(map[string]interface{})

	if expProf == nil || actProf == nil {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "ProfileRoot",
			Expected:  "valid profile",
			Actual:    "invalid profile structure",
			Message:   "Profile root object missing",
		})
		return
	}

	expImports, _ := expProf["imports"].([]interface{})
	actImports, _ := actProf["imports"].([]interface{})
	if len(expImports) == len(actImports) {
		rep.StatementMatches++
	} else {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "ProfileImports",
			Expected:  fmt.Sprintf("%d imports", len(expImports)),
			Actual:    fmt.Sprintf("%d imports", len(actImports)),
			Message:   "Profile import count mismatch",
		})
	}
}

// VerifyAllFrameworksParity iterates across all frameworks in registry and checks parity against directory.
func VerifyAllFrameworksParity(r *Registry, baseDir string) ([]*ParityReport, error) {
	var reports []*ParityReport

	for _, f := range r.List() {
		catPath := filepath.Join(baseDir, f.ID(), "catalog.json")
		if _, err := os.Stat(catPath); err == nil {
			rep, err := VerifyArtifactParity(f, catPath)
			if err != nil {
				return nil, fmt.Errorf("verify %s catalog: %w", f.ID(), err)
			}
			reports = append(reports, rep)
		} else {
			// Compare against in-memory export
			catData, err := f.CatalogJSON()
			if err == nil {
				rep, _ := VerifyBytesParity(f, catData, "memory://"+f.ID()+"/catalog.json")
				reports = append(reports, rep)
			}
		}

		if f.HasProfile() {
			profPath := filepath.Join(baseDir, f.ID(), "profile.json")
			if _, err := os.Stat(profPath); err == nil {
				rep, err := VerifyArtifactParity(f, profPath)
				if err != nil {
					return nil, fmt.Errorf("verify %s profile: %w", f.ID(), err)
				}
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
