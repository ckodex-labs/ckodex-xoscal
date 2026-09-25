// Command xoscal-backup manages SQLite database snapshots, integrity verification,
// and disaster recovery restores for xOSCAL deployments.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func main() {
	dsn := flag.String("dsn", "oscal.db", "SQLite data source path")
	out := flag.String("out", "", "New backup destination path")
	file := flag.String("file", "", "Target backup file for verify or restore")
	verify := flag.Bool("verify", false, "Verify integrity and compute SHA-256 sidecar of backup file")
	restore := flag.Bool("restore", false, "Restore verified backup file into target -dsn")
	force := flag.Bool("force", false, "Allow overwriting target database during restore")
	sidecar := flag.Bool("sidecar", true, "Automatically generate SHA-256 checksum sidecar (.sha256)")
	flag.Parse()

	ctx := context.Background()

	// Mode 1: Verify backup file
	if *verify {
		targetFile := *file
		if targetFile == "" {
			targetFile = *out
		}
		if targetFile == "" {
			log.Fatal("-file is required for verify mode")
		}
		digest, err := dbutil.VerifySQLite(ctx, targetFile)
		if err != nil {
			log.Fatalf("[FAIL] Backup verification failed: %v", err)
		}
		fmt.Printf("[OK] SQLite integrity verified for %s\n", targetFile)
		fmt.Printf("[OK] Digest: %s\n", digest)

		if *sidecar {
			if _, err := dbutil.WriteDigestSidecar(targetFile); err != nil {
				log.Fatalf("[FAIL] Failed to write sidecar: %v", err)
			}
			fmt.Printf("[OK] Sidecar written: %s.sha256\n", targetFile)
		}
		return
	}

	// Mode 2: Restore backup file into target database
	if *restore {
		if *file == "" {
			log.Fatal("-file is required for restore mode")
		}
		if *dsn == "" {
			log.Fatal("-dsn is required for restore destination")
		}
		fmt.Printf("Restoring backup %s into %s ...\n", *file, *dsn)
		if err := dbutil.RestoreSQLite(ctx, *file, *dsn, *force); err != nil {
			log.Fatalf("[FAIL] Restore failed: %v", err)
		}
		fmt.Printf("[OK] Database restored successfully to %s\n", *dsn)
		return
	}

	// Mode 3: Create backup snapshot
	if *out == "" {
		log.Fatal("-out is required for backup mode (or use -verify / -restore)")
	}
	if err := dbutil.BackupSQLite(ctx, *dsn, *out); err != nil {
		log.Fatalf("[FAIL] Backup failed: %v", err)
	}
	fmt.Printf("[OK] SQLite backup written to %s\n", *out)

	// Validate snapshot immediately and write sidecar
	digest, err := dbutil.VerifySQLite(ctx, *out)
	if err != nil {
		log.Fatalf("[FAIL] Snapshot created but failed verification: %v", err)
	}
	fmt.Printf("[OK] Snapshot verified: %s\n", digest)

	if *sidecar {
		if _, err := dbutil.WriteDigestSidecar(*out); err != nil {
			log.Fatalf("[FAIL] Failed to write sidecar: %v", err)
		}
		fmt.Printf("[OK] Sidecar written: %s.sha256\n", *out)
	}
}
