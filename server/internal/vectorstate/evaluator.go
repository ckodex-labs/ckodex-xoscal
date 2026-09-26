package vectorstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/derogation"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

// EvaluateArtifact inspects an OSCAL JSON artifact, performs schema validation,
// checks evidence integrity, and computes the complete Vector State.
func EvaluateArtifact(artifactPath string, evidenceDir string) (*VectorState, error) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Detect Kind across all 8 official OSCAL 1.2.3 model types
	var kind schemavalidate.ArtifactKind
	var rootKey string
	for key := range root {
		switch key {
		case "component-definition":
			kind = schemavalidate.KindComponentDefinition
			rootKey = key
		case "system-security-plan":
			kind = schemavalidate.KindSSP
			rootKey = key
		case "catalog":
			kind = schemavalidate.KindCatalog
			rootKey = key
		case "profile":
			kind = schemavalidate.KindProfile
			rootKey = key
		case "assessment-results":
			kind = schemavalidate.KindAssessmentResults
			rootKey = key
		case "plan-of-action-and-milestones":
			kind = schemavalidate.KindPOAM
			rootKey = key
		case "assessment-plan":
			kind = schemavalidate.KindAssessmentPlan
			rootKey = key
		case "mapping-collection", "mapping":
			kind = schemavalidate.KindMapping
			rootKey = key
		}
		if rootKey != "" {
			break
		}
	}

	if rootKey == "" {
		return nil, fmt.Errorf("unrecognized or unsupported OSCAL artifact root key")
	}

	state := &VectorState{
		ArtifactName: filepath.Base(artifactPath),
		ArtifactKind: kind.Name,
		Presence:     PresencePresent,
		Valence:      ValencePositive,
		AntiCount:    0,
		Coherence:    Coherent,
		Evidence:     EvidenceVerified,
		Lifecycle:    ModeNormal,
		Epoch:        time.Now().UTC(),
		Findings:     make([]ControlFinding, 0),
	}

	// Step 1: Strict Schema Validation
	v, err := schemavalidate.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	if err := v.Validate(data, kind); err != nil {
		state.Lifecycle = ModeFailed
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
		state.AntiCount++
		state.Findings = append(state.Findings, ControlFinding{
			ControlID:   "SCHEMA-ROOT",
			Component:   state.ArtifactName,
			Status:      "violation",
			Valence:     "NEGATIVE",
			Description: fmt.Sprintf("OSCAL %s schema validation failed", schemavalidate.SchemaVersion),
			Remediation: err.Error(),
		})
		return state, nil
	}

	// Step 2: Control Extraction & Evaluation
	targetObj, ok := root[rootKey].(map[string]interface{})
	if !ok {
		return state, nil
	}

	// Load derogation store if available
	derogPath := filepath.Join(".xoscal", "derogations.json")
	if _, err := os.Stat(derogPath); os.IsNotExist(err) {
		derogPath = filepath.Join(filepath.Dir(artifactPath), ".xoscal", "derogations.json")
	}
	derogStore, _ := derogation.Load(derogPath)

	// Component-Definition Evaluation
	if rootKey == "component-definition" {
		comps, _ := targetObj["components"].([]interface{})
		for _, rawComp := range comps {
			compMap, ok := rawComp.(map[string]interface{})
			if !ok {
				continue
			}
			compTitle := "Component"
			if t, ok := compMap["title"].(string); ok && t != "" {
				compTitle = t
			} else if tMap, ok := compMap["title"].(map[string]interface{}); ok {
				if s, ok := tMap["value"].(string); ok {
					compTitle = s
				}
			}

			ctrlImpls, _ := compMap["control-implementations"].([]interface{})
			for _, rawImpl := range ctrlImpls {
				implMap, ok := rawImpl.(map[string]interface{})
				if !ok {
					continue
				}
				reqs, _ := implMap["implemented-requirements"].([]interface{})
				for _, rawReq := range reqs {
					reqMap, ok := rawReq.(map[string]interface{})
					if !ok {
						continue
					}
					ctrlID, _ := reqMap["control-id"].(string)
					desc, _ := reqMap["description"].(string)
					evaluateControl(ctrlID, compTitle, desc, state, evidenceDir, derogStore)
				}
			}
		}
	} else if rootKey == "system-security-plan" {
		// System Security Plan Evaluation
		ctrlImpl, _ := targetObj["control-implementation"].(map[string]interface{})
		if ctrlImpl != nil {
			reqs, _ := ctrlImpl["implemented-requirements"].([]interface{})
			for _, rawReq := range reqs {
				reqMap, ok := rawReq.(map[string]interface{})
				if !ok {
					continue
				}
				ctrlID, _ := reqMap["control-id"].(string)
				desc, _ := reqMap["description"].(string)
				if desc == "" {
					if rem, ok := reqMap["remarks"].(string); ok {
						desc = rem
					}
				}
				evaluateControl(ctrlID, "System Security Plan", desc, state, evidenceDir, derogStore)
			}
		}
	} else if rootKey == "catalog" {
		// Catalog Evaluation
		evalCatalogControls(targetObj, state, evidenceDir, derogStore)
	}

	// Synthesize overall vector dimensions
	if state.Evidence == EvidenceTampered {
		state.Lifecycle = ModeQuarantined
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
	} else if state.AntiCount > 0 {
		state.Lifecycle = ModeFailed
		state.Coherence = Decoherent
		state.Valence = ValenceNegative
	} else if state.DegradedCount > 0 {
		state.Lifecycle = ModeDegraded
		state.Coherence = PartiallyCoherent
		state.Valence = ValenceMixed
	} else if state.DerogatedCount > 0 {
		state.Lifecycle = ModeSafeHold
		state.Coherence = PartiallyCoherent
		state.Valence = ValenceMixed
	} else {
		state.Lifecycle = ModeNormal
		state.Coherence = Coherent
		state.Valence = ValencePositive
	}

	return state, nil
}

