package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	mappingv1 "github.com/mchorfa/xoscal/proto/oscal/mapping/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testUUID = "123e4567-e89b-42d3-a456-426614174000"

// TestOSCALMarshalerCatalog verifies the gateway marshaler produces
// schema-valid JSON for a Catalog message.
func TestOSCALMarshalerCatalog(t *testing.T) {
	m := newOSCALMarshaler()
	cat := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Catalog",
			Version: "1.0",
		},
		Controls: []*catalogv1.Control{{
			Id:    &commonv1.Token{Value: "ctrl-1"},
			Title: &commonv1.MarkupLine{Value: "Control 1"},
		}},
	}
	b, err := m.Marshal(cat)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindCatalog); err != nil {
		t.Errorf("catalog failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerProfile verifies the gateway marshaler produces
// schema-valid JSON for a Profile message.
func TestOSCALMarshalerProfile(t *testing.T) {
	m := newOSCALMarshaler()
	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Profile",
			Version: "1.0",
		},
		Imports: []*profilev1.Import{{
			Href:       &commonv1.URIReference{Value: "https://example.gov/catalog.json"},
			IncludeAll: &profilev1.IncludeAll{},
		}},
	}
	b, err := m.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindProfile); err != nil {
		t.Errorf("profile failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerComponentDefinition verifies the gateway marshaler
// produces schema-valid JSON for a ComponentDefinition message.
func TestOSCALMarshalerComponentDefinition(t *testing.T) {
	m := newOSCALMarshaler()
	c := &componentv1.ComponentDefinition{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Component Definition",
			Version: "1.0",
		},
	}
	b, err := m.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindComponentDefinition); err != nil {
		t.Errorf("component-definition failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerAssessmentPlan verifies the gateway marshaler
// produces schema-valid JSON for an AssessmentPlan message.
func TestOSCALMarshalerAssessmentPlan(t *testing.T) {
	m := newOSCALMarshaler()
	ap := &assessment_planv1.AssessmentPlan{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test AP",
			Version: "1.0",
		},
		ImportSsp: &assessment_planv1.ImportSsp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ssp.json"},
		},
		ReviewedControls: &assessment_planv1.ReviewedControls{
			ControlSelections: []*assessment_planv1.ControlSelection{{
				Description: &commonv1.MarkupMultiline{Value: "All controls"},
				IncludeAll:  &assessment_planv1.IncludeAll{},
			}},
		},
	}
	b, err := m.Marshal(ap)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindAssessmentPlan); err != nil {
		t.Errorf("assessment-plan failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerAssessmentResults verifies the gateway marshaler
// produces schema-valid JSON for an AssessmentResults message.
func TestOSCALMarshalerAssessmentResults(t *testing.T) {
	m := newOSCALMarshaler()
	ar := &assessment_resultsv1.AssessmentResults{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test AR",
			Version: "1.0",
		},
		ImportAp: &assessment_resultsv1.ImportAp{
			Href: &commonv1.URIReference{Value: "https://example.gov/ap.json"},
		},
		Results: []*assessment_resultsv1.Result{{
			Uuid:        &commonv1.UUID{Value: testUUID},
			Title:       &commonv1.MarkupLine{Value: "Result 1"},
			Description: &commonv1.MarkupMultiline{Value: "Test result"},
			Start:       &commonv1.DateTime{Value: timestamppb.Now()},
			ReviewedControls: &assessment_resultsv1.ReviewedControls{
				ControlSelections: []*assessment_resultsv1.ControlSelection{{
					Description: &commonv1.MarkupMultiline{Value: "All controls"},
					IncludeAll:  &assessment_resultsv1.IncludeAll{},
				}},
			},
		}},
	}
	b, err := m.Marshal(ar)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindAssessmentResults); err != nil {
		t.Errorf("assessment-results failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerPOAM verifies the gateway marshaler
// produces schema-valid JSON for a POAM message.
func TestOSCALMarshalerPOAM(t *testing.T) {
	m := newOSCALMarshaler()
	poam := &poamv1.PlanOfActionAndMilestones{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test POAM",
			Version: "1.0",
		},
		PoamItems: []*poamv1.PoamItem{{
			Uuid:        &commonv1.UUID{Value: testUUID},
			Title:       &commonv1.MarkupLine{Value: "POAM Item 1"},
			Description: &commonv1.MarkupMultiline{Value: "A POAM item"},
		}},
	}
	b, err := m.Marshal(poam)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindPOAM); err != nil {
		t.Errorf("poam failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerMapping verifies the gateway marshaler produces
// schema-valid JSON for the OSCAL 1.2 Mapping model.
func TestOSCALMarshalerMapping(t *testing.T) {
	m := newOSCALMarshaler()
	mapping := &mappingv1.MappingCollection{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test Mapping",
			Version: "1.0",
		},
		Provenance: &mappingv1.MappingProvenance{
			Method:             "human",
			MatchingRationale:  "functional",
			Status:             "complete",
			MappingDescription: &commonv1.MarkupMultiline{Value: "Gateway mapping."},
		},
		Mappings: []*mappingv1.ControlMapping{{
			Uuid: &commonv1.UUID{Value: testUUID},
			SourceResource: &mappingv1.MappingResourceReference{
				Type: "catalog",
				Href: &commonv1.URIReference{Value: "https://example.gov/source.json"},
			},
			TargetResource: &mappingv1.MappingResourceReference{
				Type: "catalog",
				Href: &commonv1.URIReference{Value: "https://example.gov/target.json"},
			},
			Maps: []*mappingv1.Map{{
				Uuid:         &commonv1.UUID{Value: testUUID},
				Relationship: &commonv1.Token{Value: "subset-of"},
				Sources:      []*mappingv1.MappingItem{{Type: "control", IdRef: "ac-1"}},
				Targets:      []*mappingv1.MappingItem{{Type: "control", IdRef: "cc6.1"}},
			}},
		}},
	}
	b, err := m.Marshal(mapping)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindMapping); err != nil {
		t.Errorf("mapping-collection failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerSSP verifies the gateway marshaler
// produces schema-valid JSON for an SSP message.
func TestOSCALMarshalerSSP(t *testing.T) {
	m := newOSCALMarshaler()
	ssp := &sspv1.SystemSecurityPlan{
		Uuid: &commonv1.UUID{Value: testUUID},
		Metadata: &commonv1.Metadata{
			Title:   "Gateway Test SSP",
			Version: "1.0",
		},
		ImportProfile: &sspv1.ImportProfile{
			Href: &commonv1.URIReference{Value: "https://example.gov/profile.json"},
		},
		SystemCharacteristics: &sspv1.SystemCharacteristics{
			SystemIds:   []*sspv1.SystemId{{Id: "test-system"}},
			SystemName:  "Test System",
			Description: &commonv1.MarkupMultiline{Value: "A test system."},
			SystemInformation: &sspv1.SystemInformation{
				InformationTypes: []*sspv1.InformationType{{
					Uuid:        &commonv1.UUID{Value: testUUID},
					Title:       &commonv1.MarkupLine{Value: "Test info type"},
					Description: &commonv1.MarkupMultiline{Value: "Test information type"},
				}},
			},
			Status:                &sspv1.Status{State: "operational"},
			AuthorizationBoundary: &sspv1.AuthorizationBoundary{Description: &commonv1.MarkupMultiline{Value: "Boundary"}},
		},
		SystemImplementation: &sspv1.SystemImplementation{
			Users: []*sspv1.SystemUser{{
				Uuid:  &commonv1.UUID{Value: testUUID},
				Title: "Admin",
			}},
			Components: []*sspv1.SystemComponent{{
				Uuid:        &commonv1.UUID{Value: testUUID},
				Type:        "software",
				Title:       "App",
				Description: "App component",
				Status:      &sspv1.Status{State: "operational"},
			}},
		},
		ControlImplementation: &sspv1.ControlImplementation{
			Description: &commonv1.MarkupMultiline{Value: "Implementation"},
			ImplementedRequirements: []*sspv1.ImplementedRequirement{{
				Uuid:      &commonv1.UUID{Value: testUUID},
				ControlId: &commonv1.Token{Value: "ac-1"},
			}},
		},
	}
	b, err := m.Marshal(ssp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	v := mustValidator(t)
	if err := v.Validate(b, schemavalidate.KindSSP); err != nil {
		t.Errorf("ssp failed schema validation: %v", err)
	}
}

// TestOSCALMarshalerNonOSCALFallback verifies the marshaler delegates
// non-OSCAL types to the default protojson marshaler.
func TestOSCALMarshalerNonOSCALFallback(t *testing.T) {
	m := newOSCALMarshaler()
	// A plain map is not a proto.Message, so it should fall through to
	// the embedded JSONPb which handles non-proto values via json.Marshal.
	b, err := m.Marshal(map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("Marshal non-OSCAL: %v", err)
	}
	if string(b) == "" {
		t.Error("expected non-empty JSON output for non-OSCAL type")
	}
}

// TestOSCALMarshalerName verifies the marshaler Name method.
func TestOSCALMarshalerName(t *testing.T) {
	m := newOSCALMarshaler()
	if m.Name() != "oscal-json" {
		t.Errorf("expected name 'oscal-json', got %q", m.Name())
	}
}

// TestOSCALMarshalerContentType verifies the marshaler ContentType method.
func TestOSCALMarshalerContentType(t *testing.T) {
	m := newOSCALMarshaler()
	if ct := m.ContentType(nil); ct != "application/json" {
		t.Errorf("expected 'application/json', got %q", ct)
	}
}

func TestGatewayUTCPEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	registerUTCPHandlers(mux, "127.0.0.1:50051")

	for _, path := range []string{"/.well-known/utcp", "/api/v1/utcp"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "127.0.0.1:8080"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200 OK, got %d", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"utcp_version":"1.0.1"`) {
			t.Errorf("GET %s: expected utcp_version 1.0.1 in body: %s", path, rec.Body.String())
		}

		// Validate the returned manual via POST /api/v1/utcp/validate
		valReq := httptest.NewRequest(http.MethodPost, "/api/v1/utcp/validate", bytes.NewReader(rec.Body.Bytes()))
		valRec := httptest.NewRecorder()
		mux.ServeHTTP(valRec, valReq)
		if valRec.Code != http.StatusOK {
			t.Fatalf("POST /api/v1/utcp/validate: expected 200 OK, got %d", valRec.Code)
		}
		if !strings.Contains(valRec.Body.String(), `"verdict":"CONFORMANT"`) {
			t.Errorf("POST /api/v1/utcp/validate: expected CONFORMANT verdict, got %s", valRec.Body.String())
		}
	}

	// Verify 405 Method Not Allowed on POST /.well-known/utcp and GET /api/v1/utcp/validate
	postManualReq := httptest.NewRequest(http.MethodPost, "/.well-known/utcp", nil)
	postManualRec := httptest.NewRecorder()
	mux.ServeHTTP(postManualRec, postManualReq)
	if postManualRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /.well-known/utcp: expected 405, got %d", postManualRec.Code)
	}

	getValReq := httptest.NewRequest(http.MethodGet, "/api/v1/utcp/validate", nil)
	getValRec := httptest.NewRecorder()
	mux.ServeHTTP(getValRec, getValReq)
	if getValRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/v1/utcp/validate: expected 405, got %d", getValRec.Code)
	}

	// Verify 400 Bad Request on malformed JSON payload
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/utcp/validate", strings.NewReader("{bad-json"))
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/v1/utcp/validate malformed JSON: expected 400, got %d", badRec.Code)
	}

	// Verify token authentication wrapper on UTCP endpoints
	authedHandler := withTokenAuth(mux, "secret-token-123")
	unauthReq := httptest.NewRequest(http.MethodGet, "/.well-known/utcp", nil)
	unauthRec := httptest.NewRecorder()
	authedHandler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated GET /.well-known/utcp: expected 401, got %d", unauthRec.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/.well-known/utcp", nil)
	authReq.Header.Set("Authorization", "Bearer secret-token-123")
	authRec := httptest.NewRecorder()
	authedHandler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Errorf("authenticated GET /.well-known/utcp: expected 200, got %d", authRec.Code)
	}
}

// mustValidator creates a Validator or fails the test.
func mustValidator(t *testing.T) *schemavalidate.Validator {
	t.Helper()
	v, err := schemavalidate.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return v
}

func TestGatewayMetricsHistogramAndCardinalityCap(t *testing.T) {
	mc := newMetricsCollector()

	// Bombard collector with 500 distinct random 404/scan URLs + UUID routes
	for i := 0; i < 500; i++ {
		mc.record(fmt.Sprintf("GET /scan-probe-%d/secret", i), http.StatusNotFound, 12*time.Millisecond)
	}
	mc.record("GET /v1/catalogs/123e4567-e89b-42d3-a456-426614174000", http.StatusOK, 40*time.Millisecond)

	if got := mc.trackedRouteCount(); got > maxTrackedRoutes {
		t.Fatalf("metricsCollector cardinality exceeded maxTrackedRoutes (%d > %d)", got, maxTrackedRoutes)
	}

	prom := mc.prometheusText()
	if !strings.Contains(prom, "# TYPE http_request_duration_seconds histogram") {
		t.Errorf("expected Prometheus histogram type header in:\n%s", prom)
	}
	if !strings.Contains(prom, `http_request_duration_seconds_bucket{`) || !strings.Contains(prom, `le="+Inf"`) {
		t.Errorf("expected Prometheus histogram buckets (+Inf) in:\n%s", prom)
	}
	if !strings.Contains(prom, `path="OTHER"`) {
		t.Errorf("expected overflow routes bucketed into path=\"OTHER\"")
	}
}

func TestGatewayRouteParityAndObservability(t *testing.T) {
	root := http.NewServeMux()
	metrics := newMetricsCollector()
	dialOpts, err := grpcDialOptions("", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	prober := newHealthProber("127.0.0.1:1", dialOpts)
	defer func() { _ = prober.Close() }()

	registerUTCPHandlers(root, "127.0.0.1:50051")
	registerGatewayObservability(root, prober, metrics)

	wrapped := buildGatewayHandler(root, metrics, "", parseAllowedOrigins("http://127.0.0.1:8765"), 128)

	// 1. /api/v1/graph/export parity
	expReq := httptest.NewRequest(http.MethodGet, "/api/v1/graph/export?target=CVE-2024-3400&format=cypher&depth=1", nil)
	expRec := httptest.NewRecorder()
	wrapped.ServeHTTP(expRec, expReq)
	if expRec.Code != http.StatusOK || !strings.Contains(expRec.Body.String(), "MERGE") {
		t.Fatalf("GET /api/v1/graph/export failed: status=%d body=%s", expRec.Code, expRec.Body.String())
	}

	// 2. /api/v1/feeds/sync parity (POST & GET)
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		syncReq := httptest.NewRequest(method, "/api/v1/feeds/sync?feed=cisa-kev-active-threats", nil)
		syncRec := httptest.NewRecorder()
		wrapped.ServeHTTP(syncRec, syncReq)
		if syncRec.Code != http.StatusOK || !strings.Contains(syncRec.Body.String(), `"bundle_sha256"`) {
			t.Fatalf("%s /api/v1/feeds/sync failed: status=%d body=%s", method, syncRec.Code, syncRec.Body.String())
		}
	}

	// 3. /api/v1/feeds?feed=cisa-kev-active-threats inspection parity
	feedInspReq := httptest.NewRequest(http.MethodGet, "/api/v1/feeds?feed=cisa-kev-active-threats", nil)
	feedInspRec := httptest.NewRecorder()
	wrapped.ServeHTTP(feedInspRec, feedInspReq)
	if feedInspRec.Code != http.StatusOK || !strings.Contains(feedInspRec.Body.String(), `"kev_advisories"`) {
		t.Fatalf("GET /api/v1/feeds?feed=... failed: status=%d", feedInspRec.Code)
	}

	// 3b. Full Web Studio + UTCP HTTP tool parity (/api/v1/frameworks, /api/v1/compare, /api/v1/validate, parity traversal 400)
	fwReq := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks", nil)
	fwRec := httptest.NewRecorder()
	wrapped.ServeHTTP(fwRec, fwReq)
	if fwRec.Code != http.StatusOK || !strings.Contains(fwRec.Body.String(), `"cccs-itsp-10-033"`) {
		t.Fatalf("GET /api/v1/frameworks failed: status=%d body=%s", fwRec.Code, fwRec.Body.String())
	}

	cmpReq := httptest.NewRequest(http.MethodGet, "/api/v1/compare?from=cccs-itsg-33&to=cccs-itsp-10-033", nil)
	cmpRec := httptest.NewRecorder()
	wrapped.ServeHTTP(cmpRec, cmpReq)
	if cmpRec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/compare failed: status=%d body=%s", cmpRec.Code, cmpRec.Body.String())
	}

	valPayload := []byte(`{"component-definition":{"uuid":"` + testUUID + `","metadata":{"title":"Test","last-modified":"2026-10-23T00:00:00Z","version":"1.0","oscal-version":"1.2.3"}}}`)
	valReq := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(valPayload))
	valRec := httptest.NewRecorder()
	wrapped.ServeHTTP(valRec, valReq)
	if valRec.Code != http.StatusOK || !strings.Contains(valRec.Body.String(), `"valid":true`) {
		t.Fatalf("POST /api/v1/validate failed: status=%d body=%s", valRec.Code, valRec.Body.String())
	}

	travReq := httptest.NewRequest(http.MethodGet, "/api/v1/frameworks/cybersecure-canada/parity?layer=../../etc/passwd", nil)
	travRec := httptest.NewRecorder()
	wrapped.ServeHTTP(travRec, travReq)
	if travRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on Gateway parity traversal, got %d", travRec.Code)
	}

	// 4. /metrics & /readyz on root mux (without touching http.DefaultServeMux)
	readyReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyRec := httptest.NewRecorder()
	wrapped.ServeHTTP(readyRec, readyReq)
	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when upstream gRPC is down on /readyz, got %d", readyRec.Code)
	}

	mReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mRec := httptest.NewRecorder()
	wrapped.ServeHTTP(mRec, mReq)
	if mRec.Code != http.StatusOK || !strings.Contains(mRec.Body.String(), `path="/api/v1/graph/export"`) {
		t.Fatalf("GET /metrics missing expected route metrics: status=%d body=%s", mRec.Code, mRec.Body.String())
	}
}

