package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanMetadataBoundary(t *testing.T) {
	for _, metadata := range []string{"go.mod", "package.json", "Dockerfile", "k8s", "nested.tf"} {
		for _, mode := range []string{"absolute", "relative", "chain", "broken"} {
			t.Run(metadata+"/"+mode, func(t *testing.T) {
				selected, outside := scanBoundaryFixture(t)
				target := filepath.Join(outside, "sentinel")
				if mode == "relative" {
					target = "../selected-sibling/sentinel"
				}
				if mode == "broken" {
					target = "missing"
				}
				if mode == "chain" {
					mustSymlink(t, "../selected-sibling/sentinel", filepath.Join(selected, "bridge"))
					target = "bridge"
				}
				mustSymlink(t, target, filepath.Join(selected, metadata))
				if _, err := ScanRepository(selected); err == nil {
					t.Fatal("scanner accepted forbidden or broken metadata")
				}
			})
		}
	}
	if _, err := ScanRepository(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing selected root accepted")
	}
}

func TestScaffoldDerivedWritesConfined(t *testing.T) {
	for _, name := range []string{".xoscal", ".xoscal/xoscal.yaml", "component-definition.json"} {
		t.Run(name, func(t *testing.T) {
			selected, outside := scanBoundaryFixture(t)
			target := filepath.Join(outside, "sentinel")
			if name == ".xoscal" {
				target = outside
			}
			if name == ".xoscal/xoscal.yaml" {
				if err := os.Mkdir(filepath.Join(selected, ".xoscal"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			mustSymlink(t, target, filepath.Join(selected, name))
			if _, _, err := ScaffoldWorkspace(selected, ""); err == nil {
				t.Fatal("scaffold accepted escaping destination")
			}
			data, err := os.ReadFile(filepath.Join(outside, "sentinel"))
			if err != nil || string(data) != "outside sentinel" {
				t.Fatalf("outside changed: %q %v", data, err)
			}
			if _, err := os.Stat(filepath.Join(outside, "xoscal.yaml")); !os.IsNotExist(err) {
				t.Fatalf("outside config created: %v", err)
			}
		})
	}
}

func TestScaffoldAllowsRelativeInsideAndSelectedRootLinks(t *testing.T) {
	selected, _ := scanBoundaryFixture(t)
	if err := os.WriteFile(filepath.Join(selected, "metadata"), []byte("module example.org/inside\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, "metadata", filepath.Join(selected, "go.mod"))
	if err := os.Mkdir(filepath.Join(selected, "configuration"), 0700); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, "configuration", filepath.Join(selected, ".xoscal"))
	rootLink := filepath.Join(filepath.Dir(selected), "root-link")
	mustSymlink(t, selected, rootLink)
	scan, _, err := ScaffoldWorkspace(rootLink, "")
	if err != nil || scan.Name != "inside" {
		t.Fatalf("relative inside/root link failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(selected, "configuration", "xoscal.yaml")); err != nil {
		t.Fatal(err)
	}
}

func scanBoundaryFixture(t *testing.T) (string, string) {
	t.Helper()
	parent := t.TempDir()
	selected, outside := filepath.Join(parent, "selected"), filepath.Join(parent, "selected-sibling")
	for _, dir := range []string{selected, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("outside sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	return selected, outside
}

func mustSymlink(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}
