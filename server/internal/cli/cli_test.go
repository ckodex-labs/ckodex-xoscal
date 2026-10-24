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

func TestQuickstart_NonDestructive(t *testing.T) {
	out := captureOutput(func() {
		cli.RunQuickstart([]string{})
	})
	if !strings.Contains(out, "xOSCAL DEVELOPER ONBOARDING QUICKSTART") {
		t.Errorf("expected quickstart header in output")
	}
	if !strings.Contains(out, "Ready to scaffold") {
		t.Errorf("expected ready to scaffold in output")
	}
}

func TestVerifyParity_CLI(t *testing.T) {
	out := captureOutput(func() {
		cli.RunVerifyParity([]string{
			"-framework", "cybersecure-canada",
			"-dir", "../../../data/frameworks",
			"-strict=false",
		})
	})
	if !strings.Contains(out, "xOSCAL FRAMEWORK VERBATIM PARITY REPORT") {
		t.Errorf("expected parity report header, got: %s", out)
	}
	if !strings.Contains(out, "VERBATIM_PARITY") {
		t.Errorf("expected VERBATIM_PARITY in output, got: %s", out)
	}
}

func TestVerifyParity_CLI_JSON(t *testing.T) {
	out := captureOutput(func() {
		cli.RunVerifyParity([]string{
			"-framework", "cybersecure-canada",
			"-dir", "../../../data/frameworks",
			"-json",
			"-strict=false",
		})
	})
	if !strings.Contains(out, `"verdict": "VERBATIM_PARITY"`) {
		t.Errorf("expected JSON verdict VERBATIM_PARITY, got: %s", out)
	}
}
