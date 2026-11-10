package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func TestBackupVerifyAndRestoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcDSN := filepath.Join(dir, "source.db")
	backupPath := filepath.Join(dir, "backup.db")
	restoredDSN := filepath.Join(dir, "restored.db")

	var out bytes.Buffer
	if err := run([]string{"-version"}, &out); err != nil {
		t.Fatalf("-version failed: %v", err)
	}
	if strings.TrimSpace(out.String()) != version {
		t.Fatalf("unexpected -version output %q", out.String())
	}

	// Seed a real SQLite store.
	store, err := kg.NewSQLiteStore(srcDSN, dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("open source store: %v", err)
	}
	if _, err := store.CreateSnapshot(context.Background(), "snap-1"); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	_ = store.Close()

	// Create backup + sidecar.
	out.Reset()
	if err := run([]string{"-dsn", srcDSN, "-out", backupPath}, &out); err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if _, err := os.Stat(backupPath + ".sha256"); err != nil {
		t.Fatalf("missing backup sidecar: %v", err)
	}

	// Verify backup.
	out.Reset()
	if err := run([]string{"-verify", "-file", backupPath}, &out); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !strings.Contains(out.String(), "[OK] SQLite integrity verified") {
		t.Fatalf("unexpected verify output: %s", out.String())
	}

	// Restore into fresh destination.
	out.Reset()
	if err := run([]string{"-restore", "-file", backupPath, "-dsn", restoredDSN}, &out); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if _, err := dbutil.VerifySQLite(context.Background(), restoredDSN); err != nil {
		t.Fatalf("restored SQLite failed verification: %v", err)
	}

	// Error cases.
	if err := run([]string{}, &out); err == nil {
		t.Fatal("expected error when -out is omitted")
	}
	if err := run([]string{"-verify"}, &out); err == nil {
		t.Fatal("expected error when -file is omitted in verify mode")
	}
	if err := run([]string{"-restore"}, &out); err == nil {
		t.Fatal("expected error when -file is omitted in restore mode")
	}
}
