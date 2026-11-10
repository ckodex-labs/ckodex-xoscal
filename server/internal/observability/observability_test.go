package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrometheusMetricsRecordAndScrape(t *testing.T) {
	m := NewPrometheusMetrics()
	m.RecordRPC("/oscal.services.v1.OscalService/GetCatalog", "OK", 15*time.Millisecond)
	m.RecordRPC("/oscal.services.v1.OscalService/GetCatalog", "NotFound", 4*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `xoscal_rpc_total{code="OK",method="/oscal.services.v1.OscalService/GetCatalog"} 1`) {
		t.Fatalf("missing expected OK counter in metrics output:\n%s", body)
	}
	if !strings.Contains(body, `xoscal_rpc_duration_seconds_count{code="NotFound",method="/oscal.services.v1.OscalService/GetCatalog"} 1`) {
		t.Fatalf("missing expected histogram count in metrics output:\n%s", body)
	}
}

func TestInitTracingAndTracer(t *testing.T) {
	shutdown, err := InitTracing("xoscal-unit-test", 0.0)
	if err != nil {
		t.Fatalf("InitTracing failed: %v", err)
	}
	tr := Tracer("xoscal-component")
	if tr == nil {
		t.Fatal("Tracer returned nil")
	}
	ctx, span := tr.Start(context.Background(), "test-span")
	span.End()
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}
