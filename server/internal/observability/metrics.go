package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics holds the application metric collectors.
type PrometheusMetrics struct {
	rpcTotal    *prometheus.CounterVec
	rpcDuration *prometheus.HistogramVec
	registry    *prometheus.Registry
}

// NewPrometheusMetrics creates a fresh metrics instance with its own registry.
func NewPrometheusMetrics() *PrometheusMetrics {
	reg := prometheus.NewRegistry()
	rpcTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "xoscal_rpc_total",
		Help: "Total number of gRPC requests",
	}, []string{"method", "code"})
	rpcDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "xoscal_rpc_duration_seconds",
		Help:    "gRPC request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "code"})

	reg.MustRegister(rpcTotal, rpcDuration)
	return &PrometheusMetrics{
		rpcTotal:    rpcTotal,
		rpcDuration: rpcDuration,
		registry:    reg,
	}
}

// RecordRPC records a completed RPC.
func (m *PrometheusMetrics) RecordRPC(method, code string, duration time.Duration) {
	m.rpcTotal.WithLabelValues(method, code).Inc()
	m.rpcDuration.WithLabelValues(method, code).Observe(duration.Seconds())
}

// Handler returns an http.Handler for the /metrics endpoint.
func (m *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Run starts the metrics HTTP server on addr.
func (m *PrometheusMetrics) Run(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}
