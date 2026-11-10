package frameworks

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

//go:embed embedded_catalogs.tar.gz
var embeddedCatalogsArchive []byte

const maxEmbeddedCatalogBytes = 32 << 20 // 32 MiB bounded decompression safety cap per catalog entry

var (
	embeddedCatalogsOnce sync.Once
	embeddedCatalogsMap  map[string][]byte
	embeddedCatalogsErr  error
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
		idx := make(map[string][]byte, len(manifestFrameworkSpecs))
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
			if !strings.HasSuffix(name, "/catalog.json") {
				continue
			}
			id := strings.TrimSuffix(name, "/catalog.json")
			if id == "" || strings.Contains(id, "/") {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(tr, maxEmbeddedCatalogBytes))
			if err != nil {
				embeddedCatalogsErr = fmt.Errorf("read embedded catalog %s: %w", name, err)
				return
			}
			idx[id] = data
		}
		embeddedCatalogsMap = idx
	})
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
