package webstudio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

func TestWebStudioServer_Endpoints(t *testing.T) {
	srv, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     8080,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.Handler()

	// 1. Test GET / (index.html)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "xOSCAL Dev Studio") {
		t.Errorf("expected body to contain 'xOSCAL Dev Studio'")
	}

	// 2. Test GET /healthz and /readyz
	recHealth := httptest.NewRecorder()
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /healthz, got %d", recHealth.Code)
	}
	var healthData map[string]any
	if err := json.NewDecoder(recHealth.Body).Decode(&healthData); err != nil {
		t.Fatalf("decode health json: %v", err)
	}
	if healthData["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", healthData["status"])
	}

	// 3. Test GET /api/v1/frameworks
	recFw := httptest.NewRecorder()
	reqFw := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks", nil)
	handler.ServeHTTP(recFw, reqFw)
	if recFw.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recFw.Code)
	}
	var fws []frameworks.Info
	if err := json.NewDecoder(recFw.Body).Decode(&fws); err != nil {
		t.Fatalf("decode frameworks json: %v", err)
	}
	if len(fws) == 0 {
		t.Errorf("expected at least 1 framework")
	}

	// 4. Test GET /api/v1/frameworks/cccs-itsp-10-033/controls
	recCtrl := httptest.NewRecorder()
	reqCtrl := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cccs-itsp-10-033/controls?q=account", nil)
	handler.ServeHTTP(recCtrl, reqCtrl)
	if recCtrl.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recCtrl.Code)
	}
	var ctrls []frameworks.ControlInfo
	if err := json.NewDecoder(recCtrl.Body).Decode(&ctrls); err != nil {
		t.Fatalf("decode controls json: %v", err)
	}
	if len(ctrls) == 0 {
		t.Errorf("expected at least 1 matching control for 'account'")
	}

	// 5. Test GET /api/v1/compare
	recComp := httptest.NewRecorder()
	reqComp := httptest.NewRequest(http.MethodGet, "/api/v1/compare?from=cccs-itsg-33&to=cccs-itsp-10-033", nil)
	handler.ServeHTTP(recComp, reqComp)
	if recComp.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recComp.Code)
	}

	// 6. Test POST /api/v1/validate
	valPayload := []byte(`{"component-definition":{"uuid":"11111111-2222-3333-4444-555555555555","metadata":{"title":"Test","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}`)
	recVal := httptest.NewRecorder()
	reqVal := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(valPayload))
	reqVal.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recVal, reqVal)
	if recVal.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recVal.Code)
	}

	// 7. Test GET /api/v1/frameworks/cybersecure-canada/parity
	recParity := httptest.NewRecorder()
	reqParity := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cybersecure-canada/parity", nil)
	handler.ServeHTTP(recParity, reqParity)
	if recParity.Code != http.StatusOK {
		t.Errorf("expected 200 OK for parity, got %d", recParity.Code)
	}
	var rep frameworks.ParityReport
	if err := json.NewDecoder(recParity.Body).Decode(&rep); err != nil {
		t.Fatalf("decode parity report: %v", err)
	}
	if rep.Verdict != frameworks.VerdictVerbatimParity {
		t.Errorf("expected VERBATIM_PARITY from parity endpoint, got %s", rep.Verdict)
	}

	// 8. Test GET /api/v1/frameworks/cccs-itsp-10-033/layers
	recLayers := httptest.NewRecorder()
	reqLayers := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cccs-itsp-10-033/layers", nil)
	handler.ServeHTTP(recLayers, reqLayers)
	if recLayers.Code != http.StatusOK {
		t.Errorf("expected 200 OK for layers, got %d", recLayers.Code)
	}
	var layerSummaries []struct {
		Name      string `json:"name"`
		Kind      string `json:"kind"`
		SizeBytes int    `json:"size_bytes"`
		SHA256    string `json:"sha256"`
	}
	if err := json.NewDecoder(recLayers.Body).Decode(&layerSummaries); err != nil {
		t.Fatalf("decode layers json: %v", err)
	}
	if len(layerSummaries) != 7 {
		t.Errorf("expected 7 layers, got %d", len(layerSummaries))
	}

	// 9. Test GET /api/v1/frameworks/cccs-itsp-10-033/layers/ssp?download=1
	recSSP := httptest.NewRecorder()
	reqSSP := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cccs-itsp-10-033/layers/ssp?download=1", nil)
	handler.ServeHTTP(recSSP, reqSSP)
	if recSSP.Code != http.StatusOK {
		t.Errorf("expected 200 OK for ssp layer, got %d", recSSP.Code)
	}
	if !strings.Contains(recSSP.Body.String(), "system-security-plan") {
		t.Errorf("expected ssp layer to contain 'system-security-plan'")
	}
	if !strings.Contains(recSSP.Header().Get("Content-Disposition"), "attachment;") {
		t.Errorf("expected attachment Content-Disposition on download=1")
	}

	// 10. Test GET /api/v1/frameworks/cccs-itsp-10-033/layers/checksums.sha256
	recChecksums := httptest.NewRecorder()
	reqChecksums := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cccs-itsp-10-033/layers/checksums.sha256?download=1", nil)
	handler.ServeHTTP(recChecksums, reqChecksums)
	if recChecksums.Code != http.StatusOK {
		t.Errorf("expected 200 OK for checksums, got %d", recChecksums.Code)
	}
	if !strings.Contains(recChecksums.Body.String(), "catalog.json") {
		t.Errorf("expected checksums to list catalog.json")
	}
}

func TestWebStudioServer_PortFallback(t *testing.T) {
	// First server binds to 19876
	srv1, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     19876,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("failed to create server 1: %v", err)
	}
	addr1, err := srv1.Start()
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "permission denied") {
			t.Skip("skipping port binding test in loopback socket restricted sandbox")
		}
		t.Fatalf("failed to start server 1: %v", err)
	}
	defer func() { _ = srv1.Stop(context.Background()) }()

	// Second server also asks for 19876, should fallback to 19877
	srv2, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     19876,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("failed to create server 2: %v", err)
	}
	addr2, err := srv2.Start()
	if err != nil {
		t.Fatalf("failed to start server 2 with fallback: %v", err)
	}
	defer func() { _ = srv2.Stop(context.Background()) }()

	if addr1 == addr2 {
		t.Errorf("expected server 2 to bind to fallback port, but got same addr: %s", addr2)
	}
}
