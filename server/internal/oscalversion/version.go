// Package oscalversion exposes the single OSCAL release selected by this
// checkout. VERSION is intentionally data, so the upstream reconciler can
// advance the pin without rewriting Go source.
package oscalversion

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var rawVersion string

// Current returns the pinned OSCAL release as bare semantic version text.
func Current() string {
	return strings.TrimSpace(rawVersion)
}
