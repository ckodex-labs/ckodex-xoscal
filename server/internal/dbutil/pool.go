// Package dbutil provides shared database connection utilities.
package dbutil

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PoolConfig holds SQLite connection pool tunables.
type PoolConfig struct {
	MaxOpenConn     int
	MaxIdleConn     int
	ConnMaxLifetime time.Duration
}

// DefaultPool returns a PoolConfig with production-safe defaults.
func DefaultPool() PoolConfig {
	return PoolConfig{
		MaxOpenConn:     10,
		MaxIdleConn:     2,
		ConnMaxLifetime: time.Hour,
	}
}

// Configure applies pool settings to a sql.DB. Zero or negative values are ignored.
// For in-memory SQLite databases (":memory:"), MaxOpenConns is forced to 1 so
// that all queries share the same ephemeral database.
func Configure(db *sql.DB, dsn string, cfg PoolConfig) error {
	isMemory := dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:")
	if isMemory {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		if cfg.MaxOpenConn > 0 {
			db.SetMaxOpenConns(cfg.MaxOpenConn)
		}
		if cfg.MaxIdleConn >= 0 {
			db.SetMaxIdleConns(cfg.MaxIdleConn)
		}
		if cfg.ConnMaxLifetime > 0 {
			db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		}
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping after pool config: %w", err)
	}
	return nil
}
