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
	// #nosec G304 -- in is the explicit artifact manifest path selected by the CLI operator.
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
	// #nosec G304,G306 -- out is the explicit provenance destination selected by the CLI operator.
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
