package dbutil

import (
	"context"
	"database/sql"
	"os"
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

func TestVerifySQLiteAndSidecar(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "verify_test.db")

	db, err := sql.Open("sqlite", source)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE controls (id TEXT PRIMARY KEY, title TEXT)"); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO controls VALUES ('ac-1', 'Access Control')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_ = db.Close()

	// 1. Verify valid SQLite file
	digest, err := VerifySQLite(ctx, source)
	if err != nil {
		t.Fatalf("VerifySQLite failed: %v", err)
	}
	if digest == "" {
		t.Fatal("expected non-empty digest")
	}

	// 2. Write and check sidecar
	writtenDigest, err := WriteDigestSidecar(source)
	if err != nil {
		t.Fatalf("WriteDigestSidecar failed: %v", err)
	}
	if writtenDigest != digest {
		t.Fatalf("digest mismatch: got %s, want %s", writtenDigest, digest)
	}

	if err := CheckDigestSidecar(source); err != nil {
		t.Fatalf("CheckDigestSidecar failed: %v", err)
	}

	// 3. Test tampered sidecar
	sidecarPath := source + ".sha256"
	if err := os.WriteFile(sidecarPath, []byte("sha256:0000000000000000000000000000000000000000000000000000000000000000\n"), 0o600); err != nil {
		t.Fatalf("write tampered sidecar: %v", err)
	}
	if err := CheckDigestSidecar(source); err == nil {
		t.Fatal("CheckDigestSidecar should have failed on tampered digest")
	}
}

func TestRestoreSQLiteDisasterRecoveryDrill(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	sourceDB := filepath.Join(dir, "live_production.db")
	backupSnapshot := filepath.Join(dir, "snapshots", "2026-09-25_backup.db")
	restoredDB := filepath.Join(dir, "restored_production.db")

	// Step 1: Create live production database with critical compliance records
	db, err := sql.Open("sqlite", sourceDB)
	if err != nil {
		t.Fatalf("create live db: %v", err)
	}
	_, err = db.ExecContext(ctx, `
		CREATE TABLE oscal_entities (urn TEXT PRIMARY KEY, kind TEXT, payload TEXT);
		CREATE TABLE vector_states (entity_urn TEXT PRIMARY KEY, presence TEXT, valence TEXT, coherence TEXT);
		INSERT INTO oscal_entities VALUES ('urn:xoscal:comp:auth-svc', 'component', '{"name":"auth-svc"}');
		INSERT INTO vector_states VALUES ('urn:xoscal:comp:auth-svc', 'PRESENT', 'POSITIVE', 'COHERENT');
	`)
	if err != nil {
		t.Fatalf("seed live db: %v", err)
	}
	_ = db.Close()

	// Step 2: Perform consistent backup snapshot
	if err := BackupSQLite(ctx, sourceDB, backupSnapshot); err != nil {
		t.Fatalf("BackupSQLite failed: %v", err)
	}

	// Step 3: Generate and verify cryptographic digest sidecar
	digest, err := WriteDigestSidecar(backupSnapshot)
	if err != nil {
		t.Fatalf("WriteDigestSidecar failed: %v", err)
	}
	if err := CheckDigestSidecar(backupSnapshot); err != nil {
		t.Fatalf("CheckDigestSidecar failed: %v", err)
	}

	// Step 4: Simulate catastrophic production corruption
	if err := os.WriteFile(sourceDB, []byte("CORRUPTED_RAW_BYTES_SIMULATING_DISK_FAILURE"), 0o600); err != nil {
		t.Fatalf("corrupt source db: %v", err)
	}

	// Step 5: Execute restore drill to restored location
	if err := RestoreSQLite(ctx, backupSnapshot, restoredDB, false); err != nil {
		t.Fatalf("RestoreSQLite failed: %v", err)
	}

	// Step 6: Verify restored database integrity and data fidelity
	restoredDigest, err := VerifySQLite(ctx, restoredDB)
	if err != nil {
		t.Fatalf("VerifySQLite on restored db failed: %v", err)
	}
	if restoredDigest != digest {
		t.Fatalf("restored db digest %s != backup digest %s", restoredDigest, digest)
	}

	rdb, err := sql.Open("sqlite", restoredDB)
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}
	defer rdb.Close()

	var kind, payload string
	err = rdb.QueryRowContext(ctx, "SELECT kind, payload FROM oscal_entities WHERE urn = ?", "urn:xoscal:comp:auth-svc").Scan(&kind, &payload)
	if err != nil {
		t.Fatalf("query restored entity: %v", err)
	}
	if kind != "component" || payload != `{"name":"auth-svc"}` {
		t.Fatalf("data mismatch in restored database: kind=%s, payload=%s", kind, payload)
	}

	var presence, valence, coherence string
	err = rdb.QueryRowContext(ctx, "SELECT presence, valence, coherence FROM vector_states WHERE entity_urn = ?", "urn:xoscal:comp:auth-svc").Scan(&presence, &valence, &coherence)
	if err != nil {
		t.Fatalf("query restored vector state: %v", err)
	}
	if presence != "PRESENT" || valence != "POSITIVE" || coherence != "COHERENT" {
		t.Fatalf("vector state corruption: P=%s, V=%s, C=%s", presence, valence, coherence)
	}

	// Step 7: Verify overwrite protection (safety guard)
	if err := RestoreSQLite(ctx, backupSnapshot, restoredDB, false); err == nil {
		t.Fatal("expected RestoreSQLite to fail when target exists and overwrite=false")
	}
}
