package frameworks

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func init() {
	RegisterManifestFrameworks(DefaultRegistry())
}

// RegisterManifestFrameworks registers all non-Canadian manifest frameworks
// declared in the embedded manifest index into the provided registry.
func RegisterManifestFrameworks(r *Registry) {
	specs := loadEmbeddedManifestSpecs()
	for _, spec := range specs {
		s := spec
		var mu sync.Mutex
		var cached []byte

		loadCatalog := func() ([]byte, error) {
			if envDir := os.Getenv("XOSCAL_DATA_DIR"); envDir != "" {
				envCandidates := []string{
					filepath.Join(envDir, s.ID, "catalog.json"),
					filepath.Join(envDir, "frameworks", s.ID, "catalog.json"),
				}
				for _, p := range envCandidates {
					// #nosec G304,G703 -- path is derived from operator XOSCAL_DATA_DIR and embedded manifest index.
					if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
						return data, nil
					}
				}
				return nil, fmt.Errorf("catalog.json not found in XOSCAL_DATA_DIR for framework %s", s.ID)
			}

			mu.Lock()
			defer mu.Unlock()
			if len(cached) > 0 {
				return cached, nil
			}

			candidates := []string{
				filepath.Join("data", "frameworks", s.ID, "catalog.json"),
				filepath.Join("/data", "frameworks", s.ID, "catalog.json"),
				filepath.Join("/usr/share/xoscal/data/frameworks", s.ID, "catalog.json"),
			}
			if exePath, err := os.Executable(); err == nil && exePath != "" {
				exeDir := filepath.Dir(exePath)
				candidates = append(candidates,
					filepath.Join(exeDir, "data", "frameworks", s.ID, "catalog.json"),
					filepath.Join(exeDir, "..", "data", "frameworks", s.ID, "catalog.json"),
					filepath.Join(exeDir, "..", "share", "xoscal", "data", "frameworks", s.ID, "catalog.json"),
				)
			}

			for _, p := range candidates {
				// #nosec G304,G703 -- path is derived from the embedded manifest index.
				if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
					cached = data
					return cached, nil
				}
			}

			embData, err := loadEmbeddedCatalog(s.ID)
			if err != nil {
				return nil, fmt.Errorf("catalog.json not found for framework %s: %w", s.ID, err)
			}
			cached = embData
			return cached, nil
		}

		_ = r.Register(New(Config{
			ID:           s.ID,
			Aliases:      s.Aliases,
			Title:        s.Title,
			Version:      s.Version,
			Jurisdiction: s.Jurisdiction,
			Authority:    s.Authority,
			ControlCount: s.ControlCount,
			HasProfile:   false,
			CatalogFn:    loadCatalog,
		}))
	}
}
