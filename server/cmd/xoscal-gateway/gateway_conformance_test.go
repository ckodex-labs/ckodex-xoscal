package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/openapiembed"
	"github.com/mchorfa/xoscal/server/internal/utcp"
)

func startBufconnHealthServer(t *testing.T) *healthProber {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	grpcSrv := grpc.NewServer()
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, hs)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcSrv.Stop()
		_ = lis.Close()
	})

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
	}
	prober := newHealthProber("passthrough:///bufnet", dialOpts)
	t.Cleanup(func() { _ = prober.Close() })
	return prober
}

func TestGatewayProbeAndDiscoveryTokenExemption(t *testing.T) {
	prober := startBufconnHealthServer(t)
	metrics := newMetricsCollector()
	root := http.NewServeMux()
	registerUTCPHandlersWithAuth(root, "127.0.0.1:50051", true)
	registerGatewayObservability(root, prober, metrics)

	handler := buildGatewayHandler(root, metrics, "gateway-secret-token", parseAllowedOrigins("http://127.0.0.1:8765"), 128)

	// 1. Unauthenticated probes and UTCP discovery must succeed (200 OK)
	for _, path := range []string{"/healthz", "/readyz", "/.well-known/utcp"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "127.0.0.1:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unauthenticated GET %s: expected 200 OK, got %d (body=%s)", path, rec.Code, rec.Body.String())
		}
	}

	// Verify strict 3-key UTCP wire manual + ApiKeyAuth on protected HTTP tools
	utcpReq := httptest.NewRequest(http.MethodGet, "/.well-known/utcp", nil)
	utcpReq.Host = "127.0.0.1:8080"
	utcpRec := httptest.NewRecorder()
	handler.ServeHTTP(utcpRec, utcpReq)

	var rawManual map[string]any
	if err := json.Unmarshal(utcpRec.Body.Bytes(), &rawManual); err != nil {
		t.Fatalf("unmarshal /.well-known/utcp: %v", err)
	}
	if len(rawManual) != 3 {
		t.Fatalf("expected strict 3-key UTCP wire manual on /.well-known/utcp, got keys: %v", rawManual)
	}
	if !strings.Contains(utcpRec.Body.String(), `"auth_type":"api_key"`) ||
		!strings.Contains(utcpRec.Body.String(), `"Bearer ${XOSCAL_API_TOKEN}"`) {
		t.Fatalf("expected ApiKeyAuth with Bearer ${XOSCAL_API_TOKEN} in /.well-known/utcp, got:\n%s", utcpRec.Body.String())
	}

	// Verify ?include_cli=true&include_grpc=true&profile=extended query parameters
	extReq := httptest.NewRequest(http.MethodGet, "/.well-known/utcp?include_cli=true&include_grpc=true&profile=extended", nil)
	extRec := httptest.NewRecorder()
	handler.ServeHTTP(extRec, extReq)
	if extRec.Code != http.StatusOK || !strings.Contains(extRec.Body.String(), `"call_template_type":"cli"`) || !strings.Contains(extRec.Body.String(), `"call_template_type":"grpc"`) || !strings.Contains(extRec.Body.String(), `"metadata"`) {
		t.Fatalf("expected extended manual with cli+grpc tools, got status=%d body=%s", extRec.Code, extRec.Body.String())
	}

	// 2. Protected /api/v1/* routes must reject unauthenticated requests (401) and allow valid Bearer token (200)
	unauthFwReq := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks?jurisdiction=ca", nil)
	unauthFwRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthFwRec, unauthFwReq)
	if unauthFwRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET /api/v1/frameworks: expected 401, got %d", unauthFwRec.Code)
	}

	authFwReq := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks?jurisdiction=ca", nil)
	authFwReq.Header.Set("Authorization", "Bearer gateway-secret-token")
	authFwRec := httptest.NewRecorder()
	handler.ServeHTTP(authFwRec, authFwReq)
	if authFwRec.Code != http.StatusOK || !strings.Contains(authFwRec.Body.String(), `"cccs-itsp-10-033"`) {
		t.Fatalf("authenticated GET /api/v1/frameworks?jurisdiction=ca failed: status=%d body=%s", authFwRec.Code, authFwRec.Body.String())
	}
}

func TestGatewayEmbeddedOpenAPIFromTempDir(t *testing.T) {
	t.Chdir(t.TempDir())

	root := http.NewServeMux()
	registerGatewayObservability(root, nil, newMetricsCollector())

	routes := []string{
		"/openapi/governance_service.openapi.yaml",
		"/openapi/oscal_service.openapi.yaml",
		"/openapi/transparency_exchange_service.openapi.yaml",
		"/openapi/transparency_graph_service.openapi.yaml",
		"/openapi/governance.yaml",
		"/openapi/oscal.yaml",
	}
	if len(openapiembed.ListSpecs()) != 4 {
		t.Fatalf("expected 4 embedded OpenAPI specs, got %v", openapiembed.ListSpecs())
	}

	for _, rPath := range routes {
		req := httptest.NewRequest(http.MethodGet, rPath, nil)
		rec := httptest.NewRecorder()
		root.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s from isolated temp dir: expected 200 OK, got %d", rPath, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "openapi:") {
			t.Errorf("GET %s missing 'openapi:' header in body", rPath)
		}
	}

	postReq := httptest.NewRequest(http.MethodPost, "/openapi/governance_service.openapi.yaml", nil)
	postRec := httptest.NewRecorder()
	root.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /openapi/...: expected 405, got %d", postRec.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/openapi/does_not_exist.openapi.yaml", nil)
	missingRec := httptest.NewRecorder()
	root.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Errorf("GET /openapi/does_not_exist.openapi.yaml: expected 404, got %d", missingRec.Code)
	}
}

