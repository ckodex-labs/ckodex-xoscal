package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Server holds gRPC server tunables.
type Server struct {
	Addr                 string        `mapstructure:"addr"`
	MaxRecvMsgSize       int           `mapstructure:"max_recv_msg_size_mb"`
	MaxSendMsgSize       int           `mapstructure:"max_send_msg_size_mb"`
	MaxConcurrentStreams uint32        `mapstructure:"max_concurrent_streams"`
	KeepaliveTime        time.Duration `mapstructure:"keepalive_time"`
	KeepaliveTimeout     time.Duration `mapstructure:"keepalive_timeout"`
	EnableReflection     bool          `mapstructure:"enable_reflection"`
	EnablePProf          bool          `mapstructure:"enable_pprof"`
	PProfAddr            string        `mapstructure:"pprof_addr"`
	ShutdownTimeout      time.Duration `mapstructure:"shutdown_timeout"`
	TLSCertPath          string        `mapstructure:"tls_cert_path"`
	TLSKeyPath           string        `mapstructure:"tls_key_path"`
	TLSClientCAPath      string        `mapstructure:"tls_client_ca_path"`
	RequireTLS           bool          `mapstructure:"require_tls"`
}

// Store holds persistence tunables.
type Store struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConn     int           `mapstructure:"max_open_conn"`
	MaxIdleConn     int           `mapstructure:"max_idle_conn"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// Vector holds vector store backend selection and connection options.
type Vector struct {
	Backend       string `mapstructure:"backend"` // "lancedb" or "sqlite" (default)
	URI           string `mapstructure:"uri"`     // e.g. "./data/vectors" or "s3://bucket/prefix"
	S3Region      string `mapstructure:"s3_region"`
	S3KeyID       string `mapstructure:"s3_access_key_id"`
	S3Secret      string `mapstructure:"s3_secret_access_key"`
	OpenAIKey     string `mapstructure:"openai_api_key"`
	OpenAIModel   string `mapstructure:"openai_model"`    // e.g. "text-embedding-3-small"
	OpenAIBaseURL string `mapstructure:"openai_base_url"` // defaults to https://api.openai.com/v1
}

// Observability holds metrics and tracing settings.
type Observability struct {
	MetricsEnabled    bool    `mapstructure:"metrics_enabled"`
	MetricsAddr       string  `mapstructure:"metrics_addr"`
	TracingEnabled    bool    `mapstructure:"tracing_enabled"`
	TracingEndpoint   string  `mapstructure:"tracing_endpoint"`
	TracingSampleRate float64 `mapstructure:"tracing_sample_rate"`
	LogLevel          string  `mapstructure:"log_level"`
	LogFormat         string  `mapstructure:"log_format"`
}

// Security holds auth and rate-limit settings.
type Security struct {
	RateLimitRPS    float64  `mapstructure:"rate_limit_rps"`
	RateLimitBurst  int      `mapstructure:"rate_limit_burst"`
	AuthMode        string   `mapstructure:"auth_mode"`
	AuthSPIRESocket string   `mapstructure:"auth_spire_socket"`
	AuthTokenFile   string   `mapstructure:"auth_token_file"`
	AuthTokens      []string `mapstructure:"auth_tokens"`
}

// Config is the root application configuration.
type Config struct {
	Server        Server        `mapstructure:"server"`
	Store         Store         `mapstructure:"store"`
	Vector        Vector        `mapstructure:"vector"`
	Observability Observability `mapstructure:"observability"`
	Security      Security      `mapstructure:"security"`
}

// Default returns a Config populated with production-safe defaults.
func Default() *Config {
	return &Config{
		Server: Server{
			Addr:                 ":50051",
			MaxRecvMsgSize:       4, // 4 MB
			MaxSendMsgSize:       4,
			MaxConcurrentStreams: 1000,
			KeepaliveTime:        2 * time.Minute,
			KeepaliveTimeout:     20 * time.Second,
			EnableReflection:     true,
			EnablePProf:          false,
			PProfAddr:            ":6060",
			ShutdownTimeout:      30 * time.Second,
		},
		Store: Store{
			DSN:             "oscal.db",
			MaxOpenConn:     10,
			MaxIdleConn:     2,
			ConnMaxLifetime: time.Hour,
		},
		Vector: Vector{
			Backend: "sqlite",
			URI:     "",
		},
		Observability: Observability{
			MetricsEnabled:    true,
			MetricsAddr:       ":9090",
			TracingEnabled:    false,
			TracingEndpoint:   "",
			TracingSampleRate: 0.1,
			LogLevel:          "info",
			LogFormat:         "json",
		},
		Security: Security{
			RateLimitRPS:   100,
			RateLimitBurst: 200,
			AuthMode:       "none",
		},
	}
}

// Load reads configuration from file, environment, and flags.
// Precedence: flags > env > config file > defaults.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	defaults := Default()
	setDefaults(v, defaults)

	// Environment
	v.SetEnvPrefix("XOSCAL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Config file
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper, defaults *Config) {
	v.SetDefault("server.addr", defaults.Server.Addr)
	v.SetDefault("server.max_recv_msg_size_mb", defaults.Server.MaxRecvMsgSize)
	v.SetDefault("server.max_send_msg_size_mb", defaults.Server.MaxSendMsgSize)
	v.SetDefault("server.max_concurrent_streams", defaults.Server.MaxConcurrentStreams)
	v.SetDefault("server.keepalive_time", defaults.Server.KeepaliveTime)
	v.SetDefault("server.keepalive_timeout", defaults.Server.KeepaliveTimeout)
	v.SetDefault("server.enable_reflection", defaults.Server.EnableReflection)
	v.SetDefault("server.enable_pprof", defaults.Server.EnablePProf)
	v.SetDefault("server.pprof_addr", defaults.Server.PProfAddr)
	v.SetDefault("server.shutdown_timeout", defaults.Server.ShutdownTimeout)
	v.SetDefault("server.tls_cert_path", defaults.Server.TLSCertPath)
	v.SetDefault("server.tls_key_path", defaults.Server.TLSKeyPath)
	v.SetDefault("server.tls_client_ca_path", defaults.Server.TLSClientCAPath)
	v.SetDefault("server.require_tls", defaults.Server.RequireTLS)

	v.SetDefault("store.dsn", defaults.Store.DSN)
	v.SetDefault("store.max_open_conn", defaults.Store.MaxOpenConn)
	v.SetDefault("store.max_idle_conn", defaults.Store.MaxIdleConn)
	v.SetDefault("store.conn_max_lifetime", defaults.Store.ConnMaxLifetime)

	v.SetDefault("vector.backend", defaults.Vector.Backend)
	v.SetDefault("vector.uri", defaults.Vector.URI)
	v.SetDefault("vector.s3_region", defaults.Vector.S3Region)
	v.SetDefault("vector.s3_access_key_id", defaults.Vector.S3KeyID)
	v.SetDefault("vector.s3_secret_access_key", defaults.Vector.S3Secret)
	v.SetDefault("vector.openai_api_key", defaults.Vector.OpenAIKey)
	v.SetDefault("vector.openai_model", defaults.Vector.OpenAIModel)
	v.SetDefault("vector.openai_base_url", defaults.Vector.OpenAIBaseURL)

	v.SetDefault("observability.metrics_enabled", defaults.Observability.MetricsEnabled)
	v.SetDefault("observability.metrics_addr", defaults.Observability.MetricsAddr)
	v.SetDefault("observability.tracing_enabled", defaults.Observability.TracingEnabled)
	v.SetDefault("observability.tracing_endpoint", defaults.Observability.TracingEndpoint)
	v.SetDefault("observability.tracing_sample_rate", defaults.Observability.TracingSampleRate)
	v.SetDefault("observability.log_level", defaults.Observability.LogLevel)
	v.SetDefault("observability.log_format", defaults.Observability.LogFormat)

	v.SetDefault("security.rate_limit_rps", defaults.Security.RateLimitRPS)
	v.SetDefault("security.rate_limit_burst", defaults.Security.RateLimitBurst)
	v.SetDefault("security.auth_mode", defaults.Security.AuthMode)
	v.SetDefault("security.auth_spire_socket", defaults.Security.AuthSPIRESocket)
	v.SetDefault("security.auth_token_file", defaults.Security.AuthTokenFile)
}