func evaluateControl(
	ctrlID string,
	compTitle string,
	desc string,
	state *VectorState,
	evidenceDir string,
	derogStore *derogation.Store,
) {
	state.TotalControls++

	finding := ControlFinding{
		ControlID:   ctrlID,
		Component:   compTitle,
		Status:      "coherent",
		Valence:     "POSITIVE",
		Description: desc,
	}

	// Check description quality
	if strings.TrimSpace(desc) == "" {
		finding.Status = "incomplete"
		finding.Valence = "NEUTRAL"
		finding.Remediation = "Provide detailed implementation statement prose"
		state.DegradedCount++
	} else if len(strings.TrimSpace(desc)) < 15 {
		finding.Status = "degraded"
		finding.Valence = "MIXED"
		finding.Remediation = "Implementation statement is overly terse; elaborate mechanism and verification"
		state.DegradedCount++
	} else {
		state.PassingCount++
	}

	// Check evidence if directory supplied
	if evidenceDir != "" {
		evFiles, _ := filepath.Glob(filepath.Join(evidenceDir, fmt.Sprintf("*%s*", ctrlID)))
		if len(evFiles) == 0 {
			finding.Status = "degraded"
			finding.Valence = "MIXED"
			finding.Remediation = fmt.Sprintf("No corroborating evidence blob found matching control %s in %s", ctrlID, evidenceDir)
			if state.Evidence != EvidenceTampered {
				state.Evidence = EvidenceUnverified
			}
		} else {
			for _, ev := range evFiles {
				if strings.HasSuffix(ev, ".sha256") {
					continue
				}
				finding.Evidence = append(finding.Evidence, filepath.Base(ev))
				// Verify SHA-256 sidecar if present
				sidecar := ev + ".sha256"
				if sideData, err := os.ReadFile(sidecar); err == nil {
					evData, _ := os.ReadFile(ev)
					sum := sha256.Sum256(evData)
					expected := strings.TrimSpace(string(sideData))
					actual := hex.EncodeToString(sum[:])
					if !strings.EqualFold(expected, actual) {
						finding.Status = "violation"
						finding.Valence = "NEGATIVE"
						finding.Remediation = fmt.Sprintf("Evidence blob %s checksum mismatch! Expected %s, got %s", filepath.Base(ev), expected, actual)
						state.AntiCount++
						state.Evidence = EvidenceTampered
					}
				}
			}
		}
	}

	// Check derogation store
	if derogStore != nil {
		if active := derogStore.GetActiveForControl(ctrlID, time.Now()); active != nil {
			if finding.Status == "degraded" || finding.Status == "incomplete" {
				state.DegradedCount--
			} else if finding.Status == "coherent" {
				state.PassingCount--
			}
			finding.Status = "derogated"
			finding.Valence = "MIXED"
			finding.Remediation = fmt.Sprintf("Derogated by %s (TTL remaining: %s; justification: %s)",
				active.Authority, active.TimeRemaining(time.Now()).Round(time.Hour), active.Justification)
			state.DerogatedCount++
		} else if expired := derogStore.GetExpiredForControl(ctrlID, time.Now()); expired != nil {
			if finding.Status == "coherent" {
				state.PassingCount--
			} else if finding.Status == "degraded" || finding.Status == "incomplete" {
				state.DegradedCount--
			}
			finding.Status = "expired_derogation"
			finding.Valence = "NEGATIVE"
			finding.Remediation = fmt.Sprintf("CRITICAL: Derogation expired on %s! Authority %s must re-evaluate",
				expired.ExpiresAt.Format(time.RFC3339), expired.Authority)
			state.AntiCount++
		}
	}

	state.Findings = append(state.Findings, finding)
}

