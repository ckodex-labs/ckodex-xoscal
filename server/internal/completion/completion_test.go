package completion_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/completion"
)

func TestGenerateCompletions(t *testing.T) {
	requiredTokens := []string{
		"utcp",
		"ontology",
		"graph",
		"feeds",
		"crosswalk",
		"blast-radius",
		"reconcile",
		"verify-checkpoint",
		"-checkpoint-dir",
		"-control",
		"-depth",
		"-format",
		"-feed",
		"-offline",
		"-target",
		"-out",
		"-base-url",
		"-grpc-host",
		"-manual-url",
		"-manual-file",
		"-file",
		"-json",
		"-validate-constraints",
		"-oscal-cli",
		"-db",
		"-dsn",
	}

	for _, sh := range completion.SupportedShells {
		var buf bytes.Buffer
		err := completion.Generate(sh, "xoscal", &buf)
		if err != nil {
			t.Fatalf("failed to generate %s completion: %v", sh, err)
		}
		out := buf.String()
		if len(out) == 0 {
			t.Errorf("expected non-empty completion for %s", sh)
		}
		if !strings.Contains(out, "xoscal") {
			t.Errorf("expected completion script to contain binary name 'xoscal'")
		}
		for _, tok := range requiredTokens {
			if !strings.Contains(out, tok) {
				t.Errorf("%s completion missing expected flag/token %q", sh, tok)
			}
		}
	}

	var buf bytes.Buffer
	if err := completion.Generate("unsupported-shell", "xoscal", &buf); err == nil {
		t.Errorf("expected error for unsupported shell")
	}
}
