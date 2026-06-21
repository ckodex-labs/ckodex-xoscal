package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

// Xoscal encapsulates the full SSDLC / SLSA / Release / Dist pipeline.
type Xoscal struct {
	// +optional
	// Version is the build version injected into the binary.
	Version string
}

// New creates a new Xoscal pipeline with default version "dev".
func New() *Xoscal {
	return &Xoscal{Version: "dev"}
}

// WithVersion returns a copy of Xoscal with the version set.
func (m *Xoscal) WithVersion(version string) *Xoscal {
	m.Version = version
	return m
}

// base returns a Go builder container with source mounted and deps cached.
// Uses two-phase mounting, APT cache volumes, and persistent Go caches.
func (m *Xoscal) base(source *dagger.Directory) *dagger.Container {
	modCache := dag.CacheVolume("go-mod")
	buildCache := dag.CacheVolume("go-build")
	aptCache := dag.CacheVolume("apt-cache")
	aptLists := dag.CacheVolume("apt-lists")

	// Phase 1: download modules using only go.mod/go.sum for cache isolation.
	goBase := dag.Container().
		From("golang:1.25-bookworm").
		WithMountedCache("/go/pkg/mod", modCache).
		WithMountedCache("/root/.cache/go-build", buildCache).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("DEBIAN_FRONTEND", "noninteractive").
		WithMountedCache("/var/cache/apt", aptCache).
		WithMountedCache("/var/lib/apt/lists", aptLists).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "git", "curl"}).
		WithFile("/src/go.mod", source.File("go.mod")).
		WithFile("/src/go.sum", source.File("go.sum")).
		WithWorkdir("/src").
		WithExec([]string{"go", "mod", "download"})

	// Phase 2: mount full source on top of warmed module cache.
	return goBase.
		WithDirectory("/src", source).
		WithWorkdir("/src")
}

// Build compiles a static release binary and returns it.
func (m *Xoscal) Build(source *dagger.Directory) *dagger.File {
	ldflags := fmt.Sprintf("-ldflags=-s -w -X main.version=%s", m.Version)
	return m.base(source).
		WithExec([]string{"go", "build", ldflags, "-o", "/bin/xoscal-server", "./server/cmd/xoscal-server"}).
		File("/bin/xoscal-server")
}

// Test runs unit tests.
func (m *Xoscal) Test(source *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithExec([]string{"go", "test", "-v", "./server/..."}).
		WithExec([]string{"sh", "-c", "echo 'test-ok' > /tmp/test.ok"})
}

// TestRace runs tests with the race detector.
func (m *Xoscal) TestRace(source *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithEnvVariable("CGO_ENABLED", "1").
		WithExec([]string{"go", "test", "-race", "./server/..."}).
		WithExec([]string{"sh", "-c", "echo 'race-ok' > /tmp/race.ok"})
}

// Coverage generates an HTML coverage report and returns it.
func (m *Xoscal) Coverage(source *dagger.Directory) *dagger.File {
	return m.base(source).
		WithExec([]string{"go", "test", "-coverprofile=coverage.out", "./server/..."}).
		WithExec([]string{"go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html"}).
		File("/src/coverage.html")
}

// toolBase returns a container with lint/security tools pre-installed.
// The layer is cached independently of source changes.
func (m *Xoscal) toolBase() *dagger.Container {
	aptCache := dag.CacheVolume("apt-cache")
	aptLists := dag.CacheVolume("apt-lists")
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedCache("/var/cache/apt", aptCache).
		WithMountedCache("/var/lib/apt/lists", aptLists).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "git", "curl"}).
		WithExec([]string{"sh", "-c", "curl -fsSL https://github.com/bufbuild/buf/releases/download/v1.50.0/buf-$(uname -s)-$(uname -m) -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf"}).
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"}).
		WithExec([]string{"go", "install", "github.com/securego/gosec/v2/cmd/gosec@latest"})
}

// Lint runs buf, go vet, and gofmt checks.
func (m *Xoscal) Lint(source *dagger.Directory) *dagger.Container {
	c := m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"buf", "lint"})
	c = c.WithExec([]string{"go", "vet", "./..."})
	c = c.WithExec([]string{"sh", "-c", "gofmt -d . | tee /tmp/gofmt.diff; test -s /tmp/gofmt.diff && exit 1 || true"})
	return c.WithExec([]string{"sh", "-c", "echo 'lint-ok' > /tmp/lint.ok"})
}

// Proto regenerates all SDKs from the protos via `buf generate` and returns
// the regenerated proto tree. Runs in-container so buf.build remote plugins
// have network (they cannot run in a restricted local sandbox).
func (m *Xoscal) Proto(source *dagger.Directory) *dagger.Directory {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"buf", "generate"}).
		Directory("/src/proto")
}

// ProtoCheck is the SDK-drift gate: it regenerates from the protos and fails
// if the committed generated code differs — i.e. protos and SDKs are out of
// sync. Keeps the wire contract and its generated SDKs provably aligned.
func (m *Xoscal) ProtoCheck(source *dagger.Directory) *dagger.Container {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"git", "config", "--global", "--add", "safe.directory", "/src"}).
		WithExec([]string{"buf", "lint"}).
		WithExec([]string{"buf", "generate"}).
		WithExec([]string{"sh", "-c",
			"git diff --stat -- proto/ | tee /tmp/proto.diff; " +
				"if [ -s /tmp/proto.diff ]; then echo 'SDK drift: protos changed but generated SDKs not regenerated' >&2; exit 1; fi"}).
		WithExec([]string{"sh", "-c", "echo 'proto-ok' > /tmp/proto.ok"})
}

