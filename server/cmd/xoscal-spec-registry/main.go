// Command xoscal-spec-registry keeps the protos lock-step with the OSCAL specs.
//
//	-mode populate : fetch each model's schema for the pinned version, hash it,
//	                 write schema_sha256 + generated_at into the registry.
//	-mode verify   : re-fetch and compare to the registry; print a JSON report.
//	                 exit 0 = all match, 3 = drift, 1 = operational error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mchorfa/xoscal/server/internal/specregistry"
)

func main() {
	mode := flag.String("mode", "", "populate | verify")
	registryPath := flag.String("registry", "data/oscal/spec-registry.yaml", "Registry path")
	version := flag.String("version", "", "OSCAL version (default: registry's oscal_version)")
	flag.Parse()

	code, err := run(*mode, *registryPath, *version)
	if err != nil {
		log.Print(err)
	}
	os.Exit(code)
}

func run(mode, registryPath, version string) (int, error) {
	// #nosec G304 -- registryPath is the explicit registry path selected by the CLI operator.
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return 1, fmt.Errorf("read registry: %w", err)
	}
	reg, err := specregistry.Parse(raw)
	if err != nil {
		return 1, err
	}
	if version == "" {
		version = reg.OSCALVersion
	}

	switch mode {
	case "populate":
		return populate(reg, registryPath, version)
	case "verify":
		return verify(reg, version)
	default:
		return 1, fmt.Errorf("invalid -mode %q (want populate|verify)", mode)
	}
}

func populate(reg *specregistry.Registry, path, version string) (int, error) {
	for i := range reg.Models {
		data, err := fetch(specregistry.SchemaURL(version, reg.Models[i].SchemaAsset))
		if err != nil {
			return 1, fmt.Errorf("populate %s: %w", reg.Models[i].Model, err)
		}
		reg.Models[i].SchemaSHA256 = specregistry.Hash(data)
	}
	reg.OSCALVersion = version
	reg.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := reg.Serialize()
	if err != nil {
		return 1, err
	}
	// #nosec G304,G306 -- path is the explicit registry destination selected by the CLI operator.
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return 1, fmt.Errorf("write registry: %w", err)
	}
	fmt.Printf("populated %d models for OSCAL v%s\n", len(reg.Models), version)
	return 0, nil
}

func verify(reg *specregistry.Registry, version string) (int, error) {
	if !reg.IsPopulated() {
		fmt.Fprintln(os.Stderr, "spec-registry not populated (empty schema_sha256); run -mode populate to pin hashes — gate inert until then")
		return 0, nil
	}
	var statuses []specregistry.ModelStatus
	for _, m := range reg.Models {
		data, err := fetch(specregistry.SchemaURL(version, m.SchemaAsset))
		if err != nil {
			return 1, fmt.Errorf("verify %s: %w", m.Model, err)
		}
		statuses = append(statuses, specregistry.CompareModel(m.Model, m.SchemaSHA256, data))
	}
	result := specregistry.Aggregate(version, statuses)
	report, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return 1, fmt.Errorf("marshal report: %w", err)
	}
	fmt.Println(string(report))
	return result.ExitCode(), nil
}

func fetch(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: status %d (asset missing or renamed)", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}
	return body, nil
}