func evalCatalogControls(
	targetObj map[string]interface{},
	state *VectorState,
	evidenceDir string,
	derogStore *derogation.Store,
) {
	var processCtrl func(rawCtrl interface{}, compTitle string)
	processCtrl = func(rawCtrl interface{}, compTitle string) {
		ctrlMap, ok := rawCtrl.(map[string]interface{})
		if !ok {
			return
		}
		ctrlID := ""
		if idVal, ok := ctrlMap["id"].(string); ok {
			ctrlID = idVal
		} else if idMap, ok := ctrlMap["id"].(map[string]interface{}); ok {
			ctrlID, _ = idMap["value"].(string)
		}
		if ctrlID == "" {
			return
		}

		title := ""
		if tVal, ok := ctrlMap["title"].(string); ok {
			title = tVal
		} else if tMap, ok := ctrlMap["title"].(map[string]interface{}); ok {
			title, _ = tMap["value"].(string)
		}

		desc := title
		if parts, ok := ctrlMap["parts"].([]interface{}); ok {
			for _, p := range parts {
				if pMap, ok := p.(map[string]interface{}); ok {
					if pStr, ok := pMap["prose"].(string); ok && pStr != "" {
						desc = pStr
						break
					} else if proseList, ok := pMap["prose"].([]interface{}); ok && len(proseList) > 0 {
						if pStr, ok := proseList[0].(string); ok && pStr != "" {
							desc = pStr
							break
						}
					}
				}
			}
		}

		evaluateControl(ctrlID, compTitle, desc, state, evidenceDir, derogStore)
	}

	if ctrls, ok := targetObj["controls"].([]interface{}); ok {
		for _, c := range ctrls {
			processCtrl(c, "Catalog Root")
		}
	}

	var processGroup func(rawGroup interface{})
	processGroup = func(rawGroup interface{}) {
		gMap, ok := rawGroup.(map[string]interface{})
		if !ok {
			return
		}
		gTitle := "Catalog Group"
		if t, ok := gMap["title"].(string); ok && t != "" {
			gTitle = t
		}
		if ctrls, ok := gMap["controls"].([]interface{}); ok {
			for _, c := range ctrls {
				processCtrl(c, gTitle)
			}
		}
		if subGroups, ok := gMap["groups"].([]interface{}); ok {
			for _, sub := range subGroups {
				processGroup(sub)
			}
		}
	}

	if groups, ok := targetObj["groups"].([]interface{}); ok {
		for _, g := range groups {
			processGroup(g)
		}
	}
}
