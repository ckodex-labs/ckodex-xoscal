package rootfs

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDirectoryBoundary(t *testing.T) {
	parent := t.TempDir()
	selected := filepath.Join(parent, "selected")
	outside := filepath.Join(parent, "selected-sibling")
	for _, dir := range []string{selected, outside} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "blob"), []byte(dir), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(selected, filepath.Join(parent, "root-link")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Join(parent, "root-link"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	links := map[string]string{"inside": "blob", "absolute-inside": filepath.Join(selected, "blob"),
		"absolute-outside": filepath.Join(outside, "blob"), "relative-outside": "../selected-sibling/blob",
		"chain": "relative-outside", "broken": "missing"}
	for name, target := range links {
		if err := root.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}
	for name := range links {
		t.Run(name, func(t *testing.T) {
			data, err := ReadFile(root, name)
			if name == "inside" {
				if err != nil || string(data) != selected {
					t.Fatalf("inside link: %q, %v", data, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("escaped or invalid link read: %q", data)
			}
			if err := root.WriteFile(name, []byte("overwrite"), 0600); err == nil && name != "broken" {
				t.Fatal("escaped write")
			}
		})
	}
	data, err := os.ReadFile(filepath.Join(outside, "blob"))
	if err != nil || string(data) != outside {
		t.Fatalf("outside sentinel changed: %q %v", data, err)
	}
}

func TestSymlinkSwapDoesNotEscape(t *testing.T) {
	parent := t.TempDir()
	selected := filepath.Join(parent, "selected")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(selected, "inside"), []byte("inside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret"), []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(selected)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Symlink("inside", "link"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			target := "inside"
			if i%2 == 0 {
				target = "../secret"
			}
			if err := root.Symlink(target, "replacement"); err != nil {
				t.Error(err)
				return
			}
			if err := root.Rename("replacement", "link"); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 500; i++ {
		data, err := ReadFile(root, "link")
		if err == nil && string(data) != "inside" {
			t.Errorf("outside bytes returned: %q", data)
		}
	}
	wg.Wait()
}
