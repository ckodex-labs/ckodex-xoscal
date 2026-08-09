package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/config"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/graph"
	"github.com/mchorfa/xoscal/server/internal/interceptors"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/observability"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
	"github.com/mchorfa/xoscal/server/internal/service"
	"github.com/mchorfa/xoscal/server/internal/store"
	"github.com/mchorfa/xoscal/server/internal/transparency"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to config file (optional)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := newLogger(cfg.Observability.LogLevel, cfg.Observability.LogFormat)
	logger.Info("starting xoscal-server", slog.String("version", version))
	ctx := context.Background()

	// Observability: tracing
	if cfg.Observability.TracingEnabled {
		shutdown, err := observability.InitTracing("xoscal-server", cfg.Observability.TracingSampleRate)
		if err != nil {
			logger.Warn("tracing init failed", slog.String("error", err.Error()))
		} else {
			defer func() { _ = shutdown(ctx) }()
		}
	}

	// Observability: metrics HTTP server
	var promMetrics *observability.PrometheusMetrics
	if cfg.Observability.MetricsEnabled {
		promMetrics = observability.NewPrometheusMetrics()
		go func() {
			logger.Info("metrics server listening", slog.String("addr", cfg.Observability.MetricsAddr))
			if err := promMetrics.Run(cfg.Observability.MetricsAddr); err != nil {
				logger.Error("metrics server failed", slog.String("error", err.Error()))
			}
		}()
	}

	// Optional pprof server
	if cfg.Server.EnablePProf {
		go func() {
			logger.Info("pprof server listening", slog.String("addr", cfg.Server.PProfAddr))
			mux := http.NewServeMux()
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			pprofServer := &http.Server{
				Addr:              cfg.Server.PProfAddr,
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       15 * time.Second,
				WriteTimeout:      15 * time.Second,
				IdleTimeout:       60 * time.Second,
			}
			if err := pprofServer.ListenAndServe(); err != nil {
				logger.Error("pprof server failed", slog.String("error", err.Error()))
			}
		}()
	}

	// Persistence stores with retry
	pool := dbutil.PoolConfig{
		MaxOpenConn:     cfg.Store.MaxOpenConn,
		MaxIdleConn:     cfg.Store.MaxIdleConn,
		ConnMaxLifetime: cfg.Store.ConnMaxLifetime,
	}

	s, err := openStoreWithRetry(cfg.Store.DSN, pool, 3, logger)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	kgStore, err := openKGWithRetry(cfg.Store.DSN, pool, 3, logger)
	if err != nil {
		log.Fatalf("open kg store: %v", err)
	}
	defer kgStore.Close()

	vectorStore, err := openVectorWithRetry(cfg.Vector, pool, 3, logger)
	if err != nil {
		log.Fatalf("open vector store: %v", err)
	}
	defer vectorStore.Close()

	transparencyStore, err := transparency.NewSQLiteStore(cfg.Store.DSN)
	if err != nil {
		log.Fatalf("open transparency store: %v", err)
	}
	defer transparencyStore.Close()

	graphStore, err := graph.NewSQLiteStore(cfg.Store.DSN)
	if err != nil {
		log.Fatalf("open graph store: %v", err)
	}
	defer graphStore.Close()

	// gRPC server with production interceptors and options
	var metricCollector interceptors.MetricsCollector
	if promMetrics != nil {
		metricCollector = promMetrics
	} else {
		metricCollector = &interceptors.InMemoryMetrics{}
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.Server.MaxRecvMsgSize * 1024 * 1024),
		grpc.MaxSendMsgSize(cfg.Server.MaxSendMsgSize * 1024 * 1024),
		grpc.MaxConcurrentStreams(cfg.Server.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.Server.KeepaliveTime,
			Timeout: cfg.Server.KeepaliveTimeout,
		}),
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRecovery(logger),
			interceptors.UnaryRequestID(),
			interceptors.UnaryAuth(interceptors.AuthConfig{
				Mode:      cfg.Security.AuthMode,
				SPIREAddr: cfg.Security.AuthSPIRESocket,
			}),
			interceptors.UnaryValidate(),
			interceptors.UnaryTracing(),
			interceptors.UnaryRateLimit(cfg.Security.RateLimitRPS, cfg.Security.RateLimitBurst),
			interceptors.UnaryLogging(logger),
			interceptors.UnaryMetrics(metricCollector),
		),
	}

	if tlsCreds, err := loadTLSCredentials(cfg.Server.TLSCertPath, cfg.Server.TLSKeyPath, cfg.Server.TLSClientCAPath); err != nil {
		logger.Warn("TLS credential load failed, starting without TLS", slog.String("error", err.Error()))
	} else if tlsCreds != nil {
		opts = append(opts, grpc.Creds(tlsCreds))
		logger.Info("gRPC TLS enabled")
	}
	grpcServer := grpc.NewServer(opts...)

	servicesv1.RegisterOscalServiceServer(grpcServer, service.NewOscalServer(s))
	rec := reconciler.NewReconciler(kgStore)
	servicesv1.RegisterGovernanceServiceServer(grpcServer, service.NewGovernanceServer(kgStore, rec, vectorStore))
	servicesv1.RegisterTransparencyExchangeServiceServer(grpcServer, transparency.NewExchangeServer(transparencyStore))
	servicesv1.RegisterTransparencyGraphServiceServer(grpcServer, graph.NewGraphServer(graphStore))

	if cfg.Server.EnableReflection {
		reflection.Register(grpcServer)
	}

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthSrv)

	lis, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("shutting down", slog.String("signal", sig.String()))

		shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
		defer cancel()

		stopped := make(chan struct{})
		go func() { grpcServer.GracefulStop(); close(stopped) }()

		select {
		case <-stopped:
			logger.Info("graceful shutdown completed")
		case <-shutdownCtx.Done():
			logger.Warn("graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		}
	}()

	logger.Info("OSCAL gRPC server listening", slog.String("addr", cfg.Server.Addr))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func newLogger(level, format string) *slog.Logger {
	var lv slog.Level
	_ = lv.UnmarshalText([]byte(level))
	opts := &slog.HandlerOptions{Level: lv}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

func openStoreWithRetry(dsn string, pool dbutil.PoolConfig, retries int, logger *slog.Logger) (store.Store, error) {
	var s store.Store
	var err error
	for i := 0; i <= retries; i++ {
		s, err = store.NewSQLiteStore(dsn, pool)
		if err == nil {
			return s, nil
		}
		logger.Warn("store open failed, retrying", slog.Int("attempt", i+1), slog.String("error", err.Error()))
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return nil, err
}

func openKGWithRetry(dsn string, pool dbutil.PoolConfig, retries int, logger *slog.Logger) (kg.Store, error) {
	var s kg.Store
	var err error
	for i := 0; i <= retries; i++ {
		s, err = kg.NewSQLiteStore(dsn, pool)
		if err == nil {
			return s, nil
		}
		logger.Warn("kg store open failed, retrying", slog.Int("attempt", i+1), slog.String("error", err.Error()))
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return nil, err
}

func openVectorWithRetry(vecCfg config.Vector, pool dbutil.PoolConfig, retries int, logger *slog.Logger) (embedding.VectorStore, error) {
	var s embedding.VectorStore
	var err error
	for i := 0; i <= retries; i++ {
		s, err = embedding.NewVectorStore(vecCfg, pool)
		if err == nil {
			return s, nil
		}
		logger.Warn("vector store open failed, retrying", slog.Int("attempt", i+1), slog.String("error", err.Error()))
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return nil, err
}

func loadTLSCredentials(certPath, keyPath, clientCAPath string) (credentials.TransportCredentials, error) {
	if certPath == "" && keyPath == "" {
		return nil, nil // TLS not configured
	}
	if certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("both tls_cert_path and tls_key_path must be set")
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if clientCAPath != "" {
		// #nosec G304 -- clientCAPath is an explicit operator-configured TLS trust-store path.
		caCert, err := os.ReadFile(clientCAPath)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append client CA certs")
		}
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return credentials.NewTLS(tlsConfig), nil
}
