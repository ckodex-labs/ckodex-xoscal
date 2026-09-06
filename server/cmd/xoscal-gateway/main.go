package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// metricsCollector tracks HTTP request counts and durations without external deps.
type metricsCollector struct {
	mu        sync.RWMutex
	counts    map[string]map[int]int64
	durations map[string][]time.Duration
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		counts:    make(map[string]map[int]int64),
		durations: make(map[string][]time.Duration),
	}
}

func (m *metricsCollector) record(route string, status int, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counts[route] == nil {
		m.counts[route] = make(map[int]int64)
	}
	m.counts[route][status]++
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
		statuses := make([]int, 0, len(m.counts[r]))
		for status := range m.counts[r] {
			statuses = append(statuses, status)
		}
		sort.Ints(statuses)
		for _, status := range statuses {
			count := m.counts[r][status]
			fmt.Fprintf(&b, "http_request_total{method=%s,path=%s,status=%s} %d\n", strconv.Quote(method), strconv.Quote(path), strconv.Quote(strconv.Itoa(status)), count)
		}
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
	corsOrigin := flag.String("cors-origin", "http://127.0.0.1:8765,http://localhost:8765", "comma-separated CORS allowed origins")
	tokenFile := flag.String("token-file", "", "bearer token file for HTTP and upstream gRPC auth")
	grpcTLSCA := flag.String("grpc-tls-ca", "", "CA bundle for the upstream gRPC server")
	grpcTLSCert := flag.String("grpc-tls-cert", "", "client certificate for upstream gRPC mTLS")
	grpcTLSKey := flag.String("grpc-tls-key", "", "client key for upstream gRPC mTLS")
	grpcTLSServerName := flag.String("grpc-tls-server-name", "", "TLS server name for the upstream gRPC server")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	token, err := loadGatewayToken(*tokenFile)
	if err != nil {
		log.Fatalf("load gateway token: %v", err)
	}
	dialOpts, err := grpcDialOptions(*grpcTLSCA, *grpcTLSCert, *grpcTLSKey, *grpcTLSServerName, token)
	if err != nil {
		log.Fatalf("configure upstream gRPC transport: %v", err)
	}

	mux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, newOSCALMarshaler()),
	)

	if err := servicesv1.RegisterGovernanceServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register governance handler: %v", err)
	}
	if err := servicesv1.RegisterOscalServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register oscal handler: %v", err)
	}
	if err := servicesv1.RegisterTransparencyExchangeServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register transparency exchange handler: %v", err)
	}
	if err := servicesv1.RegisterTransparencyGraphServiceHandlerFromEndpoint(ctx, mux, *grpcEndpoint, dialOpts); err != nil {
		log.Fatalf("register transparency graph handler: %v", err)
	}

	// --- Observability ---
	metrics := newMetricsCollector()

	// Health check: probe gRPC server.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		conn, err := grpc.Dial(*grpcEndpoint, dialOpts...)
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

	handler := withCORS(withTokenAuth(withMetrics(root, metrics), token), parseAllowedOrigins(*corsOrigin))

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

type staticTokenCredentials struct {
	token string
}

func (c staticTokenCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}

func (c staticTokenCredentials) RequireTransportSecurity() bool {
	return true
}

func loadGatewayToken(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	// #nosec G304 -- path is an explicit operator-configured secret path.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line, nil
		}
	}
	return "", fmt.Errorf("token file %q does not contain a token", path)
}

func grpcDialOptions(caPath, certPath, keyPath, serverName, token string) ([]grpc.DialOption, error) {
	tlsRequested := strings.TrimSpace(caPath) != "" || strings.TrimSpace(certPath) != "" || strings.TrimSpace(keyPath) != "" || strings.TrimSpace(serverName) != ""
	if !tlsRequested {
		if token != "" {
			return nil, fmt.Errorf("upstream token authentication requires TLS configuration")
		}
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, nil
	}
	if (certPath == "") != (keyPath == "") {
		return nil, fmt.Errorf("grpc client TLS certificate and key must be configured together")
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName}
	if caPath != "" {
		// #nosec G304 -- caPath is an explicit operator-configured trust-store path.
		data, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("failed to append upstream gRPC CA certificates")
		}
		tlsConfig.RootCAs = pool
	}
	if certPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	if token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(staticTokenCredentials{token: token}))
	}
	return opts, nil
}

func withMetrics(next http.Handler, m *metricsCollector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseRecorder{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)
		d := time.Since(start)
		route := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		m.record(route, rw.statusCode, d)
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

func parseAllowedOrigins(value string) map[string]bool {
	origins := make(map[string]bool)
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" && origin != "*" {
			origins[origin] = true
		}
	}
	return origins
}

func withCORS(next http.Handler, allowedOrigins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !allowedOrigins[origin] {
				if r.Method == http.MethodOptions {
					http.Error(w, "origin is not allowed", http.StatusForbidden)
					return
				}
				w.Header().Set("Vary", "Origin")
				http.Error(w, "origin is not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withTokenAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(provided, "Bearer ") {
			provided = strings.TrimSpace(strings.TrimPrefix(provided, "Bearer "))
		}
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
