package frameworks

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

//go:embed embedded_catalogs.tar.gz
var embeddedCatalogsArchive []byte

const maxEmbeddedCatalogBytes = 32 << 20 // 32 MiB bounded decompression safety cap per catalog entry

type manifestFrameworkSpec struct {
	ID           string   `json:"id"`
	Aliases      []string `json:"aliases"`
	Title        string   `json:"title"`
	Version      string   `json:"version"`
	Jurisdiction string   `json:"jurisdiction"`
	Authority    string   `json:"authority"`
	ControlCount int      `json:"control_count"`
}

var (
	embeddedCatalogsOnce  sync.Once
	embeddedCatalogsMap   map[string][]byte
	embeddedManifestSpecs []manifestFrameworkSpec
	embeddedCatalogsErr   error
)

func initEmbeddedCatalogs() {
	embeddedCatalogsOnce.Do(func() {
		if len(embeddedCatalogsArchive) == 0 {
			embeddedCatalogsErr = errors.New("embedded_catalogs.tar.gz is empty")
			return
		}
		gz, err := gzip.NewReader(bytes.NewReader(embeddedCatalogsArchive))
		if err != nil {
			embeddedCatalogsErr = fmt.Errorf("open embedded catalogs gzip: %w", err)
			return
		}
		defer func() { _ = gz.Close() }()

		tr := tar.NewReader(gz)
		idx := make(map[string][]byte)
		var specs []manifestFrameworkSpec
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				embeddedCatalogsErr = fmt.Errorf("read embedded catalogs tar: %w", err)
				return
			}
			if hdr.Typeflag != tar.TypeReg {
				continue
			}
			name := strings.TrimPrefix(hdr.Name, "./")
			data, err := io.ReadAll(io.LimitReader(tr, maxEmbeddedCatalogBytes))
			if err != nil {
				embeddedCatalogsErr = fmt.Errorf("read embedded entry %s: %w", name, err)
				return
			}
			if name == "index.json" {
				if err := json.Unmarshal(data, &specs); err != nil {
					embeddedCatalogsErr = fmt.Errorf("decode embedded index.json: %w", err)
					return
				}
				continue
			}
			if !strings.HasSuffix(name, "/catalog.json") {
				continue
			}
			id := strings.TrimSuffix(name, "/catalog.json")
			if id == "" || strings.Contains(id, "/") {
				continue
			}
			idx[id] = data
		}
		if len(specs) == 0 {
			embeddedCatalogsErr = errors.New("embedded_catalogs.tar.gz missing index.json entries")
			return
		}
		embeddedCatalogsMap = idx
		embeddedManifestSpecs = specs
	})
}

func loadEmbeddedManifestSpecs() []manifestFrameworkSpec {
	initEmbeddedCatalogs()
	if embeddedCatalogsErr != nil || len(embeddedManifestSpecs) == 0 {
		return nil
	}
	out := make([]manifestFrameworkSpec, len(embeddedManifestSpecs))
	for i, spec := range embeddedManifestSpecs {
		cp := spec
		if len(spec.Aliases) > 0 {
			cp.Aliases = append([]string(nil), spec.Aliases...)
		}
		out[i] = cp
	}
	return out
}

// ManifestFrameworkIDs returns a sorted copy of all manifest framework IDs
// bundled in the embedded catalog archive.
func ManifestFrameworkIDs() []string {
	specs := loadEmbeddedManifestSpecs()
	ids := make([]string, len(specs))
	for i, spec := range specs {
		ids[i] = spec.ID
	}
	sort.Strings(ids)
	return ids
}

// loadEmbeddedCatalog returns a defensive copy or read-only slice of the embedded
// OSCAL catalog.json bytes for a manifest framework ID.
func loadEmbeddedCatalog(id string) ([]byte, error) {
	initEmbeddedCatalogs()
	if embeddedCatalogsErr != nil {
		return nil, embeddedCatalogsErr
	}
	norm := strings.TrimSpace(id)
	data, ok := embeddedCatalogsMap[norm]
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("embedded catalog.json not found for framework %s", id)
	}
	return data, nil
}
