package webstudio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/utcp"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

func TestWebStudioServer_Endpoints(t *testing.T) {
	t.Setenv("XOSCAL_FEED_CACHE_DIR", t.TempDir())

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

	// 3. Test GET /api/v1/frameworks & ?jurisdiction=ca
	recFw := httptest.NewRecorder()
	reqFw := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks?jurisdiction=ca", nil)
	handler.ServeHTTP(recFw, reqFw)
	if recFw.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recFw.Code)
	}
	var fws []frameworks.Info
	if err := json.NewDecoder(recFw.Body).Decode(&fws); err != nil {
		t.Fatalf("decode frameworks json: %v", err)
	}
	if len(fws) != 5 {
		t.Errorf("expected 5 Canadian frameworks for jurisdiction=ca, got %d", len(fws))
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

	// 6. Test POST /api/v1/validate (both raw OSCAL and {"kind", "document"} envelope)
	valPayload := []byte(`{"component-definition":{"uuid":"11111111-2222-4333-8444-555555555555","metadata":{"title":"Test","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}`)
	recVal := httptest.NewRecorder()
	reqVal := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(valPayload))
	reqVal.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recVal, reqVal)
	if recVal.Code != http.StatusOK || !strings.Contains(recVal.Body.String(), `"valid":true`) {
		t.Errorf("expected 200 OK valid:true, got %d: %s", recVal.Code, recVal.Body.String())
	}

	envPayload := []byte(`{"kind":"component-definition","document":{"component-definition":{"uuid":"11111111-2222-4333-8444-555555555555","metadata":{"title":"Test","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}}`)
	recEnvVal := httptest.NewRecorder()
	reqEnvVal := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(envPayload))
	reqEnvVal.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recEnvVal, reqEnvVal)
	if recEnvVal.Code != http.StatusOK || !strings.Contains(recEnvVal.Body.String(), `"valid":true`) {
		t.Errorf("expected 200 OK valid:true for kind+document envelope, got %d: %s", recEnvVal.Code, recEnvVal.Body.String())
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

	// 11. Test GET /.well-known/utcp, GET /api/v1/utcp/manual, GET /api/v1/utcp/ecosystem, POST /api/v1/utcp/validate
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

	// 13. Test GET /api/v1/graph/crosswalk?framework=cccs-itsp-10-033&control=ac-2
	recCW := httptest.NewRecorder()
	reqCW := httptest.NewRequest(http.MethodGet, "/api/v1/graph/crosswalk?framework=cccs-itsp-10-033&control=ac-2", nil)
	handler.ServeHTTP(recCW, reqCW)
	if recCW.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/graph/crosswalk, got %d: %s", recCW.Code, recCW.Body.String())
	}
	if !strings.Contains(recCW.Body.String(), `"canonical_control_id":"ac-2"`) {
		t.Errorf("expected canonical_control_id ac-2 in crosswalk response, got %s", recCW.Body.String())
	}

	// 14. Test GET /api/v1/graph/blast-radius?node=CVE-2024-3400
	recBR := httptest.NewRecorder()
	reqBR := httptest.NewRequest(http.MethodGet, "/api/v1/graph/blast-radius?node=CVE-2024-3400", nil)
	handler.ServeHTTP(recBR, reqBR)
	if recBR.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/graph/blast-radius, got %d: %s", recBR.Code, recBR.Body.String())
	}
	if !strings.Contains(recBR.Body.String(), `"CVE-2024-3400"`) {
		t.Errorf("expected CVE-2024-3400 in blast radius response")
	}

	// 15. Test GET /api/v1/feeds and POST-only /api/v1/feeds/sync
	recFeeds := httptest.NewRecorder()
	reqFeeds := httptest.NewRequest(http.MethodGet, "/api/v1/feeds", nil)
	handler.ServeHTTP(recFeeds, reqFeeds)
	if recFeeds.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/v1/feeds, got %d", recFeeds.Code)
	}
	if !strings.Contains(recFeeds.Body.String(), `"cisa-kev-active-threats"`) {
		t.Errorf("expected cisa-kev-active-threats in /api/v1/feeds response")
	}

	recGetSync := httptest.NewRecorder()
	reqGetSync := httptest.NewRequest(http.MethodGet, "/api/v1/feeds/sync", nil)
	handler.ServeHTTP(recGetSync, reqGetSync)
	if recGetSync.Code != http.StatusMethodNotAllowed || recGetSync.Header().Get("Allow") != http.MethodPost {
		t.Errorf("GET /api/v1/feeds/sync: expected 405 with Allow=POST, got %d (Allow=%q)", recGetSync.Code, recGetSync.Header().Get("Allow"))
	}

	recSync := httptest.NewRecorder()
	reqSync := httptest.NewRequest(http.MethodPost, "/api/v1/feeds/sync", nil)
	handler.ServeHTTP(recSync, reqSync)
	if recSync.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/v1/feeds/sync, got %d", recSync.Code)
	}
}

