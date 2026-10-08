// Package attemptpolicy validates explicit external-evidence observation labels.
package attemptpolicy

import (
	"fmt"
	"regexp"
)

var pattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)

// Validate refuses empty, ambiguous or shell-active observation labels.
func Validate(attempt string) error {
	if !pattern.MatchString(attempt) {
		return fmt.Errorf("analysis attempt must be 1–80 ASCII letters, digits, dots, underscores or hyphens, starting with a letter or digit")
	}
	return nil
}
