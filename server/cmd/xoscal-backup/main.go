// Command xoscal-backup manages SQLite database snapshots, integrity verification,
// and disaster recovery restores for xOSCAL deployments.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("xoscal-backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "Print build version and exit")
	dsn := fs.String("dsn", "oscal.db", "SQLite data source path")
	out := fs.String("out", "", "New backup destination path")
	file := fs.String("file", "", "Target backup file for verify or restore")
	verify := fs.Bool("verify", false, "Verify integrity and compute SHA-256 sidecar of backup file")
	restore := fs.Bool("restore", false, "Restore verified backup file into target -dsn")
	force := fs.Bool("force", false, "Allow overwriting target database during restore")
	sidecar := fs.Bool("sidecar", true, "Automatically generate SHA-256 checksum sidecar (.sha256)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return nil
	}

	ctx := context.Background()

	// Mode 1: Verify backup file
	if *verify {
		targetFile := *file
		if targetFile == "" {
			targetFile = *out
		}
		if targetFile == "" {
			return errors.New("-file is required for verify mode")
		}
		digest, err := dbutil.VerifySQLite(ctx, targetFile)
		if err != nil {
			return fmt.Errorf("[FAIL] Backup verification failed: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "[OK] SQLite integrity verified for %s\n", targetFile)
		_, _ = fmt.Fprintf(stdout, "[OK] Digest: %s\n", digest)

		if *sidecar {
			if _, err := dbutil.WriteDigestSidecar(targetFile); err != nil {
				return fmt.Errorf("[FAIL] Failed to write sidecar: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "[OK] Sidecar written: %s.sha256\n", targetFile)
		}
		return nil
	}

	// Mode 2: Restore backup file into target database
	if *restore {
		if *file == "" {
			return errors.New("-file is required for restore mode")
		}
		if *dsn == "" {
			return errors.New("-dsn is required for restore destination")
		}
		_, _ = fmt.Fprintf(stdout, "Restoring backup %s into %s ...\n", *file, *dsn)
		if err := dbutil.RestoreSQLite(ctx, *file, *dsn, *force); err != nil {
			return fmt.Errorf("[FAIL] Restore failed: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "[OK] Database restored successfully to %s\n", *dsn)
		return nil
	}

	// Mode 3: Create backup snapshot
	if *out == "" {
		return errors.New("-out is required for backup mode (or use -verify / -restore)")
	}
	if err := dbutil.BackupSQLite(ctx, *dsn, *out); err != nil {
		return fmt.Errorf("[FAIL] Backup failed: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "[OK] SQLite backup written to %s\n", *out)

	// Validate snapshot immediately and write sidecar
	digest, err := dbutil.VerifySQLite(ctx, *out)
	if err != nil {
		return fmt.Errorf("[FAIL] Snapshot created but failed verification: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "[OK] Snapshot verified: %s\n", digest)

	if *sidecar {
		if _, err := dbutil.WriteDigestSidecar(*out); err != nil {
			return fmt.Errorf("[FAIL] Failed to write sidecar: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "[OK] Sidecar written: %s.sha256\n", *out)
	}
	return nil
}