func TestGatewayUTCPHTTPToolsEndToEnd(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XOSCAL_FEED_CACHE_DIR", cacheDir)

	prober := startBufconnHealthServer(t)
	metrics := newMetricsCollector()
	root := http.NewServeMux()
	registerUTCPHandlers(root, "127.0.0.1:50051")
	registerGatewayObservability(root, prober, metrics)

	handler := buildGatewayHandler(root, metrics, "", parseAllowedOrigins("http://127.0.0.1:8765"), 128)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Discover manual from live /.well-known/utcp
	resp, err := ts.Client().Get(ts.URL + "/.well-known/utcp")
	if err != nil {
		t.Fatalf("GET /.well-known/utcp: %v", err)
	}
	manualBytes, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read manual body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /.well-known/utcp status = %d", resp.StatusCode)
	}

	var manual utcp.Manual
	if err := json.Unmarshal(manualBytes, &manual); err != nil {
		t.Fatalf("unmarshal manual: %v", err)
	}
	if len(manual.Tools) != 18 {
		t.Fatalf("expected 18 HTTP tools in discovered manual, got %d", len(manual.Tools))
	}

	// Execute every discovered UTCP HTTP tool against ts.URL
	for _, tool := range manual.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			rawURL := tool.ToolCallTemplate.URL
			rawURL = strings.ReplaceAll(rawURL, "{framework}", "cccs-itsp-10-033")
			if tool.Name == "xoscal_frameworks_parity" || tool.Name == "xoscal_frameworks_layers" || tool.Name == "xoscal_frameworks_get" {
				rawURL = strings.ReplaceAll(rawURL, "{id}", "cybersecure-canada")
			} else {
				rawURL = strings.ReplaceAll(rawURL, "{id}", "ac-2")
			}

			u, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("parse tool URL %q: %v", rawURL, err)
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
				bodyReader = strings.NewReader(`{"offline":true,"feed_ids":["cisa-kev-active-threats"]}`)
			case "xoscal_schema_validate":
				doc := map[string]any{
					"kind": "component-definition",
					"document": map[string]any{
						"component-definition": map[string]any{
							"uuid": testUUID,
							"metadata": map[string]any{
								"title":         "UTCP E2E Test",
								"last-modified": "2026-10-23T00:00:00Z",
								"version":       "1.0",
								"oscal-version": "1.2.3",
							},
						},
					},
				}
				b, _ := json.Marshal(doc)
				bodyReader = bytes.NewReader(b)
			}
			u.RawQuery = q.Encode()

			req, err := http.NewRequest(tool.ToolCallTemplate.HTTPMethod, u.String(), bodyReader)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if tool.ToolCallTemplate.ContentType != "" {
				req.Header.Set("Content-Type", tool.ToolCallTemplate.ContentType)
			}
			for hk, hv := range tool.ToolCallTemplate.Headers {
				req.Header.Set(hk, hv)
			}

			res, err := ts.Client().Do(req)
			if err != nil {
				t.Fatalf("HTTP %s %s: %v", req.Method, u.String(), err)
			}
			resBody, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("tool %s (%s %s): status = %d, want 200; body=%s", tool.Name, req.Method, u.String(), res.StatusCode, string(resBody))
			}
			var decoded any
			if err := json.Unmarshal(resBody, &decoded); err != nil {
				t.Fatalf("tool %s response is not valid JSON: %v\nbody=%s", tool.Name, err, string(resBody))
			}
		})
	}

	// Verify feed sync persisted verifiable checkpoint in XOSCAL_FEED_CACHE_DIR
	if _, entries, err := feeds.VerifyCheckpointDir(cacheDir); err != nil || entries == 0 {
		t.Fatalf("expected verified feed sync checkpoint in %s, got entries=%d err=%v", cacheDir, entries, err)
	}

	// Verify GET /api/v1/utcp/ecosystem and method enforcement (405) on read-only & sync endpoints
	ecoRes, err := ts.Client().Get(ts.URL + "/api/v1/utcp/ecosystem")
	if err != nil {
		t.Fatalf("GET /api/v1/utcp/ecosystem: %v", err)
	}
	_ = ecoRes.Body.Close()
	if ecoRes.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/utcp/ecosystem status = %d", ecoRes.StatusCode)
	}

	readOnlyRoutes := []string{
		"/api/v1/doctor",
		"/api/v1/frameworks",
		"/api/v1/frameworks/cybersecure-canada",
		"/api/v1/frameworks/cybersecure-canada/layers",
		"/api/v1/frameworks/cybersecure-canada/parity",
		"/api/v1/controls",
		"/api/v1/controls/search?q=ac",
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
	}
	for _, p := range readOnlyRoutes {
		postRes, pErr := ts.Client().Post(ts.URL+p, "application/json", strings.NewReader(`{}`))
		if pErr != nil {
			t.Fatalf("POST %s: %v", p, pErr)
		}
		_ = postRes.Body.Close()
		if postRes.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: expected 405 Method Not Allowed, got %d", p, postRes.StatusCode)
		}
	}
}
