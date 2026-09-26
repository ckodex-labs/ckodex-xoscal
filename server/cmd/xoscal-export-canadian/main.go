// Command xoscal-export-canadian exports authoritative Canadian cybersecurity frameworks
// (CCCS ITSG-33, CCCS PBMM Cloud Profile, CyberSecure Canada, CCCS ITSP.10.171) into
// schema-validated OSCAL 1.2.3 JSON artifacts with SHA-256 sidecars.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/canadianframeworks"
)

func main() {
	outDir := flag.String("out", "data/frameworks", "Output directory for Canadian framework artifacts")
	framework := flag.String("framework", "", "Optional specific framework ref_id to export (default: all)")
	list := flag.Bool("list", false, "List available Canadian frameworks and exit")
	flag.Parse()

	if *list {
		fmt.Println("Authoritative Canadian Cybersecurity Frameworks:")
		for _, fw := range canadianframeworks.ListFrameworks() {
			profStr := ""
			if fw.HasProfile {
				profStr = " (+ Profile)"
			}
			fmt.Printf("  - %-25s [%3d controls%s] %s (%s)\n",
				fw.RefID, fw.ControlCount, profStr, fw.Name, fw.Version)
		}
		return
	}

	if *framework != "" {
		if !canadianframeworks.IsCanadian(*framework) {
			log.Fatalf("Unknown Canadian framework: %s", *framework)
		}
		if err := canadianframeworks.Export(*outDir, *framework); err != nil {
			log.Fatalf("Export framework %s: %v", *framework, err)
		}
		fmt.Printf("Successfully exported %s to %s\n", *framework, *outDir)
		return
	}

	if err := canadianframeworks.ExportAll(*outDir); err != nil {
		log.Fatalf("Export Canadian frameworks: %v", err)
	}

	fmt.Printf("Successfully exported Canadian cybersecurity frameworks to %s\n", *outDir)
	_ = os.Stdout.Sync()
}
