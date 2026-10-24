package webstudio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

func TestWebStudioServer(t *testing.T) {
	srv, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     0, // random available port
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	addr, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() { _ = srv.Stop(ctx) }()

	baseURL := "http://" + addr

	// 1. Test GET / (index.html)
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("xOSCAL Dev Studio")) {
		t.Errorf("expected body to contain 'xOSCAL Dev Studio'")
	}

	// 2. Test GET /api/v1/frameworks
	respFw, err := http.Get(baseURL + "/api/v1/frameworks")
	if err != nil {
		t.Fatalf("GET /api/v1/frameworks failed: %v", err)
	}
	defer respFw.Body.Close()
	if respFw.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", respFw.StatusCode)
	}
	var fws []frameworks.Info
	if err := json.NewDecoder(respFw.Body).Decode(&fws); err != nil {
		t.Fatalf("decode frameworks json: %v", err)
	}
	if len(fws) == 0 {
		t.Errorf("expected at least 1 framework")
	}

	// 3. Test GET /api/v1/frameworks/cccs-itsp-10-033/controls
	respCtrl, err := http.Get(baseURL + "/api/v1/frameworks/cccs-itsp-10-033/controls?q=account")
	if err != nil {
		t.Fatalf("GET controls failed: %v", err)
	}
	defer respCtrl.Body.Close()
	if respCtrl.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", respCtrl.StatusCode)
	}
	var ctrls []frameworks.ControlInfo
	if err := json.NewDecoder(respCtrl.Body).Decode(&ctrls); err != nil {
		t.Fatalf("decode controls json: %v", err)
	}
	if len(ctrls) == 0 {
		t.Errorf("expected at least 1 matching control for 'account'")
	}

	// 4. Test GET /api/v1/compare
	respComp, err := http.Get(baseURL + "/api/v1/compare?from=cccs-itsg-33&to=cccs-itsp-10-033")
	if err != nil {
		t.Fatalf("GET compare failed: %v", err)
	}
	defer respComp.Body.Close()
	if respComp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", respComp.StatusCode)
	}

	// 5. Test POST /api/v1/validate
	valPayload := []byte(`{"component-definition":{"uuid":"11111111-2222-3333-4444-555555555555","metadata":{"title":"Test","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}`)
	respVal, err := http.Post(baseURL+"/api/v1/validate", "application/json", bytes.NewReader(valPayload))
	if err != nil {
		t.Fatalf("POST validate failed: %v", err)
	}
	defer respVal.Body.Close()
	if respVal.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", respVal.StatusCode)
	}
}
