package dbutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBackupSQLiteRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "source.db")
	backup := filepath.Join(dir, "backup", "snapshot.db")

	db, err := sql.Open("sqlite", source)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE claims (id TEXT PRIMARY KEY, trust_state TEXT NOT NULL)"); err != nil {
		t.Fatalf("create source schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO claims(id, trust_state) VALUES (?, ?)", "claim_backup_1", "candidate"); err != nil {
		t.Fatalf("insert source row: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	if err := BackupSQLite(ctx, source, backup); err != nil {
		t.Fatalf("backup: %v", err)
	}
	backupDB, err := sql.Open("sqlite", backup)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer backupDB.Close()
	var trustState string
	if err := backupDB.QueryRowContext(ctx, "SELECT trust_state FROM claims WHERE id = ?", "claim_backup_1").Scan(&trustState); err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if trustState != "candidate" {
		t.Fatalf("backup trust state = %q, want candidate", trustState)
	}

	if err := BackupSQLite(ctx, source, backup); err == nil {
		t.Fatal("second backup unexpectedly overwrote existing destination")
	}
}

func TestBackupSQLiteRejectsUnsafeInputs(t *testing.T) {
	ctx := context.Background()
	if err := BackupSQLite(ctx, ":memory:", filepath.Join(t.TempDir(), "backup.db")); err == nil {
		t.Fatal("in-memory database backup unexpectedly succeeded")
	}
	path := filepath.Join(t.TempDir(), "database.db")
	if err := BackupSQLite(ctx, path, path); err == nil {
		t.Fatal("same source and destination unexpectedly accepted")
	}
}
