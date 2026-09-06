package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesDefaultsForPartialConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  addr: 127.0.0.1:50052\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.Addr != "127.0.0.1:50052" {
		t.Fatalf("server addr = %q", cfg.Server.Addr)
	}
	if cfg.Server.MaxRecvMsgSize != 4 || cfg.Server.MaxSendMsgSize != 4 {
		t.Fatalf("partial config lost message limits: recv=%d send=%d", cfg.Server.MaxRecvMsgSize, cfg.Server.MaxSendMsgSize)
	}
	if cfg.Store.DSN != "oscal.db" || cfg.Security.AuthMode != "none" {
		t.Fatalf("partial config lost defaults: dsn=%q auth=%q", cfg.Store.DSN, cfg.Security.AuthMode)
	}
}
