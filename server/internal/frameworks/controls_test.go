package frameworks_test

import (
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
)

func TestLoadAndSearchControls(t *testing.T) {
	fw, ok := frameworks.Get("cccs-itsp-10-033")
	if !ok {
		t.Fatalf("expected cccs-itsp-10-033 to be registered")
	}

	controls, err := frameworks.LoadControls(fw)
	if err != nil {
		t.Fatalf("failed to load controls: %v", err)
	}

	if len(controls) == 0 {
		t.Fatalf("expected controls, got 0")
	}

	// Verify we can find ac-2
	ac2, found := frameworks.GetControl(controls, "ac-2")
	if !found {
		t.Fatalf("expected to find control ac-2")
	}

	if ac2.Title == "" {
		t.Errorf("expected non-empty title for ac-2")
	}

	// Verify search
	matches := frameworks.SearchControls(controls, "account", 10)
	if len(matches) == 0 {
		t.Errorf("expected search matches for 'account'")
	}
}

func TestCompareFrameworks(t *testing.T) {
	fromFw, ok := frameworks.Get("cccs-itsg-33")
	if !ok {
		t.Fatalf("expected cccs-itsg-33 to be registered")
	}

	toFw, ok := frameworks.Get("cccs-itsp-10-033")
	if !ok {
		t.Fatalf("expected cccs-itsp-10-033 to be registered")
	}

	comp, err := frameworks.CompareFrameworks(fromFw, toFw)
	if err != nil {
		t.Fatalf("failed to compare frameworks: %v", err)
	}

	if comp.ToTotal <= comp.FromTotal {
		t.Errorf("expected target framework to have more controls than base, got from=%d, to=%d", comp.FromTotal, comp.ToTotal)
	}

	if comp.AddedCount == 0 {
		t.Errorf("expected added controls, got 0")
	}

	termReport := comp.FormatTerminal()
	if termReport == "" {
		t.Errorf("expected non-empty terminal report")
	}

	mdReport := comp.FormatMarkdown()
	if mdReport == "" {
		t.Errorf("expected non-empty markdown report")
	}
}
