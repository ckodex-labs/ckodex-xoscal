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
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
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
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
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

// SpecRegistryCheck verifies the committed spec-registry hashes are lock-step with the pinned OSCAL release assets (re-fetches each model schema and compares).
// Needs network (fetches release assets), so it runs in-container.
//
// Drift posture (intentional, distinct from the reconcile workflow): any
// non-zero verify exit — including exit 3 (per-model drift) — hard-fails this
// gate, because CI MUST be lock-step with the pinned spec. The async reconcile
// workflow (oscal-reconcile.yml) instead tolerates exit 3 and opens a handoff
// PR. CI = enforce now; reconcile = propose a fix. (Rule 7: one policy each, stated.)
func (m *Xoscal) SpecRegistryCheck(source *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/spec-registry", "./server/cmd/xoscal-spec-registry"}).
		WithExec([]string{"/bin/spec-registry", "-mode", "verify", "-registry", "data/oscal/spec-registry.yaml"}).
		WithExec([]string{"sh", "-c", "echo 'specreg-ok' > /tmp/specreg.ok"})
}

// sdkBundles runs buf generate and zips each language SDK with a sha256 sidecar.
// Also extracts the OpenAPI document for the docs page.
func (m *Xoscal) sdkBundles(source *dagger.Directory) *dagger.Directory {
	langs := "go python java csharp ts swift"
	gen := m.toolBase().
		WithExec([]string{"sh", "-c", "apt-get update && apt-get install -y --no-install-recommends zip"}).
		WithDirectory("/src", source).
		WithWorkdir("/src").
		// Use the nested proto/oscal module config: it emits every SDK to a
		// uniform gen/<lang> layout (incl. gen/go). The ROOT buf.gen.yaml sends
		// the Go SDK to proto/oscal/ source-relative, so gen/go never exists there.
		WithExec([]string{"sh", "-c", "cd proto/oscal && buf generate"}).
		WithExec([]string{"mkdir", "-p", "/out"}).
		WithExec([]string{"sh", "-c",
			"for l in " + langs + "; do " +
				"d=proto/oscal/gen/$l; [ -d \"$d\" ] || { echo \"missing $d\" >&2; exit 1; }; " +
				"(cd \"$d\" && zip -qr /out/$l.zip .); " +
				"sha256sum /out/$l.zip | awk '{print \"sha256:\"$1}' > /out/$l.zip.sha256; " +
				"done"}).
		WithExec([]string{"sh", "-c",
			"f=\"$(find proto/oscal/gen/openapi -name '*.json' -o -name '*.yaml' | head -n1)\"; [ -n \"$f\" ] || { echo 'no openapi doc found' >&2; exit 1; }; cp \"$f\" /out/openapi.json; test -s /out/openapi.json"})
	return gen.Directory("/out")
}

// oscalFrameworks builds and runs xoscal-export-frameworks over the manifest,
// emitting one OSCAL catalog per framework. Needs network (fetches upstream).
func (m *Xoscal) oscalFrameworks(source *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/export-fw", "./server/cmd/xoscal-export-frameworks"}).
		WithExec([]string{"/bin/export-fw", "-manifest", "data/frameworks/manifest.yaml", "-out", "/out", "-dsn", "/tmp/fw.db"}).
		Directory("/out")
}

// provenanceManifest digests the proto set and each SDK zip and renders
// provenance.json. Signing fields are left empty here (no cosign/rekor in this
// build stage), so every record is honestly signed:false until Release wires them.
func (m *Xoscal) provenanceManifest(source *dagger.Directory, sdks *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithDirectory("/sdks", sdks).
		WithExec([]string{"go", "build", "-o", "/bin/prov", "./server/cmd/xoscal-provenance"}).
		WithExec([]string{"sh", "-c",
			"proto_digest=$(find proto/oscal -name '*.proto' | sort | xargs sha256sum | sha256sum | awk '{print \"sha256:\"$1}'); " +
				"printf '[{\"Name\":\"proto-set\",\"Digest\":\"%s\"}' \"$proto_digest\" > /tmp/arts.json; " +
				"for z in /sdks/*.zip; do " +
				"d=$(sha256sum \"$z\" | awk '{print \"sha256:\"$1}'); " +
				"printf ',{\"Name\":\"%s\",\"Digest\":\"%s\"}' \"$(basename $z .zip)-sdk\" \"$d\" >> /tmp/arts.json; " +
				"done; printf ']' >> /tmp/arts.json"}).
		WithExec([]string{"mkdir", "-p", "/out"}).
		WithExec([]string{"/bin/prov", "-in", "/tmp/arts.json", "-out", "/out/provenance.json"}).
		Directory("/out")
}

