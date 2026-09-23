package bundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/oscalversion"
)

// FileDigest records a relative file path and its SHA-256 checksum.
type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// BundleManifest describes the contents and cryptographic receipts of an audit bundle.
type BundleManifest struct {
	CreatedAt     time.Time    `json:"created_at"`
	BundleVersion string       `json:"bundle_version"`
	OSCALVersion  string       `json:"oscal_version"`
	Artifacts     []FileDigest `json:"artifacts"`
	Evidence      []FileDigest `json:"evidence"`
}

// CreateAuditBundle packages OSCAL JSON artifacts, evidence blobs, and the standalone
// offline audit-viewer.html into a self-contained .tar.gz archive.
func CreateAuditBundle(artifactPaths []string, evidenceDir string, outTarGzPath string) error {
	outDir := filepath.Dir(outTarGzPath)
	if err := os.MkdirAll(outDir, 0750); err != nil {
		return fmt.Errorf("create bundle output directory: %w", err)
	}

	outFile, err := os.Create(outTarGzPath)
	if err != nil {
		return fmt.Errorf("create bundle output file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	manifest := BundleManifest{
		CreatedAt:     time.Now().UTC(),
		BundleVersion: "1.0.0",
		OSCALVersion:  oscalversion.Current(),
		Artifacts:     make([]FileDigest, 0),
		Evidence:      make([]FileDigest, 0),
	}

	// 1. Write audit-viewer.html
	viewerBytes := []byte(AuditViewerHTML)
	if err := writeTarEntry(tw, "audit-viewer.html", viewerBytes); err != nil {
		return fmt.Errorf("write viewer entry: %w", err)
	}

	// 2. Add OSCAL artifacts
	for _, artPath := range artifactPaths {
		data, err := os.ReadFile(artPath)
		if err != nil {
			return fmt.Errorf("read artifact %s: %w", artPath, err)
		}
		base := filepath.Base(artPath)
		sum := sha256.Sum256(data)
		hashStr := hex.EncodeToString(sum[:])

		entryPath := filepath.Join("oscal", base)
		if err := writeTarEntry(tw, entryPath, data); err != nil {
			return fmt.Errorf("write artifact entry %s: %w", entryPath, err)
		}

		// Write SHA-256 sidecar
		if err := writeTarEntry(tw, entryPath+".sha256", []byte(hashStr+"\n")); err != nil {
			return fmt.Errorf("write sidecar %s: %w", entryPath, err)
		}

		manifest.Artifacts = append(manifest.Artifacts, FileDigest{
			Path:   entryPath,
			SHA256: hashStr,
			Size:   int64(len(data)),
		})
	}

	// 3. Add Evidence if present
	if evidenceDir != "" {
		entries, err := os.ReadDir(evidenceDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || strings.HasSuffix(e.Name(), ".sha256") {
					continue
				}
				evPath := filepath.Join(evidenceDir, e.Name())
				data, err := os.ReadFile(evPath)
				if err != nil {
					continue
				}
				sum := sha256.Sum256(data)
				hashStr := hex.EncodeToString(sum[:])

				entryPath := filepath.Join("evidence", e.Name())
				if err := writeTarEntry(tw, entryPath, data); err != nil {
					return fmt.Errorf("write evidence entry %s: %w", entryPath, err)
				}
				if err := writeTarEntry(tw, entryPath+".sha256", []byte(hashStr+"\n")); err != nil {
					return fmt.Errorf("write evidence sidecar %s: %w", entryPath, err)
				}

				manifest.Evidence = append(manifest.Evidence, FileDigest{
					Path:   entryPath,
					SHA256: hashStr,
					Size:   int64(len(data)),
				})
			}
		}
	}

	// 4. Write manifest.json
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := writeTarEntry(tw, "manifest.json", manifestBytes); err != nil {
		return fmt.Errorf("write manifest entry: %w", err)
	}

	return nil
}

func writeTarEntry(tw *tar.Writer, name string, content []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(content)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(content)
	return err
}

// VerifyBundle unpacks and checks all SHA-256 sidecars and manifest digests.
func VerifyBundle(bundleTarGzPath string) (*BundleManifest, error) {
	f, err := os.Open(bundleTarGzPath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", hdr.Name, err)
		}
		files[hdr.Name] = data
	}

	manifestBytes, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("manifest.json not found in bundle")
	}

	var m BundleManifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	// Verify all artifact checksums
	for _, art := range m.Artifacts {
		data, ok := files[art.Path]
		if !ok {
			return nil, fmt.Errorf("artifact missing: %s", art.Path)
		}
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if !strings.EqualFold(actual, art.SHA256) {
			return nil, fmt.Errorf("digest mismatch on %s: expected %s, got %s", art.Path, art.SHA256, actual)
		}
	}

	return &m, nil
}
