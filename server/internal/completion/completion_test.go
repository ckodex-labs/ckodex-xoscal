package completion_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/completion"
)

func TestGenerateCompletions(t *testing.T) {
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
	}

	var buf bytes.Buffer
	if err := completion.Generate("unsupported-shell", "xoscal", &buf); err == nil {
		t.Errorf("expected error for unsupported shell")
	}
}
