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
	"unicode/utf16"
)

func TestVerifyBundleRejectsPathAliasesAndStrayMembers(t *testing.T) {
	for _, name := range []string{"./evidence/blob", "evidence//blob", "evidence/../evidence/blob", `evidence\blob`, "evidence/BLOB", "stray", "evidence/blob.sha256.sha256", "/evidence/blob", "evidence/blob.", "evidence/NUL"} {
		t.Run(name, func(t *testing.T) {
			files := validBundleFiles(t)
			files[name] = []byte("unverified overwrite")
			archive := filepath.Join(t.TempDir(), "alias.tar.gz")
			writeRawTestBundle(t, archive, files, nil)
			if _, err := VerifyBundle(archive); err == nil {
				t.Fatal("alias or stray archive member accepted")
			}
		})
	}
}

func TestCanonicalMemberComponentLengthCompatibility(t *testing.T) {
	for _, test := range []struct {
		name  string
		valid bool
	}{
		{strings.Repeat("a", 255), true},
		{strings.Repeat("a", 256), false},
		{strings.Repeat("é", 127) + "a", true},
		{strings.Repeat("é", 128), false},
		{strings.Repeat("😀", 63) + "aaa", true},
		{strings.Repeat("😀", 64), false},
	} {
		err := canonicalMember("evidence/" + test.name)
		if (err == nil) != test.valid {
			t.Errorf("component length %d bytes: valid=%t, error=%v", len(test.name), test.valid, err)
		}
		if test.valid && len(utf16.Encode([]rune(test.name))) > 255 {
			t.Fatal("accepted component exceeds Windows UTF-16 limit")
		}
	}
	if err := canonicalMember(strings.Repeat("a", 256) + "/report.json"); err == nil {
		t.Fatal("oversized directory component accepted")
	}
}

func TestGeneratedSidecarComponentLengthCompatibility(t *testing.T) {
	for _, payloadBytes := range []int{248, 249, 255, 256} {
		members := make(map[string]bool)
		name := "oscal/" + strings.Repeat("a", payloadBytes)
		payloadErr := registerMember(members, name)
		sidecarErr := registerMember(members, name+".sha256")
		if (payloadErr == nil) != (payloadBytes <= 255) {
			t.Errorf("payload %d-byte component: %v", payloadBytes, payloadErr)
		}
		if (sidecarErr == nil) != (payloadBytes <= 248) {
			t.Errorf("payload %d-byte generated sidecar: %v", payloadBytes, sidecarErr)
		}
	}
}

func TestVerifyBundleRejectsManifestTraversalAndWrongCategories(t *testing.T) {
	for _, name := range []string{"../outside", "/outside", "C:/outside", "oscal/blob", "evidence/nested/blob", "./evidence/blob"} {
		t.Run(name, func(t *testing.T) {
			files := validBundleFiles(t)
			var manifest BundleManifest
			if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
				t.Fatal(err)
			}
			manifest.Evidence[0].Path = name
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			files["manifest.json"] = data
			files[name], files[name+".sha256"] = files["evidence/blob"], files["evidence/blob.sha256"]
			delete(files, "evidence/blob")
			delete(files, "evidence/blob.sha256")
			archive := filepath.Join(t.TempDir(), "traversal.tar.gz")
			writeRawTestBundle(t, archive, files, nil)
			if _, err := VerifyBundle(archive); err == nil {
				t.Fatal("invalid manifest path accepted")
			}
		})
	}
}

func TestVerifyBundleRejectsHiddenSecondTarAndAllowsZeroPadding(t *testing.T) {
	files := validBundleFiles(t)
	var second bytes.Buffer
	writer := tar.NewWriter(&second)
	if err := writeTarEntry(writer, "evidence/blob", []byte("overwrite verified evidence")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range [][]byte{second.Bytes(), bytes.Repeat([]byte{0}, 4096)} {
		archive := filepath.Join(t.TempDir(), "trailing.tar.gz")
		writeRawTestBundle(t, archive, files, suffix)
		_, err := VerifyBundle(archive)
		if suffix[0] != 0 && err == nil {
			t.Fatal("hidden second TAR accepted")
		}
		if suffix[0] == 0 && err != nil {
			t.Fatalf("legitimate zero TAR padding rejected: %v", err)
		}
	}
}

func TestProducerEnforcesPortableNamesAndSameByteLimit(t *testing.T) {
	parent := t.TempDir()
	artifact := filepath.Join(parent, `invalid\name.json`)
	mustBundleWrite(t, artifact, []byte("artifact"))
	output := filepath.Join(parent, "bundle.tar.gz")
	mustBundleWrite(t, output, []byte("original archive"))
	if err := CreateAuditBundle([]string{artifact}, "", output); err == nil {
		t.Fatal("non-portable artifact basename accepted")
	}
	if err := createAuditBundle(nil, "", output, 1024); err == nil {
		t.Fatal("oversized producer archive accepted")
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "original archive" {
		t.Fatalf("failed producer replaced output: %q %v", data, err)
	}
	if err := CreateAuditBundle(nil, "", output); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := createAuditBundle(nil, "", output, int64(len(stream))); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBundle(output); err != nil {
		t.Fatal(err)
	}
}

func validBundleFiles(t *testing.T) map[string][]byte {
	t.Helper()
	parent := t.TempDir()
	mustBundleWrite(t, filepath.Join(parent, "blob"), []byte("verified evidence"))
	archive := filepath.Join(t.TempDir(), "valid.tar.gz")
	if err := CreateAuditBundle(nil, parent, archive); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBundle(archive); err != nil {
		t.Fatalf("legitimate archive rejected: %v", err)
	}
	return inspectBundle(t, archive)
}

func writeRawTestBundle(t *testing.T, filename string, files map[string][]byte, suffix []byte) {
	t.Helper()
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(f)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, data := range files {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := gzipWriter.Write(suffix); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