func TestWebStudioServer_UTCPHTTPToolsAndMethodEnforcementEndToEnd(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XOSCAL_FEED_CACHE_DIR", cacheDir)

	srv, err := webstudio.NewServer(webstudio.Config{
		Host:     "127.0.0.1",
		Port:     8080,
		Registry: frameworks.DefaultRegistry(),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Fetch UTCP manual from live WebStudio server
	res, err := ts.Client().Get(ts.URL + "/.well-known/utcp")
	if err != nil {
		t.Fatalf("GET /.well-known/utcp: %v", err)
	}
	manualBytes, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()

	var manual utcp.Manual
	if err := json.Unmarshal(manualBytes, &manual); err != nil {
		t.Fatalf("Unmarshal UTCP manual: %v", err)
	}
	if len(manual.Tools) != 18 {
		t.Fatalf("expected 18 HTTP tools in WebStudio UTCP manual, got %d", len(manual.Tools))
	}

	// Execute all 18 UTCP HTTP tools end-to-end against WebStudio
	for _, tool := range manual.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			rawURL := strings.ReplaceAll(tool.ToolCallTemplate.URL, "{framework}", "cccs-itsp-10-033")
			if tool.Name == "xoscal_frameworks_parity" || tool.Name == "xoscal_frameworks_layers" || tool.Name == "xoscal_frameworks_get" {
				rawURL = strings.ReplaceAll(rawURL, "{id}", "cybersecure-canada")
			} else {
				rawURL = strings.ReplaceAll(rawURL, "{id}", "ac-2")
			}

			u, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("parse URL %q: %v", rawURL, err)
			}
			var bodyReader io.Reader
			q := u.Query()
			switch tool.Name {
			case "xoscal_frameworks_list":
				q.Set("jurisdiction", "ca")
			case "xoscal_controls_list":
				q.Set("framework", "cybersecure-canada")
			case "xoscal_controls_search":
				q.Set("q", "encryption")
				q.Set("framework", "cccs-itsp-10-033")
				q.Set("limit", "5")
			case "xoscal_graph_crosswalk":
				q.Set("framework", "cccs-itsp-10-033")
				q.Set("control", "ac-2")
			case "xoscal_graph_blast_radius":
				q.Set("node", "CVE-2024-3400")
				q.Set("depth", "2")
			case "xoscal_graph_export":
				q.Set("framework", "cccs-itsp-10-033")
				q.Set("format", "jsonld")
			case "xoscal_feeds_sync":
				bodyReader = strings.NewReader(`{"offline":true,"feed_ids":["cisa-kev-active-threats","mitre-attack-800-53"]}`)
			case "xoscal_schema_validate":
				bodyReader = strings.NewReader(`{"kind":"component-definition","document":{"component-definition":{"uuid":"11111111-2222-3333-4444-555555555555","metadata":{"title":"WebStudio E2E","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}}`)
			}
			u.RawQuery = q.Encode()

			req, err := http.NewRequest(tool.ToolCallTemplate.HTTPMethod, u.String(), bodyReader)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if tool.ToolCallTemplate.ContentType != "" {
				req.Header.Set("Content-Type", tool.ToolCallTemplate.ContentType)
			}
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatalf("Do(%s %s): %v", req.Method, u.String(), err)
			}
			respBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("tool %s (%s %s): got %d, want 200; body=%s", tool.Name, req.Method, u.String(), resp.StatusCode, string(respBytes))
			}
		})
	}

	if _, entries, err := feeds.VerifyCheckpointDir(cacheDir); err != nil || entries == 0 {
		t.Fatalf("expected verified feed checkpoint in %s, got entries=%d err=%v", cacheDir, entries, err)
	}

	// Enforce GET-only (405) on read endpoints including /layers and /parity
	for _, readPath := range []string{
		"/api/v1/doctor",
		"/api/v1/frameworks",
		"/api/v1/frameworks/cybersecure-canada",
		"/api/v1/frameworks/cybersecure-canada/layers",
		"/api/v1/frameworks/cybersecure-canada/parity",
		"/api/v1/controls",
		"/api/v1/controls/search",
		"/api/v1/controls/cccs-itsp-10-033/ac-2",
		"/api/v1/ontology",
		"/api/v1/graph/stats",
		"/api/v1/graph/crosswalk",
		"/api/v1/graph/blast-radius",
		"/api/v1/graph/reconcile",
		"/api/v1/graph/export",
		"/api/v1/feeds",
		"/api/v1/utcp/manual",
		"/api/v1/utcp/ecosystem",
	} {
		postResp, pErr := ts.Client().Post(ts.URL+readPath, "application/json", strings.NewReader(`{}`))
		if pErr != nil {
			t.Fatalf("POST %s: %v", readPath, pErr)
		}
		_ = postResp.Body.Close()
		if postResp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: expected 405, got %d", readPath, postResp.StatusCode)
		}
	}
}

func TestWebStudioServer_PortFallback(t *testing.T) {
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
