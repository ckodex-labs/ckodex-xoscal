// Command xoscal-backup creates a consistent, non-overwriting SQLite snapshot.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func main() {
	dsn := flag.String("dsn", "oscal.db", "SQLite data source")
	out := flag.String("out", "", "new backup destination path")
	flag.Parse()
	if *out == "" {
		log.Fatal("-out is required")
	}
	if err := dbutil.BackupSQLite(context.Background(), *dsn, *out); err != nil {
		log.Fatalf("backup failed: %v", err)
	}
	log.Printf("SQLite backup written to %s", *out)
}
