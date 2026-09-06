package conformance

import (
	_ "embed"
	"testing"

	"gopkg.in/yaml.v3"
)

//go:embed graph.yaml
var graphVectorsYAML []byte

type graphVectorManifest struct {
	SchemaVersion string `yaml:"schema_version"`
	Suite         string `yaml:"suite"`
	Vectors       []struct {
		ID           string `yaml:"id"`
		Name         string `yaml:"name"`
		Requirement  string `yaml:"requirement"`
		Verification string `yaml:"verification"`
	} `yaml:"vectors"`
}

func TestGraphVectorManifest(t *testing.T) {
	var manifest graphVectorManifest
	if err := yaml.Unmarshal(graphVectorsYAML, &manifest); err != nil {
		t.Fatalf("decode graph conformance manifest: %v", err)
	}
	if manifest.SchemaVersion != "graph-conformance-v1" || manifest.Suite != "transparency-graph" {
		t.Fatalf("unexpected graph conformance manifest header: %+v", manifest)
	}
	if len(manifest.Vectors) != 5 {
		t.Fatalf("vector count = %d, want 5", len(manifest.Vectors))
	}
	seen := make(map[string]struct{}, len(manifest.Vectors))
	for _, vector := range manifest.Vectors {
		if vector.ID == "" || vector.Name == "" || vector.Requirement == "" || vector.Verification == "" {
			t.Fatalf("incomplete graph vector: %+v", vector)
		}
		if _, ok := seen[vector.ID]; ok {
			t.Fatalf("duplicate graph vector id %q", vector.ID)
		}
		seen[vector.ID] = struct{}{}
	}
}
