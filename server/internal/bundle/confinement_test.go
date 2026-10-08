package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleEvidenceBoundaryAndFailurePublication(t *testing.T) {
	for _, kind := range []string{"absolute", "relative", "chain", "broken", "missing-root"} {
		t.Run(kind, func(t *testing.T) {
			parent := t.TempDir()
			selected := filepath.Join(parent, "evidence")
			outside := filepath.Join(parent, "evidence-sibling")
			for _, dir := range []string{selected, outside} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			mustBundleWrite(t, filepath.Join(outside, "secret"), []byte("outside sentinel"))
			mustBundleWrite(t, filepath.Join(selected, "valid"), []byte("inside evidence"))
			target := filepath.Join(outside, "secret")
			if kind == "relative" {
				target = "../evidence-sibling/secret"
			}
			if kind == "broken" {
				target = "missing"
			}
			if kind == "chain" {
				mustBundleLink(t, "../evidence-sibling/secret", filepath.Join(selected, "bridge"))
				target = "bridge"
			}
			mustBundleLink(t, target, filepath.Join(selected, "secret"))
			if kind == "missing-root" {
				selected = filepath.Join(parent, "missing")
			}
			out := filepath.Join(parent, "bundle.tar.gz")
			mustBundleWrite(t, out, []byte("existing archive"))
			if err := CreateAuditBundle(nil, selected, out); err == nil {
				t.Fatal("invalid evidence accepted")
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "existing archive" || strings.Contains(string(data), "outside sentinel") {
				t.Fatalf("failed publication replaced archive: %q %v", data, err)
			}
			entries, err := os.ReadDir(parent)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".xoscal-bundle-") {
					t.Fatal("private failed bundle left behind")
				}
			}
		})
	}
}

func TestBundleInsideLinkArchiveContents(t *testing.T) {
	parent := t.TempDir()
	selected := filepath.Join(parent, "selected")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	mustBundleWrite(t, filepath.Join(selected, "payload"), []byte("inside evidence"))
	mustBundleWrite(t, filepath.Join(parent, "outside"), []byte("outside sentinel"))
	mustBundleLink(t, "payload", filepath.Join(selected, "evidence.log"))
	mustBundleLink(t, selected, filepath.Join(parent, "root-link"))
	out := filepath.Join(parent, "bundle.tar.gz")
	if err := CreateAuditBundle(nil, filepath.Join(parent, "root-link"), out); err != nil {
		t.Fatal(err)
	}
	files := inspectBundle(t, out)
	if string(files["evidence/evidence.log"]) != "inside evidence" {
		t.Fatal("relative inside link omitted")
	}
	for name, data := range files {
		if strings.Contains(string(data), "outside sentinel") {
			t.Fatalf("outside bytes in archive member %s", name)
		}
	}
	if _, err := VerifyBundle(out); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBundleRejectsEvidenceIntegrityFailures(t *testing.T) {
	parent := t.TempDir()
	mustBundleWrite(t, filepath.Join(parent, "blob"), []byte("evidence bytes"))
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := CreateAuditBundle(nil, parent, out); err != nil {
		t.Fatal(err)
	}
	original := inspectBundle(t, out)
	for _, scenario := range []string{"evidence", "sidecar", "missing-sidecar", "size", "duplicate", "gzip-crc"} {
		t.Run(scenario, func(t *testing.T) {
			files := make(map[string][]byte)
			for name, data := range original {
				files[name] = append([]byte(nil), data...)
			}
			mutateBundle(t, files, scenario)
			path := filepath.Join(t.TempDir(), "changed.tar.gz")
			writeTestBundle(t, path, files, scenario == "duplicate")
			if scenario == "gzip-crc" {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data[len(data)-8] ^= 0xff
				mustBundleWrite(t, path, data)
			}
			if _, err := VerifyBundle(path); err == nil {
				t.Fatal("corrupted bundle verified")
			}
		})
	}
}

func mutateBundle(t *testing.T, files map[string][]byte, scenario string) {
	t.Helper()
	switch scenario {
	case "evidence":
		files["evidence/blob"] = []byte("tampered bytes")
	case "sidecar":
		files["evidence/blob.sha256"] = []byte(strings.Repeat("0", 64))
	case "missing-sidecar":
		delete(files, "evidence/blob.sha256")
	case "size":
		var manifest BundleManifest
		if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
			t.Fatal(err)
		}
		manifest.Evidence[0].Size++
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		files["manifest.json"] = data
	}
}

func inspectBundle(t *testing.T, name string) map[string][]byte {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	files := make(map[string][]byte)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = data
	}
}

func writeTestBundle(t *testing.T, name string, files map[string][]byte, duplicate bool) {
	t.Helper()
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, data := range files {
		if err := writeTarEntry(tw, name, data); err != nil {
			t.Fatal(err)
		}
	}
	if duplicate {
		if err := writeTarEntry(tw, "evidence/blob", files["evidence/blob"]); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustBundleWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func mustBundleLink(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBundleDecompressedByteLimit(t *testing.T) {
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	if err := writeTarEntry(writer, "blob", bytes.Repeat([]byte("a"), 4096)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readVerifiedFiles(bytes.NewReader(output.Bytes()), 1024); err == nil {
		t.Fatal("oversized decompressed stream accepted")
	}
	if _, err := readVerifiedFiles(bytes.NewReader(output.Bytes()), int64(output.Len())); err != nil {
		t.Fatal(err)
	}
}
