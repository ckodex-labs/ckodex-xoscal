package cli_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/cli"
)

// captureOutput captures stdout and stderr during f's execution.
func captureOutput(f func()) string {
	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = origOut

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintUsage(t *testing.T) {
	out := captureOutput(func() {
		cli.PrintUsage("v0.4.0-test")
	})
	if !strings.Contains(out, "Unified xOSCAL Compliance & Governance Engine") {
		t.Errorf("expected usage title in output")
	}
	if !strings.Contains(out, "controls") || !strings.Contains(out, "compare") || !strings.Contains(out, "serve") {
		t.Errorf("expected new commands in usage output")
	}
}
