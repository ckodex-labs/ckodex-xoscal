package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// metricsCollector tracks HTTP request counts and durations without external deps.
type metricsCollector struct {
	mu        sync.RWMutex
	counts    map[string]int64
	durations map[string][]time.Duration
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		counts:    make(map[string]int64),
		durations: make(map[string][]time.Duration),
	}
}

func (m *metricsCollector) record(route string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counts[route]++
	m.durations[route] = append(m.durations[route], d)
}

func (m *metricsCollector) prometheusText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var b strings.Builder
	b.WriteString("# TYPE http_request_total counter\n")
	b.WriteString("# HELP http_request_total Total HTTP requests by route and status\n")
	routes := make([]string, 0, len(m.counts))
	for r := range m.counts {
		routes = append(routes, r)
	}
	sort.Strings(routes)
	for _, r := range routes {
		parts := strings.SplitN(r, " ", 2)
		method := parts[0]
		path := r
		if len(parts) == 2 {
			path = parts[1]
		}
		fmt.Fprintf(&b, "http_request_total{method=\"%s\",path=\"%s\",status=\"200\"} %d\n", method, path, m.counts[r])
	}
	b.WriteString("# TYPE http_request_duration_seconds summary\n")
	b.WriteString("# HELP http_request_duration_seconds HTTP request latencies in seconds\n")
	for _, r := range routes {
		parts := strings.SplitN(r, " ", 2)
		method := parts[0]
		path := r
		if len(parts) == 2 {
			path = parts[1]
		}
		vals := m.durations[r]
		if len(vals) > 0 {
			var sum float64
			for _, d := range vals {
				sum += d.Seconds()
			}
			fmt.Fprintf(&b, "http_request_duration_seconds_sum{method=\"%s\",path=\"%s\"} %.4f\n", method, path, sum)
			fmt.Fprintf(&b, "http_request_duration_seconds_count{method=\"%s\",path=\"%s\"} %d\n", method, path, len(vals))
		}
	}
	return b.String()
}

func main() {
	grpcEndpoint := flag.String("grpc", "localhost:50051", "gRPC server endpoint")
	httpAddr := flag.String("http", ":8080", "HTTP gateway listen address")
	corsOrigin := flag.String("cors-origin", "", "CORS allowed origin (default: same-origin only)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, newOSCALMarshaler()),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := servicesv1.RegisterGovernanceServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, opts); err != nil {
		log.Fatalf("register governance handler: %v", err)
	}
	if err := servicesv1.RegisterOscalServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, opts); err != nil {
		log.Fatalf("register oscal handler: %v", err)
	}
	if err := servicesv1.RegisterTransparencyExchangeServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, opts); err != nil {
		log.Fatalf("register transparency exchange handler: %v", err)
	}
	if err := servicesv1.RegisterTransparencyGraphServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, opts); err != nil {
		log.Fatalf("register transparency graph handler: %v", err)
	}

	// --- Observability ---
	metrics := newMetricsCollector()

	// Health check: probe gRPC server.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		conn, err := grpc.Dial(*grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			http.Error(w, `{"status":"unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		defer conn.Close()
		client := healthpb.NewHealthClient(conn)
		resp, err := client.Check(r.Context(), &healthpb.HealthCheckRequest{})
		if err != nil || resp.Status != healthpb.HealthCheckResponse_SERVING {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Metrics endpoint.
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(metrics.prometheusText()))
	})

	// OpenAPI spec serving.
	openAPIFiles := map[string]string{
		"/openapi/governance.yaml": "proto/oscal/gen/openapi/services/v1/governance_service.openapi.yaml",
		"/openapi/oscal.yaml":      "proto/oscal/gen/openapi/services/v1/oscal_service.openapi.yaml",
	}
	for route, relPath := range openAPIFiles {
		path := relPath
		http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			// #nosec G304 -- path is selected from the static OpenAPI route map above.
			data, err := os.ReadFile(path)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/x-yaml")
			_, _ = w.Write(data)
		})
	}

	// Wrap the gateway mux with metrics and CORS.
	root := http.NewServeMux()
	root.Handle("/", mux)
	for route := range openAPIFiles {
		root.Handle(route, http.DefaultServeMux)
	}
	root.Handle("/healthz", http.DefaultServeMux)
	root.Handle("/metrics", http.DefaultServeMux)

	handler := withCORS(withMetrics(root, metrics), *corsOrigin)

	server := &http.Server{
		Addr:              *httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down gateway")
		cancel()
		_ = server.Shutdown(context.Background())
	}()

	logger.Info("HTTP gateway listening", "addr", *httpAddr, "grpc", *grpcEndpoint)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gateway serve: %v", err)
	}
}

func withMetrics(next http.Handler, m *metricsCollector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseRecorder{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		d := time.Since(start)
		route := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		m.record(route, d)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := allowedOrigin
		if origin == "" {
			origin = r.Header.Get("Origin")
			if origin == "" {
				origin = "null"
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
