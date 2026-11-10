package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/graph"
	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/resilience"
	"github.com/mchorfa/xoscal/server/internal/utcp"
	"github.com/mchorfa/xoscal/server/internal/webstudio"
)

const maxTrackedRoutes = 128

var defaultHistogramBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
}

type routeHistogram struct {
	method  string
	path    string
	counts  map[int]int64
	buckets []int64 // non-cumulative counts per bucket index; accumulated at exposition time
	sum     float64
	count   int64
}

// metricsCollector tracks HTTP request counts and fixed-bucket latency histograms
// with strict cardinality bounding (maxTrackedRoutes) to prevent OOM leaks.
type metricsCollector struct {
	mu     sync.RWMutex
	routes map[string]*routeHistogram
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		routes: make(map[string]*routeHistogram, 32),
	}
}

func normalizeHTTPMethod(m string) string {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions:
		return strings.ToUpper(strings.TrimSpace(m))
	default:
		return "OTHER"
	}
}

func looksDynamicSegment(seg string) bool {
	if seg == "" {
		return false
	}
	// Numeric IDs
	allDigits := true
	for i := 0; i < len(seg); i++ {
		if seg[i] < '0' || seg[i] > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}
	// UUIDs or long hex/random tokens
	if len(seg) >= 16 {
		hexOrDash := true
		for i := 0; i < len(seg); i++ {
			c := seg[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-') {
				hexOrDash = false
				break
			}
		}
		if hexOrDash {
			return true
		}
	}
	return false
}

// normalizeMetricsRoute canonicalizes URL paths to bound Prometheus label cardinality.
func normalizeMetricsRoute(rawPath string) string {
	p := strings.TrimSpace(rawPath)
	if idx := strings.IndexByte(p, '?'); idx >= 0 {
		p = p[:idx]
	}
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) > 6 {
		parts = append(parts[:6], ":tail")
	}
	for i, seg := range parts {
		if looksDynamicSegment(seg) {
			parts[i] = ":id"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func (m *metricsCollector) record(rawRoute string, status int, d time.Duration) {
	parts := strings.SplitN(strings.TrimSpace(rawRoute), " ", 2)
	method := "GET"
	rawPath := rawRoute
	if len(parts) == 2 {
		method = normalizeHTTPMethod(parts[0])
		rawPath = parts[1]
	} else {
		method = "OTHER"
	}
	normPath := normalizeMetricsRoute(rawPath)
	key := method + " " + normPath

	sec := d.Seconds()
	if sec < 0 {
		sec = 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	rh, exists := m.routes[key]
	if !exists {
		overflowKey := "OTHER OTHER"
		if len(m.routes) >= maxTrackedRoutes-1 && key != overflowKey {
			method = "OTHER"
			normPath = "OTHER"
			key = overflowKey
			rh = m.routes[key]
		}
		if rh == nil {
			rh = &routeHistogram{
				method:  method,
				path:    normPath,
				counts:  make(map[int]int64, 4),
				buckets: make([]int64, len(defaultHistogramBuckets)),
			}
			m.routes[key] = rh
		}
	}

	// Bound distinct status codes per route as well
	if status < 100 || status > 599 {
		status = 0
	}
	rh.counts[status]++
	rh.sum += sec
	rh.count++

	for i, bound := range defaultHistogramBuckets {
		if sec <= bound {
			rh.buckets[i]++
			break
		}
	}
}

func (m *metricsCollector) trackedRouteCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.routes)
}

func (m *metricsCollector) prometheusText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	b.WriteString("# TYPE http_request_total counter\n")
	b.WriteString("# HELP http_request_total Total HTTP requests by route and status\n")

	keys := make([]string, 0, len(m.routes))
	for k := range m.routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		rh := m.routes[k]
		statuses := make([]int, 0, len(rh.counts))
		for st := range rh.counts {
			statuses = append(statuses, st)
		}
		sort.Ints(statuses)
		for _, st := range statuses {
			fmt.Fprintf(&b, "http_request_total{method=%s,path=%s,status=%s} %d\n",
				strconv.Quote(rh.method), strconv.Quote(rh.path), strconv.Quote(strconv.Itoa(st)), rh.counts[st])
		}
	}

	b.WriteString("# TYPE http_request_duration_seconds histogram\n")
	b.WriteString("# HELP http_request_duration_seconds HTTP request latencies in seconds\n")
	for _, k := range keys {
		rh := m.routes[k]
		if rh.count == 0 {
			continue
		}
		qMethod := strconv.Quote(rh.method)
		qPath := strconv.Quote(rh.path)
		var cumulative int64
		for i, bound := range defaultHistogramBuckets {
			cumulative += rh.buckets[i]
			leStr := strconv.FormatFloat(bound, 'g', -1, 64)
			fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%s,path=%s,le=%s} %d\n",
				qMethod, qPath, strconv.Quote(leStr), cumulative)
		}
		fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%s,path=%s,le=\"+Inf\"} %d\n",
			qMethod, qPath, rh.count)
		fmt.Fprintf(&b, "http_request_duration_seconds_sum{method=%s,path=%s} %.6f\n",
			qMethod, qPath, rh.sum)
		fmt.Fprintf(&b, "http_request_duration_seconds_count{method=%s,path=%s} %d\n",
			qMethod, qPath, rh.count)
	}
	return b.String()
}

