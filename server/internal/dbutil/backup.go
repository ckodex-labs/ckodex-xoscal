package dbutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// BackupSQLite creates a consistent SQLite backup without overwriting an
// existing destination. VACUUM INTO runs inside SQLite, so the output is a
// coherent database snapshot rather than a best-effort file copy.
func BackupSQLite(ctx context.Context, dsn, destination string) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("sqlite dsn is required")
	}
	if dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return errors.New("in-memory sqlite databases cannot be backed up")
	}
	if strings.TrimSpace(destination) == "" {
		return errors.New("backup destination is required")
	}

	destination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve backup destination: %w", err)
	}
	if info, statErr := os.Stat(destination); statErr == nil {
		return fmt.Errorf("backup destination already exists: %s (%d bytes)", destination, info.Size())
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect backup destination: %w", statErr)
	}

	if sourcePath := sqliteFilePath(dsn); sourcePath != "" {
		sourceAbs, absErr := filepath.Abs(sourcePath)
		if absErr == nil && sourceAbs == destination {
			return errors.New("backup destination must differ from the source database")
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open sqlite source: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite source: %w", err)
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		return fmt.Errorf("create sqlite backup: %w", err)
	}
	if err := os.Chmod(destination, 0o600); err != nil {
		return fmt.Errorf("protect sqlite backup: %w", err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		return fmt.Errorf("stat sqlite backup: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("sqlite backup is empty")
	}
	return nil
}

func sqliteFilePath(dsn string) string {
	if strings.HasPrefix(dsn, "file:") {
		dsn = strings.TrimPrefix(dsn, "file:")
	}
	if index := strings.IndexByte(dsn, '?'); index >= 0 {
		dsn = dsn[:index]
	}
	if dsn == "" || strings.HasPrefix(dsn, ":") {
		return ""
	}
	return dsn
}
