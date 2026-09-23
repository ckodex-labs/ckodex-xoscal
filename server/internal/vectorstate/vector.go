package vectorstate

import (
	"fmt"
	"strings"
	"time"
)

// PresenceState represents the presence dimension (P).
type PresenceState string

const (
	PresenceEmpty    PresenceState = "EMPTY"
	PresencePresent  PresenceState = "PRESENT"
	PresenceUnknown  PresenceState = "UNKNOWN"
	PresenceRedacted PresenceState = "REDACTED"
)

// ValenceState represents the valence dimension (V).
type ValenceState string

const (
	ValencePositive   ValenceState = "POSITIVE"
	ValenceNegative   ValenceState = "NEGATIVE"
	ValenceNeutral    ValenceState = "NEUTRAL"
	ValenceMixed      ValenceState = "MIXED"
	ValenceUnresolved ValenceState = "UNRESOLVED"
)

// CoherenceState represents the coherence dimension (C).
type CoherenceState string

const (
	Coherent          CoherenceState = "COHERENT"
	PartiallyCoherent CoherenceState = "PARTIALLY_COHERENT"
	Decoherent        CoherenceState = "DECOHERENT"
	Reconciling       CoherenceState = "RECONCILING"
)

// EvidenceStatus represents the evidence dimension (E).
type EvidenceStatus string

const (
	EvidenceVerified   EvidenceStatus = "VERIFIED"
	EvidenceUnverified EvidenceStatus = "UNVERIFIED"
	EvidenceExpired    EvidenceStatus = "EXPIRED"
	EvidenceTampered   EvidenceStatus = "TAMPERED"
)

// LifecycleMode represents the runtime operational mode (L).
type LifecycleMode string

const (
	ModeNormal      LifecycleMode = "NORMAL"
	ModeDegraded    LifecycleMode = "DEGRADED"
	ModeSafeHold    LifecycleMode = "SAFE_HOLD"
	ModeQuarantined LifecycleMode = "QUARANTINED"
	ModeFailed      LifecycleMode = "FAILED"
)

// ControlFinding provides detailed per-control posture.
type ControlFinding struct {
	ControlID   string   `json:"control_id"`
	Component   string   `json:"component"`
	Status      string   `json:"status"` // "coherent", "degraded", "incomplete", "violation"
	Valence     string   `json:"valence"`
	Description string   `json:"description"`
	Remediation string   `json:"remediation,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
}

// VectorState represents the formal multi-dimensional state product S(e,t) = <P, V, A, C, E, L, τ>.
type VectorState struct {
	ArtifactName   string           `json:"artifact_name"`
	ArtifactKind   string           `json:"artifact_kind"`
	Presence       PresenceState    `json:"presence"`
	Valence        ValenceState     `json:"valence"`
	AntiCount      int              `json:"anti_count"`
	Coherence      CoherenceState   `json:"coherence"`
	Evidence       EvidenceStatus   `json:"evidence_status"`
	Lifecycle      LifecycleMode    `json:"lifecycle_mode"`
	Epoch          time.Time        `json:"epoch"`
	TotalControls  int              `json:"total_controls"`
	PassingCount   int              `json:"passing_count"`
	DegradedCount  int              `json:"degraded_count"`
	DerogatedCount int              `json:"derogated_count"`
	Findings       []ControlFinding `json:"findings"`
}

// FormatTable returns a structured, human-readable terminal report.
func (v *VectorState) FormatTable(useColor bool) string {
	var sb strings.Builder

	reset := ""
	green := ""
	yellow := ""
	red := ""
	cyan := ""
	magenta := ""
	bold := ""

	if useColor {
		reset = "\033[0m"
		green = "\033[32m"
		yellow = "\033[33m"
		red = "\033[31m"
		cyan = "\033[36m"
		magenta = "\033[35m"
		bold = "\033[1m"
	}

	sb.WriteString(fmt.Sprintf("\n%s=== xOSCAL GOVERNANCE VECTOR POSTURE ===%s\n", bold, reset))
	sb.WriteString(fmt.Sprintf("Artifact:  %s (%s)\n", v.ArtifactName, v.ArtifactKind))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n\n", v.Epoch.Format(time.RFC3339)))

	// Vector State Header
	sb.WriteString(fmt.Sprintf("%sVector Product: S(e,t) = <P=%s, V=%s, A=%d, C=%s, E=%s, L=%s>%s\n\n",
		bold, v.Presence, v.Valence, v.AntiCount, v.Coherence, v.Evidence, v.Lifecycle, reset))

	// Status line
	statusColor := green
	if v.Lifecycle == ModeDegraded || v.Lifecycle == ModeSafeHold {
		statusColor = yellow
	} else if v.Lifecycle == ModeQuarantined || v.Lifecycle == ModeFailed {
		statusColor = red
	}
	sb.WriteString(fmt.Sprintf("Operational Posture: %s%s%s (Coherence: %s)\n",
		statusColor, v.Lifecycle, reset, v.Coherence))
	sb.WriteString(fmt.Sprintf("Controls Summary:    %d Total | %s%d Passing%s | %s%d Degraded%s | %s%d Derogated%s | %s%d Anti-Conflicts%s\n\n",
		v.TotalControls, green, v.PassingCount, reset, yellow, v.DegradedCount, reset, magenta, v.DerogatedCount, reset, red, v.AntiCount, reset))

	sb.WriteString(fmt.Sprintf("%s%-14s %-20s %-16s %s%s\n",
		bold, "CONTROL", "COMPONENT", "POSTURE", "DESCRIPTION / REMEDIATION", reset))
	sb.WriteString(strings.Repeat("-", 85) + "\n")

	for _, f := range v.Findings {
		col := green
		badge := "PASS"
		if f.Status == "degraded" {
			col = yellow
			badge = "DEGRADED"
		} else if f.Status == "violation" {
			col = red
			badge = "VIOLATION"
		} else if f.Status == "incomplete" {
			col = cyan
			badge = "INCOMPLETE"
		} else if f.Status == "derogated" {
			col = magenta
			badge = "DEROGATED"
		} else if f.Status == "expired_derogation" {
			col = red
			badge = "EXPIRED DEROG"
		}

		desc := f.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-14s %-20s %s%-16s%s %s\n",
			f.ControlID, f.Component, col, badge, reset, desc))

		if f.Remediation != "" {
			sb.WriteString(fmt.Sprintf("  └─ %sNotice/Fix:%s %s\n", yellow, reset, f.Remediation))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}