// healthProber reuses a single shared upstream *grpc.ClientConn across /healthz and /readyz probes
// to eliminate per-probe TCP socket churn.
type healthProber struct {
	mu       sync.Mutex
	endpoint string
	dialOpts []grpc.DialOption
	conn     *grpc.ClientConn
}

func newHealthProber(endpoint string, dialOpts []grpc.DialOption) *healthProber {
	return &healthProber{
		endpoint: endpoint,
		dialOpts: dialOpts,
	}
}

func (p *healthProber) getConn() (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		return p.conn, nil
	}
	conn, err := grpc.NewClient(p.endpoint, p.dialOpts...)
	if err != nil {
		return nil, err
	}
	p.conn = conn
	return conn, nil
}

func (p *healthProber) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	conn, err := p.getConn()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	probeCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(probeCtx, &healthpb.HealthCheckRequest{})
	if err != nil || resp.Status != healthpb.HealthCheckResponse_SERVING {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (p *healthProber) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		return nil
	}
	err := p.conn.Close()
	p.conn = nil
	return err
}

func sanitizeGatewayHost(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 255 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == ':' || c == '[' || c == ']') {
			return ""
		}
	}
	return s
}

func registerUTCPHandlers(mux *http.ServeMux, grpcEndpoint string) {
	if ws, err := webstudio.NewServer(webstudio.Config{Registry: frameworks.DefaultRegistry()}); err == nil {
		ws.RegisterFrameworkAndValidationRoutes(mux)
	}

	handleManual := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		scheme := "http://"
		if r.TLS != nil {
			scheme = "https://"
		}
		baseURL := "http://127.0.0.1:8080"
		if host := sanitizeGatewayHost(r.Host); host != "" {
			baseURL = scheme + host
		}
		manual := utcp.GenerateManual(utcp.ManualOptions{
			BaseHTTPURL: baseURL,
			GRPCHost:    grpcEndpoint,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(manual)
	}

	handleValidate := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		rep, err := utcp.ValidateManual(body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rep)
	}

	mux.HandleFunc("/.well-known/utcp", handleManual)
	mux.HandleFunc("/api/v1/utcp", handleManual)
	mux.HandleFunc("/api/v1/utcp/validate", handleValidate)

	mux.HandleFunc("/api/v1/ontology", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ontology.CanonicalSpec())
	})

	mux.HandleFunc("/api/v1/graph/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ug, err := graph.DefaultUnifiedGraph()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ug.Stats())
	})

	mux.HandleFunc("/api/v1/graph/crosswalk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctrl := strings.TrimSpace(r.URL.Query().Get("control"))
		if ctrl == "" {
			ctrl = "ac-2"
		}
		ug, err := graph.DefaultUnifiedGraph()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rep, err := ug.QueryCrosswalk(ctrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rep)
	})

	mux.HandleFunc("/api/v1/graph/blast-radius", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		target := strings.TrimSpace(r.URL.Query().Get("target"))
		if target == "" {
			target = "CVE-2024-3400"
		}
		depth := 2
		if dStr := strings.TrimSpace(r.URL.Query().Get("depth")); dStr != "" {
			if dVal, err := strconv.Atoi(dStr); err == nil && dVal > 0 {
				depth = dVal
			}
		}
		ug, err := graph.DefaultUnifiedGraph()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rep, err := ug.ComputeBlastRadius(target, depth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rep)
	})

	mux.HandleFunc("/api/v1/graph/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		target := strings.TrimSpace(r.URL.Query().Get("target"))
		if target == "" {
			target = "CVE-2024-3400"
		}
		format := strings.TrimSpace(r.URL.Query().Get("format"))
		if format == "" {
			format = "jsonld"
		}
		depth := 2
		if dStr := strings.TrimSpace(r.URL.Query().Get("depth")); dStr != "" {
			if dVal, err := strconv.Atoi(dStr); err == nil && dVal > 0 {
				depth = dVal
			}
		}
		ug, err := graph.DefaultUnifiedGraph()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := ug.ExportSubgraph(target, depth, format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.EqualFold(format, "cypher") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		// #nosec G705 -- response is strictly text/plain or application/json with X-Content-Type-Options: nosniff.
		_, _ = w.Write([]byte(out))
	})

	mux.HandleFunc("/api/v1/feeds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		feedID := strings.TrimSpace(r.URL.Query().Get("feed"))
		if feedID != "" {
			insp, err := feeds.InspectFeed(feedID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(insp)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(feeds.ListFeeds())
	})

	mux.HandleFunc("/api/v1/feeds/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		feedID := strings.TrimSpace(r.URL.Query().Get("feed"))
		receipt, err := feeds.SyncFeeds(r.Context(), feeds.SyncOptions{
			FeedID:  feedID,
			Offline: true,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(receipt)
	})
}

func registerGatewayObservability(root *http.ServeMux, prober *healthProber, metrics *metricsCollector) {
	if prober != nil {
		root.Handle("/healthz", prober)
		root.Handle("/readyz", prober)
	}
	if metrics != nil {
		root.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
			_, _ = w.Write([]byte(metrics.prometheusText()))
		})
	}

	openAPIFiles := map[string]string{
		"/openapi/governance.yaml": "proto/oscal/gen/openapi/services/v1/governance_service.openapi.yaml",
		"/openapi/oscal.yaml":      "proto/oscal/gen/openapi/services/v1/oscal_service.openapi.yaml",
	}
	for route, relPath := range openAPIFiles {
		path := relPath
		root.HandleFunc(route, func(w http.ResponseWriter, _ *http.Request) {
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

	metrics := newMetricsCollector()
	prober := newHealthProber(*grpcEndpoint, dialOpts)
	defer func() { _ = prober.Close() }()

	root := http.NewServeMux()
	registerUTCPHandlers(root, *grpcEndpoint)
	registerGatewayObservability(root, prober, metrics)
	root.Handle("/", mux)

	handler := buildGatewayHandler(root, metrics, token, parseAllowedOrigins(*corsOrigin), 128)

	server := &http.Server{
		Addr:              *httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down gateway")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
		_ = prober.Close()
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

func buildGatewayHandler(root http.Handler, metrics *metricsCollector, token string, allowedOrigins map[string]bool, maxInFlight int) http.Handler {
	h := withTokenAuth(root, token)
	h = withCORS(h, allowedOrigins)
	h = resilience.WrapHTTPHandler(h, maxInFlight)
	if metrics != nil {
		h = withMetrics(h, metrics)
	}
	return h
}

func withMetrics(next http.Handler, m *metricsCollector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		d := time.Since(start)
		route := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		m.record(route, rw.statusCode, d)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.wroteHeader {
		rr.statusCode = code
		rr.wroteHeader = true
	}
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.wroteHeader {
		rr.wroteHeader = true
	}
	return rr.ResponseWriter.Write(b)
}

func (rr *responseRecorder) Flush() {
	if fl, ok := rr.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func (rr *responseRecorder) Unwrap() http.ResponseWriter {
	return rr.ResponseWriter
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