// buildDownloadsIndex emits /out/downloads.json from the SDK zips and OSCAL catalogs.
const buildDownloadsIndex = `
{
  echo '{"sdks":['
  first=1
  for z in /out/sdk/*.zip; do
    [ -e "$z" ] || continue
    d=$(cat "$z.sha256" 2>/dev/null || echo "")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"sdk/%s","digest":"%s"}' "$(basename "$z")" "$(basename "$z")" "$d"
  done
  echo '],"frameworks":['
  first=1
  for c in /out/frameworks/*/catalog.json; do
    [ -e "$c" ] || continue
    d=$(cat "$c.sha256" 2>/dev/null || echo "")
    name=$(basename "$(dirname "$c")")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"frameworks/%s/catalog.json","digest":"%s"}' "$name" "$name" "$d"
  done
  echo ']}'
} > /out/downloads.json
`

// Site assembles the static GitHub Pages portal: docs (Scalar+OpenAPI),
// transparency (provenance.json), and downloads (SDK zips + OSCAL catalogs).
func (m *Xoscal) Site(source *dagger.Directory) *dagger.Directory {
	sdks := m.sdkBundles(source)
	frameworks := m.oscalFrameworks(source)
	prov := m.provenanceManifest(source, sdks)

	// Vendor a PINNED Scalar standalone bundle into the site (no runtime CDN).
	scalarVer := "1.25.0" // pin explicitly; bump deliberately
	scalarURL := "https://cdn.jsdelivr.net/npm/@scalar/api-reference@" + scalarVer + "/dist/browser/standalone.js"

	asm := m.base(source).
		WithDirectory("/out", source.Directory("site")).
		WithDirectory("/out/sdk", sdks).
		WithDirectory("/out/frameworks", frameworks).
		WithFile("/out/openapi.json", sdks.File("openapi.json")).
		WithFile("/out/provenance.json", prov.File("provenance.json")).
		WithExec([]string{"sh", "-c",
			"curl -fsSL '" + scalarURL + "' -o /out/scalar.js && test -s /out/scalar.js"}).
		WithExec([]string{"sh", "-c", buildDownloadsIndex}).
		Directory("/out")
	return asm
}

// Serve builds the portal site and serves it as a static HTTP service on :8080.
// One command to preview the real site locally (built on the engine, so buf
// generate + framework fetch have network):
//
//	dagger call serve --source=. up --ports 8080:8080
//
// then open http://localhost:8080.
func (m *Xoscal) Serve(source *dagger.Directory) *dagger.Service {
	return dag.Container().
		From("python:3.13-alpine").
		WithDirectory("/site", m.Site(source)).
		WithWorkdir("/site").
		WithExposedPort(8080).
		AsService(dagger.ContainerAsServiceOpts{
			Args: []string{"python", "-m", "http.server", "8080"},
		})
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
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
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

// All runs lint, test, race, security, proto-drift, and spec-registry checks in parallel branches.
// Each branch shares the cached base container. The returned directory
// contains outputs from all four parallel checks.
func (m *Xoscal) All(source *dagger.Directory) *dagger.Directory {
	lint := m.Lint(source)
	test := m.Test(source)
	race := m.TestRace(source)
	sec := m.Security(source)
	proto := m.ProtoCheck(source)
	specreg := m.SpecRegistryCheck(source)

	return dag.Directory().
		WithFile("lint.ok", lint.File("/tmp/lint.ok")).
		WithFile("test.ok", test.File("/tmp/test.ok")).
		WithFile("race.ok", race.File("/tmp/race.ok")).
		WithFile("proto.ok", proto.File("/tmp/proto.ok")).
		WithFile("specreg.ok", specreg.File("/tmp/specreg.ok")).
		WithFile("gosec-results.sarif", sec)
}
