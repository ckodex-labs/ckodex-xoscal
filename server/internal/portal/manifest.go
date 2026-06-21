package portal

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// Framework is one entry in data/frameworks/manifest.yaml.
type Framework struct {
	RefID        string `yaml:"ref_id"`
	Jurisdiction string `yaml:"jurisdiction"`
	Filename     string `yaml:"filename"`
}

// Manifest is the parsed framework manifest.
type Manifest struct {
	Version       string      `yaml:"version"`
	SourceBaseURL string      `yaml:"source_base_url"`
	Frameworks    []Framework `yaml:"frameworks"`
}

// ParseManifest parses the framework manifest YAML.
func ParseManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// Source derives the GitHub owner, repo, and directory path from a
// raw.githubusercontent.com source_base_url of the form
// https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path...}.
func (m *Manifest) Source() (owner, repo, path string, err error) {
	u, err := url.Parse(m.SourceBaseURL)
	if err != nil {
		return "", "", "", fmt.Errorf("parse source_base_url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("source_base_url path too short: %q", u.Path)
	}
	return parts[0], parts[1], strings.Join(parts[3:], "/"), nil
}

// RefIDs returns the ref_id of each framework in manifest order.
func (m *Manifest) RefIDs() []string {
	ids := make([]string, 0, len(m.Frameworks))
	for _, f := range m.Frameworks {
		ids = append(ids, f.RefID)
	}
	return ids
}
