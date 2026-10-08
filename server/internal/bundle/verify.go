package bundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// maxVerifiedBundleBytes bounds the entire decompressed archive, including padding.
const maxVerifiedBundleBytes int64 = 256 << 20

// VerifyBundle checks artifact and evidence bytes, sizes, and SHA-256 sidecars
// against manifest digests, and rejects ambiguous duplicate archive members.
func VerifyBundle(bundleTarGzPath string) (result *BundleManifest, resultErr error) {
	f, err := os.Open(bundleTarGzPath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, f.Close()) }()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, gr.Close()) }()
	files, err := readVerifiedFiles(gr, maxVerifiedBundleBytes)
	if err != nil {
		return nil, err
	}
	manifestBytes, ok := files["manifest.json"]
	if !ok {
		return nil, fmt.Errorf("manifest.json not found in bundle")
	}
	var manifest BundleManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	if err := verifyManifest(files, manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func readVerifiedFiles(reader io.Reader, limit int64) (map[string][]byte, error) {
	stream := &io.LimitedReader{R: reader, N: limit + 1}
	files, err := readBundleFiles(tar.NewReader(stream))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(zeroTarPadding{}, stream); err != nil {
		return nil, fmt.Errorf("validate gzip stream: %w", err)
	}
	if stream.N == 0 {
		return nil, fmt.Errorf("decompressed bundle exceeds %d bytes", limit)
	}
	return files, nil
}

type zeroTarPadding struct{}

func (zeroTarPadding) Write(data []byte) (int, error) {
	for _, value := range data {
		if value != 0 {
			return 0, fmt.Errorf("nonzero bytes after TAR end marker")
		}
	}
	return len(data), nil
}

func verifyManifest(files map[string][]byte, manifest BundleManifest) error {
	allowed := map[string]bool{"manifest.json": true, "audit-viewer.html": true}
	normalized := map[string]bool{memberKey("manifest.json"): true, memberKey("audit-viewer.html"): true}
	if err := declareMembers(files, manifest.Artifacts, "oscal", allowed, normalized); err != nil {
		return err
	}
	if err := declareMembers(files, manifest.Evidence, "evidence", allowed, normalized); err != nil {
		return err
	}
	for name := range files {
		if !allowed[name] {
			return fmt.Errorf("unlisted bundle member: %s", name)
		}
	}
	for name := range allowed {
		if _, exists := files[name]; !exists {
			return fmt.Errorf("bundle member missing: %s", name)
		}
	}
	return nil
}

func declareMembers(files map[string][]byte, digests []FileDigest, category string, allowed, normalized map[string]bool) error {
	for _, digest := range digests {
		if err := categoryMember(digest.Path, category); err != nil {
			return err
		}
		for _, name := range []string{digest.Path, digest.Path + ".sha256"} {
			if err := registerMember(normalized, name); err != nil {
				return err
			}
			allowed[name] = true
		}
		if err := verifyDigest(files, digest); err != nil {
			return err
		}
	}
	return nil
}

func readBundleFiles(reader *tar.Reader) (map[string][]byte, error) {
	files := make(map[string][]byte)
	normalized := make(map[string]bool)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if err := registerMember(normalized, header.Name); err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("non-regular bundle member: %s", header.Name)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", header.Name, err)
		}
		files[header.Name] = data
	}
}

func verifyDigest(files map[string][]byte, digest FileDigest) error {
	data, exists := files[digest.Path]
	if !exists {
		return fmt.Errorf("bundle member missing: %s", digest.Path)
	}
	if int64(len(data)) != digest.Size {
		return fmt.Errorf("size mismatch on %s", digest.Path)
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, digest.SHA256) {
		return fmt.Errorf("digest mismatch on %s", digest.Path)
	}
	sidecar, exists := files[digest.Path+".sha256"]
	if !exists || !strings.EqualFold(strings.TrimSpace(string(sidecar)), actual) {
		return fmt.Errorf("sidecar mismatch or missing on %s", digest.Path)
	}
	return nil
}
