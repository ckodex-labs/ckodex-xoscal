// Package specregistry holds the OSCAL spec integrity registry: per-model
// upstream schema hashes bound to the protos that implement them. Pure logic;
// the CLI wraps it with network I/O.
package specregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Model binds one OSCAL model's upstream schema (and its hash) to a proto file.
type Model struct {
	Model        string `yaml:"model" json:"model"`
	SchemaAsset  string `yaml:"schema_asset" json:"schema_asset"`
	SchemaSHA256 string `yaml:"schema_sha256" json:"schema_sha256"`
	Proto        string `yaml:"proto" json:"proto"`
}

// Registry is the committed per-model integrity ledger for one OSCAL version.
type Registry struct {
	Version      string  `yaml:"version"`
	OSCALVersion string  `yaml:"oscal_version"`
	GeneratedAt  string  `yaml:"generated_at"`
	Models       []Model `yaml:"models"`
}

// Parse decodes a registry from YAML.
func Parse(b []byte) (*Registry, error) {
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &r, nil
}

// Serialize encodes the registry to YAML.
func (r *Registry) Serialize() ([]byte, error) {
	b, err := yaml.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("serialize registry: %w", err)
	}
	return b, nil
}

// IsPopulated reports whether every model has a non-empty schema hash. An
// unpopulated registry cannot verify lock-step (nothing is pinned yet).
func (r *Registry) IsPopulated() bool {
	for _, m := range r.Models {
		if m.SchemaSHA256 == "" {
			return false
		}
	}
	return len(r.Models) > 0
}

// SchemaURL builds the GitHub release-asset URL for an OSCAL model schema.
func SchemaURL(oscalVersion, asset string) string {
	return fmt.Sprintf("https://github.com/usnistgov/OSCAL/releases/download/v%s/%s", oscalVersion, asset)
}

// Hash returns the sha256 of data in the repo's "sha256:<hex>" format.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
