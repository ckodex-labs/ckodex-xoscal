package oscalversion

import (
	"regexp"
	"strings"
	"testing"
)

func TestCurrent(t *testing.T) {
	got := Current()
	if got != strings.TrimSpace(rawVersion) {
		t.Fatalf("Current() = %q, embedded VERSION = %q", got, rawVersion)
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(got) {
		t.Fatalf("Current() = %q, want bare semantic version", got)
	}
}
