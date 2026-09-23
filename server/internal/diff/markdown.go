package diff

import (
	"fmt"
	"strings"
)

// FormatMarkdown formats a publication-grade GitHub/GitLab PR comment table.
func (d *ComplianceDiff) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString("### 🛡️ xOSCAL Compliance Impact Report\n\n")

	// Posture Summary
	regressionAlert := "✅ **Posture Preserved**"
	if d.HasRegression {
		regressionAlert = "🚨 **Compliance Regression Detected**"
	}
	sb.WriteString(fmt.Sprintf("%s\n\n", regressionAlert))

	sb.WriteString("| Vector Dimension | Base Baseline | Pull Request (Head) | Impact |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| **Lifecycle Mode** | `%s` | `%s` | %s |\n",
		d.BaseVector.Lifecycle, d.HeadVector.Lifecycle, formatLifecycleDelta(d.BaseVector.Lifecycle, d.HeadVector.Lifecycle)))
	sb.WriteString(fmt.Sprintf("| **Coherence** | `%s` | `%s` | %s |\n",
		d.BaseVector.Coherence, d.HeadVector.Coherence, formatCoherenceDelta(d.BaseVector.Coherence, d.HeadVector.Coherence)))
	sb.WriteString(fmt.Sprintf("| **Anti-Conflicts** | `%d` | `%d` | %s |\n",
		d.BaseVector.AntiCount, d.HeadVector.AntiCount, formatAntiDelta(d.BaseVector.AntiCount, d.HeadVector.AntiCount)))
	sb.WriteString(fmt.Sprintf("| **Passing Controls** | `%d` | `%d` | %+d |\n",
		d.BaseVector.PassingCount, d.HeadVector.PassingCount, d.HeadVector.PassingCount-d.BaseVector.PassingCount))
	sb.WriteString(fmt.Sprintf("| **Derogated Controls** | `%d` | `%d` | %+d |\n\n",
		d.BaseVector.DerogatedCount, d.HeadVector.DerogatedCount, d.HeadVector.DerogatedCount-d.BaseVector.DerogatedCount))

	if len(d.Deltas) == 0 {
		sb.WriteString("*No control modifications detected. Baseline is identical.*\n")
		return sb.String()
	}

	sb.WriteString("#### Detailed Control Modifications\n\n")
	sb.WriteString("| Control ID | Component | Change | Status Delta | New Prose Snippet |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, delta := range d.Deltas {
		changeBadge := string(delta.ChangeType)
		if delta.Regression {
			changeBadge += " ⚠️"
		}

		statusDelta := fmt.Sprintf("`%s` → `%s`", delta.BaseStatus, delta.HeadStatus)
		if delta.ChangeType == ChangeAdded {
			statusDelta = fmt.Sprintf("`NEW` → `%s`", delta.HeadStatus)
		} else if delta.ChangeType == ChangeRemoved {
			statusDelta = fmt.Sprintf("`%s` → `DELETED`", delta.BaseStatus)
		}

		prose := delta.HeadProse
		if prose == "" {
			prose = delta.BaseProse
		}
		if len(prose) > 45 {
			prose = prose[:42] + "..."
		}
		// Escape pipes for markdown
		prose = strings.ReplaceAll(prose, "|", "\\|")

		sb.WriteString(fmt.Sprintf("| **%s** | %s | `%s` | %s | %s |\n",
			delta.ControlID, delta.Component, changeBadge, statusDelta, prose))
	}

	sb.WriteString("\n*Report generated deterministically by xOSCAL GitOps Engine.*\n")
	return sb.String()
}

func formatLifecycleDelta(base, head interface{}) string {
	if base == head {
		return "Stable"
	}
	return "Changed"
}

func formatCoherenceDelta(base, head interface{}) string {
	if base == head {
		return "Coherent"
	}
	return "Drift Detected"
}

func formatAntiDelta(base, head int) string {
	if head > base {
		return "⚠️ Violations Added"
	} else if head < base {
		return "Fixed Violations"
	}
	return "0 Invariant Violations"
}
