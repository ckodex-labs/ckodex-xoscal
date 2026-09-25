package dbutil

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// ComputeFileSHA256 returns the prefixed SHA-256 digest ("sha256:<hex>") of a file.
func ComputeFileSHA256(filePath string) (string, error) {
	// #nosec G304 -- explicit file path supplied by operator
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// VerifySQLite validates the structural integrity and calculates the digest
// of an SQLite database file using PRAGMA integrity_check.
func VerifySQLite(ctx context.Context, filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat database: %w", err)
	}
	if info.Size() == 0 {
		return "", errors.New("database file is empty")
	}

	digest, err := ComputeFileSHA256(absPath)
	if err != nil {
		return "", fmt.Errorf("compute digest: %w", err)
	}

	// Open read-only to verify database structure
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", absPath))
	if err != nil {
		return "", fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	var result string
	row := db.QueryRowContext(ctx, "PRAGMA integrity_check;")
	if err := row.Scan(&result); err != nil {
		return "", fmt.Errorf("run integrity check: %w", err)
	}
	if result != "ok" {
		return "", fmt.Errorf("database integrity check failed: %s", result)
	}

	return digest, nil
}

// WriteDigestSidecar calculates the SHA-256 digest of filePath and writes
// it to filePath + ".sha256".
func WriteDigestSidecar(filePath string) (string, error) {
	digest, err := ComputeFileSHA256(filePath)
	if err != nil {
		return "", err
	}
	sidecar := filePath + ".sha256"
	// #nosec G703,G304,G306 -- bounded sidecar path
	if err := os.WriteFile(sidecar, []byte(digest+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write sidecar: %w", err)
	}
	return digest, nil
}

// CheckDigestSidecar verifies that filePath matches the digest in filePath + ".sha256".
func CheckDigestSidecar(filePath string) error {
	sidecar := filePath + ".sha256"
	// #nosec G304 -- explicit sidecar path
	data, err := os.ReadFile(sidecar)
	if err != nil {
		return fmt.Errorf("read sidecar: %w", err)
	}
	expected := strings.TrimSpace(string(data))

	actual, err := ComputeFileSHA256(filePath)
	if err != nil {
		return fmt.Errorf("compute actual digest: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// RestoreSQLite restores a verified SQLite snapshot into a target database location.
// If target exists and overwrite is false, it returns an error to prevent accidental destruction.
func RestoreSQLite(ctx context.Context, backupFile, targetDSN string, overwrite bool) error {
	if strings.TrimSpace(backupFile) == "" {
		return errors.New("backup file is required")
	}
	if strings.TrimSpace(targetDSN) == "" {
		return errors.New("target dsn is required")
	}

	backupAbs, err := filepath.Abs(backupFile)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}

	// 1. Verify backup integrity before restoration
	if _, err := VerifySQLite(ctx, backupAbs); err != nil {
		return fmt.Errorf("preflight verify backup failed: %w", err)
	}

	// 2. If sidecar exists, verify cryptographic digest
	sidecar := backupAbs + ".sha256"
	if _, err := os.Stat(sidecar); err == nil {
		if err := CheckDigestSidecar(backupAbs); err != nil {
			return fmt.Errorf("cryptographic sidecar check failed: %w", err)
		}
	}

	// 3. Resolve destination
	targetPath := sqliteFilePath(targetDSN)
	if targetPath == "" {
		return errors.New("target DSN must resolve to a file path")
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	if targetAbs == backupAbs {
		return errors.New("target database path cannot be the same as backup file")
	}

	if _, err := os.Stat(targetAbs); err == nil {
		if !overwrite {
			return fmt.Errorf("target database %s already exists and overwrite is false", targetAbs)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o750); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	// 4. Perform atomic copy via temp file
	tmpTarget := targetAbs + ".restore.tmp"
	defer func() {
		_ = os.Remove(tmpTarget)
	}()

	// #nosec G304 -- explicit paths
	src, err := os.Open(backupAbs)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer src.Close()

	// #nosec G304 -- bounded temp target
	dst, err := os.OpenFile(tmpTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create temp target: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("copy backup to temp target: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("close temp target: %w", err)
	}

	// 5. Verify restored database before moving into place
	if _, err := VerifySQLite(ctx, tmpTarget); err != nil {
		return fmt.Errorf("verify restored temp target failed: %w", err)
	}

	// 6. Rename temp file to final destination
	if err := os.Rename(tmpTarget, targetAbs); err != nil {
		return fmt.Errorf("rename to target path: %w", err)
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
