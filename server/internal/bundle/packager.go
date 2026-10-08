package bundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/oscalversion"
	"github.com/mchorfa/xoscal/server/internal/rootfs"
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
	return createAuditBundle(artifactPaths, evidenceDir, outTarGzPath, maxVerifiedBundleBytes)
}

func createAuditBundle(artifactPaths []string, evidenceDir string, outTarGzPath string, byteLimit int64) (result error) {
	var root *os.Root
	if evidenceDir != "" {
		var err error
		root, err = os.OpenRoot(evidenceDir)
		if err != nil {
			return fmt.Errorf("open evidence directory: %w", err)
		}
		defer func() { result = errors.Join(result, root.Close()) }()
	}
	outDir := filepath.Dir(outTarGzPath)
	if err := os.MkdirAll(outDir, 0750); err != nil {
		return fmt.Errorf("create bundle output directory: %w", err)
	}
	outFile, err := os.CreateTemp(outDir, ".xoscal-bundle-*.tmp")
	if err != nil {
		return fmt.Errorf("create private bundle output: %w", err)
	}
	defer func() { _ = os.Remove(outFile.Name()) }()
	gw := gzip.NewWriter(outFile)
	tw := tar.NewWriter(&boundedBundleWriter{writer: gw, remaining: byteLimit})
	err = buildBundle(tw, artifactPaths, root)
	err = errors.Join(err, tw.Close(), gw.Close(), outFile.Close())
	if err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}
	if err := os.Rename(outFile.Name(), outTarGzPath); err != nil {
		return fmt.Errorf("publish bundle: %w", err)
	}
	return nil
}

type bundleBuilder struct {
	tar      *tar.Writer
	manifest BundleManifest
	members  map[string]bool
}

func buildBundle(tw *tar.Writer, artifactPaths []string, root *os.Root) error {
	builder := &bundleBuilder{tar: tw, members: make(map[string]bool), manifest: BundleManifest{
		CreatedAt: time.Now().UTC(), BundleVersion: "1.0.0", OSCALVersion: oscalversion.Current(),
		Artifacts: make([]FileDigest, 0), Evidence: make([]FileDigest, 0),
	}}
	if err := writeTarEntry(tw, "audit-viewer.html", []byte(AuditViewerHTML)); err != nil {
		return err
	}
	for _, artPath := range artifactPaths {
		data, err := os.ReadFile(artPath)
		if err != nil {
			return fmt.Errorf("read artifact %s: %w", artPath, err)
		}
		if err := builder.add("oscal/"+filepath.Base(artPath), data, &builder.manifest.Artifacts); err != nil {
			return err
		}
	}
	if root != nil {
		if err := builder.addEvidence(root); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(builder.manifest, "", "  ")
	if err != nil {
		return err
	}
	return writeTarEntry(tw, "manifest.json", data)
}

func (builder *bundleBuilder) add(name string, data []byte, digests *[]FileDigest) error {
	if err := categoryMember(name, strings.SplitN(name, "/", 2)[0]); err != nil {
		return err
	}
	if err := registerMember(builder.members, name); err != nil {
		return err
	}
	if err := registerMember(builder.members, name+".sha256"); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if err := writeTarEntry(builder.tar, name, data); err != nil {
		return err
	}
	if err := writeTarEntry(builder.tar, name+".sha256", []byte(digest+"\n")); err != nil {
		return err
	}
	*digests = append(*digests, FileDigest{Path: name, SHA256: digest, Size: int64(len(data))})
	return nil
}

func (builder *bundleBuilder) addEvidence(root *os.Root) error {
	entries, err := rootfs.ReadDir(root, ".")
	if err != nil {
		return fmt.Errorf("enumerate evidence: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".sha256") {
			continue
		}
		info, err := root.Stat(entry.Name())
		if err != nil {
			return fmt.Errorf("inspect evidence %s: %w", entry.Name(), err)
		}
		if info.IsDir() {
			continue
		}
		data, err := rootfs.ReadFile(root, entry.Name())
		if err != nil {
			return fmt.Errorf("read evidence %s: %w", entry.Name(), err)
		}
		if err := builder.add("evidence/"+entry.Name(), data, &builder.manifest.Evidence); err != nil {
			return err
		}
	}
	return nil
}

func writeTarEntry(tw *tar.Writer, name string, content []byte) error {
	if err := canonicalMember(name); err != nil {
		return err
	}
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
