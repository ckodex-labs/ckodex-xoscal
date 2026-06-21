// Command xoscal-provenance renders a provenance.json from an artifacts JSON file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/portal"
)

func main() {
	in := flag.String("in", "", "Input artifacts JSON (array of {name,digest,...})")
	out := flag.String("out", "provenance.json", "Output path")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		log.Fatal(err)
	}
}

func run(in, out string) error {
	raw, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("read artifacts: %w", err)
	}
	var arts []portal.Artifact
	if err := json.Unmarshal(raw, &arts); err != nil {
		return fmt.Errorf("parse artifacts: %w", err)
	}
	data, err := portal.BuildManifest(arts)
	if err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
