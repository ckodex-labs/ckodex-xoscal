package modulecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise actual Go tidy with local-only fixtures, including independent
// manifest and checksum drift. No Git repository or network oracle is needed.
func TestModuleTidyAdmission(t *testing.T) {
	for _, fixture := range []struct {
		name, mod, sum string
		pass           bool
	}{
		{"unchanged", "module example.invalid/fixture\n\ngo 1.25.0\n", "", true},
		{"module-drift", "module example.invalid/fixture\n\ngo 1.25.0\n\nrequire example.invalid/unused v0.0.0\n\nreplace example.invalid/unused => ./unused\n", "", false},
		{"checksum-drift", "module example.invalid/fixture\n\ngo 1.25.0\n", "example.invalid/unused v1.0.0 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			write := func(name, contents string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(filepath.Join(root, "unused"), 0o700); err != nil {
				t.Fatal(err)
			}
			write("unused/go.mod", "module example.invalid/unused\n\ngo 1.25.0\n")
			write("main.go", "package main\nfunc main() {}\n")
			write("go.mod", fixture.mod)
			write("go.sum", fixture.sum)
			write("input.mod", fixture.mod)
			write("input.sum", fixture.sum)
			marker := filepath.Join(root, "tidy.ok")
			command := exec.Command("sh", "-c", Script, "module-tidy-check",
				filepath.Join(root, "input.mod"), filepath.Join(root, "input.sum"), marker)
			command.Dir = root
			for _, entry := range os.Environ() {
				key := strings.SplitN(entry, "=", 2)[0]
				if key != "GOWORK" && key != "GOFLAGS" && key != "GOTOOLCHAIN" && key != "GOPROXY" {
					command.Env = append(command.Env, entry)
				}
			}
			command.Env = append(command.Env, "GOWORK=off", "GOFLAGS=-p=2 -mod=readonly", "GOTOOLCHAIN=local", "GOPROXY=off")
			output, err := command.CombinedOutput()
			if fixture.pass {
				if err != nil {
					t.Fatalf("tidy admission failed: %v\n%s", err, output)
				}
				contents, err := os.ReadFile(marker)
				if err != nil || string(contents) != "module-tidy-ok\n" {
					t.Fatalf("missing exact admission marker: %q, %v", contents, err)
				}
			} else {
				if err == nil || !(strings.Contains(string(output), "differ") || strings.Contains(string(output), "EOF")) {
					t.Fatalf("tidy drift did not fail byte comparison: %v\n%s", err, output)
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatalf("failed tidy must not emit admission: %v", err)
				}
			}
		})
	}
}
