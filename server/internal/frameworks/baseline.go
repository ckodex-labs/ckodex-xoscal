package frameworks

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed baseline_inventory.json
var embeddedBaselineBytes []byte

// FrameworkBaseline pins the independent cryptographic and structural baseline
// for a registered framework so that disk-backed manifest frameworks cannot
// pass parity verification via tautological self-comparison if tampered on disk.
type FrameworkBaseline struct {
	FrameworkID    string            `json:"framework_id"`
	CatalogSHA256  string            `json:"catalog_sha256"`
	TotalControls  int               `json:"total_controls"`
	BaseControls   int               `json:"base_controls"`
	Enhancements   int               `json:"enhancements"`
	ParameterCount int               `json:"parameter_count"`
	GuidanceCount  int               `json:"guidance_count"`
	ControlsDigest string            `json:"controls_digest"`
	ControlDigests map[string]string `json:"control_digests"`
}

// BaselineInventory holds the pinned baselines for all 36 registered frameworks.
type BaselineInventory struct {
	Frameworks map[string]FrameworkBaseline `json:"frameworks"`
}

var (
	baselineOnce sync.Once
	baselineInv  BaselineInventory
)

// LookupBaseline returns the pinned baseline for a framework ID if present in the embedded inventory.
func LookupBaseline(frameworkID string) (FrameworkBaseline, bool) {
	baselineOnce.Do(func() {
		_ = json.Unmarshal(embeddedBaselineBytes, &baselineInv)
	})
	if len(baselineInv.Frameworks) == 0 {
		return FrameworkBaseline{}, false
	}
	b, ok := baselineInv.Frameworks[strings.ToLower(strings.TrimSpace(frameworkID))]
	return b, ok
}

// ControlFingerprint computes a deterministic 16-hex-char (64-bit) SHA-256 prefix
// over the normalized title, statement AST items, guidance, Canadian guidance,
// ODP parameters, enhancements, and sovereign tag of a control.
func ControlFingerprint(c ControlInfo) string {
	var paramParts []string
	for k, v := range c.Parameters {
		paramParts = append(paramParts, strings.ToLower(k)+"="+norm(v))
	}
	sort.Strings(paramParts)

	var enhParts []string
	for _, e := range normSlice(c.Enhancements) {
		enhParts = append(enhParts, strings.ToLower(e))
	}
	sort.Strings(enhParts)

	payload := fmt.Sprintf(
		"id=%s|title=%s|stmt=%s|gdn=%s|cagdn=%s|params=%s|enh=%s|ca=%t",
		strings.ToLower(c.ID),
		norm(c.Title),
		strings.Join(normSlice(c.Statements), "||"),
		norm(c.Guidance),
		norm(c.CanadianGuidance),
		strings.Join(paramParts, "||"),
		strings.Join(enhParts, ","),
		c.IsCanadian,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:8])
}

// ComputeFrameworkBaseline computes a FrameworkBaseline from a Framework's current catalog.
func ComputeFrameworkBaseline(f Framework) (FrameworkBaseline, error) {
	catBytes, err := f.CatalogJSON()
	if err != nil {
		return FrameworkBaseline{}, err
	}
	catSum := sha256.Sum256(catBytes)

	ctrls, err := loadCachedControlsFromBytes(catBytes, catSum, f.ID())
	if err != nil {
		return FrameworkBaseline{}, err
	}

	b := FrameworkBaseline{
		FrameworkID:    strings.ToLower(f.ID()),
		CatalogSHA256:  hex.EncodeToString(catSum[:]),
		TotalControls:  len(ctrls),
		ControlDigests: make(map[string]string, len(ctrls)),
	}

	var sortedIDs []string
	for _, c := range ctrls {
		cid := strings.ToLower(c.ID)
		if c.IsEnhancement {
			b.Enhancements++
		} else {
			b.BaseControls++
		}
		b.ParameterCount += len(c.Parameters)
		if norm(c.Guidance) != "" {
			b.GuidanceCount++
		}
		if norm(c.CanadianGuidance) != "" {
			b.GuidanceCount++
		}
		fp := ControlFingerprint(c)
		b.ControlDigests[cid] = fp
		sortedIDs = append(sortedIDs, cid+":"+fp)
	}
	sort.Strings(sortedIDs)
	allSum := sha256.Sum256([]byte(strings.Join(sortedIDs, "\n")))
	b.ControlsDigest = hex.EncodeToString(allSum[:])
	return b, nil
}

// verifyBaselineInventory checks actual controls against the embedded independent
// baseline inventory, preventing tautological self-comparison if a disk-backed
// framework catalog is modified on disk.
func verifyBaselineInventory(f Framework, actual []ControlInfo, rep *ParityReport) {
	base, ok := LookupBaseline(f.ID())
	if !ok || len(base.ControlDigests) == 0 {
		return
	}

	if rep.TotalExpectedControls != base.TotalControls {
		rep.TotalExpectedControls = base.TotalControls
	}

	if len(actual) != base.TotalControls {
		rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
			ControlID: "*",
			Aspect:    "BaselineControlCount",
			Expected:  fmt.Sprintf("%d", base.TotalControls),
			Actual:    fmt.Sprintf("%d", len(actual)),
			Message:   fmt.Sprintf("Artifact control count (%d) deviates from pinned baseline inventory (%d)", len(actual), base.TotalControls),
		})
	}

	actualMap := make(map[string]ControlInfo, len(actual))
	for _, c := range actual {
		actualMap[strings.ToLower(c.ID)] = c
	}

	missingSet := make(map[string]bool, len(rep.MissingControls))
	for _, id := range rep.MissingControls {
		missingSet[strings.ToLower(id)] = true
	}
	extraSet := make(map[string]bool, len(rep.ExtraControls))
	for _, id := range rep.ExtraControls {
		extraSet[strings.ToLower(id)] = true
	}
	discrepantIDs := make(map[string]bool, len(rep.Discrepancies))
	for _, d := range rep.Discrepancies {
		discrepantIDs[strings.ToLower(d.ControlID)] = true
	}

	for id, expDigest := range base.ControlDigests {
		act, exists := actualMap[id]
		if !exists {
			if !missingSet[id] {
				rep.MissingControls = append(rep.MissingControls, id)
				rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
					ControlID: id,
					Aspect:    "BaselinePresence",
					Expected:  "PRESENT in pinned baseline",
					Actual:    "MISSING",
					Message:   fmt.Sprintf("Control '%s' in pinned baseline inventory is missing from artifact", id),
				})
			}
			continue
		}
		actDigest := ControlFingerprint(act)
		if actDigest != expDigest && !discrepantIDs[id] {
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "BaselineControlDigest",
				Expected:  expDigest,
				Actual:    actDigest,
				Message:   fmt.Sprintf("Control '%s' content fingerprint deviates from pinned baseline inventory", id),
			})
			if rep.StatementMatches > 0 {
				rep.StatementMatches--
			}
		}
	}
	sort.Strings(rep.MissingControls)

	for id := range actualMap {
		if _, exists := base.ControlDigests[id]; !exists && !extraSet[id] {
			rep.ExtraControls = append(rep.ExtraControls, id)
			rep.Discrepancies = append(rep.Discrepancies, Discrepancy{
				ControlID: id,
				Aspect:    "BaselineExtraControl",
				Expected:  "ABSENT from pinned baseline",
				Actual:    "PRESENT",
				Message:   fmt.Sprintf("Control '%s' is not present in pinned baseline inventory", id),
			})
		}
	}
	sort.Strings(rep.ExtraControls)
}
