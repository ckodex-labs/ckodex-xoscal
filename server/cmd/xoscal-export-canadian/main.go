// Command xoscal-export-canadian exports authoritative Canadian cybersecurity frameworks
// (CCCS ITSG-33, CCCS PBMM Cloud Profile, CyberSecure Canada, CCCS ITSP.10.171) into
// schema-validated OSCAL 1.2.3 JSON artifacts with SHA-256 sidecars.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
)

func main() {
	outDir := flag.String("out", "data/frameworks", "Output directory for Canadian framework artifacts")
	flag.Parse()

	if err := canadianframeworks.ExportAll(*outDir); err != nil {
		log.Fatalf("Export Canadian frameworks: %v", err)
	}

	fmt.Printf("Successfully exported Canadian cybersecurity frameworks to %s\n", *outDir)
}
