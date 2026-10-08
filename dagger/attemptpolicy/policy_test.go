package attemptpolicy

import (
	"strings"
	"testing"
)

func TestObservationLabelBoundary(t *testing.T) {
	for _, label := range []string{"local-20261008-1", "github.37777996669_2", strings.Repeat("a", 80)} {
		if err := Validate(label); err != nil {
			t.Fatalf("valid selection failed: %q: %v", label, err)
		}
	}
	for _, label := range []string{"", strings.Repeat("a", 81), "-leading", "a\n", "a\x00", "a/b", "a b", "é", "$(id)", "a\"b"} {
		if err := Validate(label); err == nil {
			t.Fatalf("invalid selection admitted: %q", label)
		}
	}
}