// Security runs govulncheck and gosec, returning the SARIF report file.
// Reuses the cached toolBase layer so tools are not reinstalled on every run.
func (m *Xoscal) Security(source *dagger.Directory) *dagger.File {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"govulncheck", "./..."}).
		WithExec([]string{"gosec", "-fmt", "sarif", "-out", "gosec-results.sarif", "-exclude-dir=proto", "-no-fail", "./..."}).
		File("/src/gosec-results.sarif")
}

// Image builds a distroless container image and returns it.
func (m *Xoscal) Image(source *dagger.Directory) *dagger.Container {
	bin := m.Build(source)
	return dag.Container().
		From("gcr.io/distroless/base-debian12:nonroot").
		WithWorkdir("/data").
		WithFile("/xoscal-server", bin).
		WithExposedPort(50051).
		WithExposedPort(9090).
		WithEntrypoint([]string{"/xoscal-server"}).
		WithDefaultArgs([]string{"-config", "/etc/xoscal/config.yaml"})
}

// Sbom generates a CycloneDX SBOM for the built binary using syft.
func (m *Xoscal) Sbom(source *dagger.Directory) *dagger.File {
	bin := m.Build(source)
	return dag.Container().
		From("anchore/syft:latest").
		WithFile("/xoscal-server", bin).
		WithExec([]string{"packages", "file:/xoscal-server", "-o", "cyclonedx-json", "-q", "--file", "/tmp/sbom.json"}).
		File("/tmp/sbom.json")
}

// Dist packages the binary and SBOM into a directory.
func (m *Xoscal) Dist(source *dagger.Directory) *dagger.Directory {
	bin := m.Build(source)
	sbom := m.Sbom(source)
	return dag.Directory().
		WithFile("xoscal-server", bin).
		WithFile("sbom.cyclonedx.json", sbom)
}

// goreleaserBase returns a container with GoReleaser tooling pre-installed.
func (m *Xoscal) goreleaserBase() *dagger.Container {
	aptCache := dag.CacheVolume("apt-cache")
	aptLists := dag.CacheVolume("apt-lists")
	return dag.Container().
		From("golang:1.25").
		WithMountedCache("/var/cache/apt", aptCache).
		WithMountedCache("/var/lib/apt/lists", aptLists).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "git", "curl"}).
		WithExec([]string{"sh", "-c", "curl -fsSL https://github.com/goreleaser/goreleaser/releases/latest/download/goreleaser_Linux_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz | tar -xzf - -C /usr/local/bin goreleaser"}).
		WithExec([]string{"sh", "-c", "curl -fsSL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"}).
		WithExec([]string{"sh", "-c", "curl -fsSL https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o /usr/local/bin/cosign && chmod +x /usr/local/bin/cosign"})
}

// goreleaser returns a GoReleaser container with source, caches, and token mounted.
func (m *Xoscal) goreleaser(source *dagger.Directory, githubToken *dagger.Secret) *dagger.Container {
	modCache := dag.CacheVolume("go-mod")
	buildCache := dag.CacheVolume("go-build")
	return m.goreleaserBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", modCache).
		WithMountedCache("/root/.cache/go-build", buildCache).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("COSIGN_EXPERIMENTAL", "1").
		WithSecretVariable("GITHUB_TOKEN", githubToken)
}

// Release runs GoReleaser release --clean (publishes GitHub release, archives, SBOMs, signs).
func (m *Xoscal) Release(source *dagger.Directory, githubToken *dagger.Secret) *dagger.Directory {
	return m.goreleaser(source, githubToken).
		WithExec([]string{"goreleaser", "release", "--clean"}).
		Directory("/src/dist")
}

// Snapshot runs GoReleaser in snapshot mode (no publish, no sign, for CI validation).
func (m *Xoscal) Snapshot(source *dagger.Directory, githubToken *dagger.Secret) *dagger.Directory {
	return m.goreleaser(source, githubToken).
		WithExec([]string{"goreleaser", "release", "--clean", "--snapshot", "--skip=sign"}).
		Directory("/src/dist")
}

// All runs lint, test, security, and race checks in parallel branches.
// Each branch shares the cached base container. The returned directory
// contains outputs from all four parallel checks.
func (m *Xoscal) All(source *dagger.Directory) *dagger.Directory {
	lint := m.Lint(source)
	test := m.Test(source)
	race := m.TestRace(source)
	sec := m.Security(source)
	proto := m.ProtoCheck(source)

	return dag.Directory().
		WithFile("lint.ok", lint.File("/tmp/lint.ok")).
		WithFile("test.ok", test.File("/tmp/test.ok")).
		WithFile("race.ok", race.File("/tmp/race.ok")).
		WithFile("proto.ok", proto.File("/tmp/proto.ok")).
		WithFile("gosec-results.sarif", sec)
}
