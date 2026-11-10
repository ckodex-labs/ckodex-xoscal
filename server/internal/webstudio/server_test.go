package webstudio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "xOSCAL Dev Studio") {
		t.Errorf("expected body to contain 'xOSCAL Dev Studio'")
	}
	for _, cssOrText := range []string{".badge-pass", ".badge-fail", "16,237"} {
		if !strings.Contains(bodyStr, cssOrText) {
			t.Errorf("expected index.html to contain %q", cssOrText)
		}
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

	// 11. Test GET /.well-known/utcp and POST /api/v1/utcp/validate
	recUTCP := httptest.NewRecorder()
	reqUTCP := httptest.NewRequest(http.MethodGet, "/.well-known/utcp", nil)
	handler.ServeHTTP(recUTCP, reqUTCP)
	if recUTCP.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /.well-known/utcp, got %d", recUTCP.Code)
	}
	if !strings.Contains(recUTCP.Body.String(), `"utcp_version":"1.0.1"`) {
		t.Errorf("expected /.well-known/utcp to return utcp_version 1.0.1, got %s", recUTCP.Body.String())
	}

	recUTCPVal := httptest.NewRecorder()
	reqUTCPVal := httptest.NewRequest(http.MethodPost, "/api/v1/utcp/validate", bytes.NewReader(recUTCP.Body.Bytes()))
	reqUTCPVal.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recUTCPVal, reqUTCPVal)
	if recUTCPVal.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/utcp/validate, got %d", recUTCPVal.Code)
	}
	if !strings.Contains(recUTCPVal.Body.String(), `"CONFORMANT"`) {
		t.Errorf("expected CONFORMANT verdict from /api/v1/utcp/validate, got %s", recUTCPVal.Body.String())
	}

	// 12. Test GET /api/v1/ontology
	recOnt := httptest.NewRecorder()
	reqOnt := httptest.NewRequest(http.MethodGet, "/api/v1/ontology", nil)
	handler.ServeHTTP(recOnt, reqOnt)
	if recOnt.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/ontology, got %d", recOnt.Code)
	}
	if !strings.Contains(recOnt.Body.String(), `"https://xoscal.dev/ns/ontology/v1#"`) {
		t.Errorf("expected ontology namespace in /api/v1/ontology response")
	}

	// 13. Test GET /api/v1/graph/crosswalk?control=ac-2
	recCW := httptest.NewRecorder()
	reqCW := httptest.NewRequest(http.MethodGet, "/api/v1/graph/crosswalk?control=ac-2", nil)
	handler.ServeHTTP(recCW, reqCW)
	if recCW.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/graph/crosswalk, got %d: %s", recCW.Code, recCW.Body.String())
	}
	if !strings.Contains(recCW.Body.String(), `"canonical_control_id":"ac-2"`) {
		t.Errorf("expected canonical_control_id ac-2 in crosswalk response, got %s", recCW.Body.String())
	}

	// 14. Test GET /api/v1/graph/blast-radius?target=CVE-2024-3400
	recBR := httptest.NewRecorder()
	reqBR := httptest.NewRequest(http.MethodGet, "/api/v1/graph/blast-radius?target=CVE-2024-3400", nil)
	handler.ServeHTTP(recBR, reqBR)
	if recBR.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/graph/blast-radius, got %d: %s", recBR.Code, recBR.Body.String())
	}
	if !strings.Contains(recBR.Body.String(), `"CVE-2024-3400"`) {
		t.Errorf("expected CVE-2024-3400 in blast radius response")
	}

	// 15. Test GET /api/v1/feeds and POST /api/v1/feeds/sync
	recFeeds := httptest.NewRecorder()
	reqFeeds := httptest.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
	handler.ServeHTTP(recFeeds, reqFeeds)
	if recFeeds.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/feeds, got %d", recFeeds.Code)
	}
	if !strings.Contains(recFeeds.Body.String(), `"cisa-kev-active-threats"`) {
		t.Errorf("expected cisa-kev-active-threats in /api/v1/feeds response")
	}

	recSync := httptest.NewRecorder()
	reqSync := httptest.NewRequest(http.MethodPost, "/api/v1/feeds/sync", nil)
	handler.ServeHTTP(recSync, reqSync)
	if recSync.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/feeds/sync, got %d", recSync.Code)
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

func TestWebStudioServer_ParityPathTraversalAndSecurityHeaders(t *testing.T) {
	srv, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     8080,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	handler := srv.Handler()

	maliciousURLs := []string{
		"/api/v1/frameworks/cybersecure-canada/parity?layer=../../etc/passwd",
		"/api/v1/frameworks/cybersecure-canada/parity?layer=secret-config.json",
		"/api/v1/frameworks/cybersecure-canada/parity?dir=../../../etc",
		"/api/v1/frameworks/cybersecure-canada/parity?dir=/etc",
		"/api/v1/frameworks/cybersecure-canada/parity?dir=data/../../etc",
	}

	for _, u := range maliciousURLs {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, u, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s: expected 400 Bad Request, got %d (body=%s)", u, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("GET %s: expected X-Frame-Options=DENY, got %q", u, got)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET %s: expected X-Content-Type-Options=nosniff, got %q", u, got)
		}
		if got := rec.Header().Get("X-Request-ID"); got == "" {
			t.Errorf("GET %s: expected non-empty X-Request-ID header", u)
		}
	}

	// Verify symlink inside workspace pointing outside workspace is rejected with 400 Bad Request (os.OpenRoot(cwd) confinement)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	outsideDir := t.TempDir()
	localTmpDir, err := os.MkdirTemp(cwd, ".tmp-symlink-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(localTmpDir) })

	relBase := filepath.Base(localTmpDir)
	escapeLink := filepath.Join(localTmpDir, "escape_link")
	if err := os.Symlink(outsideDir, escapeLink); err == nil {
		symURL := "/api/v1/frameworks/cybersecure-canada/parity?dir=" + relBase + "/escape_link"
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, symURL, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s (workspace symlink escape): expected 400 Bad Request, got %d (body=%s)", symURL, rec.Code, rec.Body.String())
		}
	}
}
