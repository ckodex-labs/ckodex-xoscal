// Package openapiembed embeds the generated OpenAPI v3 specifications for xOSCAL
// services so distroless and -trimpath binaries serve /openapi/*.openapi.yaml
// without relying on working-directory filesystem paths.
package openapiembed

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed specs/*.openapi.yaml
var specsFS embed.FS

var specAliases = map[string]string{
	"governance.yaml":            "governance_service.openapi.yaml",
	"oscal.yaml":                 "oscal_service.openapi.yaml",
	"transparency_exchange.yaml": "transparency_exchange_service.openapi.yaml",
	"transparency_graph.yaml":    "transparency_graph_service.openapi.yaml",
}

// ReadSpec returns the raw OpenAPI YAML bytes for the named spec (e.g.
// "governance_service.openapi.yaml" or short alias "governance.yaml").
func ReadSpec(name string) ([]byte, error) {
	cleaned := strings.TrimSpace(name)
	cleaned = strings.TrimPrefix(cleaned, "/openapi/")
	cleaned = strings.TrimPrefix(cleaned, "specs/")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || strings.Contains(cleaned, "/") || strings.Contains(cleaned, "\\") || strings.Contains(cleaned, "..") || strings.ContainsRune(cleaned, '\x00') {
		return nil, fmt.Errorf("invalid OpenAPI spec name %q: %w", name, fs.ErrNotExist)
	}
	if canonical, ok := specAliases[cleaned]; ok {
		cleaned = canonical
	}
	data, err := specsFS.ReadFile("specs/" + cleaned)
	if err != nil {
		return nil, fmt.Errorf("read embedded OpenAPI spec %q: %w", cleaned, err)
	}
	return data, nil
}

// ListSpecs returns the sorted list of embedded *.openapi.yaml filenames.
func ListSpecs() []string {
	entries, err := fs.ReadDir(specsFS, "specs")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".openapi.yaml") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}