func TestGatewayMiddlewareMetricsCapture(t *testing.T) {
	blockCh := make(chan struct{})
	enteredCh := make(chan struct{}, 2)

	root := http.NewServeMux()
	metrics := newMetricsCollector()
	registerGatewayObservability(root, nil, metrics)
	root.HandleFunc("/boom", func(_ http.ResponseWriter, _ *http.Request) {
		panic("simulated handler crash")
	})
	root.HandleFunc("/hold", func(w http.ResponseWriter, _ *http.Request) {
		enteredCh <- struct{}{}
		<-blockCh
		w.WriteHeader(http.StatusOK)
	})

	handler := buildGatewayHandler(root, metrics, "tok-xyz", parseAllowedOrigins("http://127.0.0.1:8765"), 1)

	// 1. 401 unauthenticated request must be recorded in metrics
	unauthReq := httptest.NewRequest(http.MethodGet, "/boom", nil)
	unauthRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthRec.Code)
	}

	// 2. 500 recovered panic must be recorded in metrics
	panicReq := httptest.NewRequest(http.MethodGet, "/boom", nil)
	panicReq.Header.Set("Authorization", "Bearer tok-xyz")
	panicRec := httptest.NewRecorder()
	handler.ServeHTTP(panicRec, panicReq)
	if panicRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", panicRec.Code)
	}

	// 3. 503 concurrency backpressure must be recorded in metrics
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		holdReq := httptest.NewRequest(http.MethodGet, "/hold", nil)
		holdReq.Header.Set("Authorization", "Bearer tok-xyz")
		handler.ServeHTTP(httptest.NewRecorder(), holdReq)
	}()
	<-enteredCh

	satReq := httptest.NewRequest(http.MethodGet, "/hold", nil)
	satReq.Header.Set("Authorization", "Bearer tok-xyz")
	satRec := httptest.NewRecorder()
	handler.ServeHTTP(satRec, satReq)
	if satRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when saturated, got %d", satRec.Code)
	}
	close(blockCh)
	wg.Wait()

	prom := metrics.prometheusText()
	for _, want := range []string{`status="401"`, `status="500"`, `status="503"`} {
		if !strings.Contains(prom, want) {
			t.Errorf("expected Prometheus metrics to record %s, got:\n%s", want, prom)
		}
	}
}
