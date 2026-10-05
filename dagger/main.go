package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

const (
	goBuilderImage         = "golang:1.25-bookworm@sha256:e401dae1bf814e29204a8cb7915682e1780951e609ca0dd8865ee1937f510c48"
	govulncheckVersion     = "v1.7.0"
	gosecVersion           = "v2.28.0"
	goreleaserVersion      = "v2.17.1"
	bufVersion             = "v1.71.0"
	bufLinuxAMD64SHA       = "d3de2838c68a5759ca276884254bc70df4e4ad185d6ed5f65f327b6ce6363eab"
	bufLinuxARM64SHA       = "041c15f3a8c4bd6cf36285d7a9ef290cd3e2536ef3bfd3de64d1f70cc5144c6e"
	sbomToolsVersion       = "v0.2.0"
	sbomToolsLinuxAMD64SHA = "0f93406da9643a8ea3afe4fb56c7418fb74d29ecaa0af49b260dc4b2ff597956"
	sbomToolsLinuxARM64SHA = "fd3ed73f4d259704fa1eb7065e3fcbf45213751b3a7cbe3f0ae3429d88ad24bb"
	syftVersion            = "v1.51.0"
	syftLinuxAMD64SHA      = "2a2e837a2c8d59ec9af5472ee22d3b04ee463c4e44476ecf993fd1e5ab6ebc7f"
	syftLinuxARM64SHA      = "6c0466811541ea03add5213a60a1562f0851e4c0b0ecfdee1a694a9455285900"
	distrolessBase         = "gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3"
	ubuntuBase             = "ubuntu:24.04@sha256:561618e2c15bf2397621dd04f96926663a3b5616c189cf7e38db7e82f5c538ea"
)

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
		From(goBuilderImage).
		WithMountedCache("/go/pkg/mod", modCache).
		WithMountedCache("/root/.cache/go-build", buildCache).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOFLAGS", "-p=2 -mod=readonly").
		WithEnvVariable("GOWORK", "off").
		WithEnvVariable("GOMAXPROCS", "2").
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

// Build compiles a static release binary for xoscal-server and returns it.
func (m *Xoscal) Build(source *dagger.Directory) *dagger.File {
	ldflags := fmt.Sprintf("-ldflags=-s -w -X main.version=%s", m.Version)
	return m.base(source).
		WithExec([]string{"go", "build", ldflags, "-o", "/bin/xoscal-server", "./server/cmd/xoscal-server"}).
		File("/bin/xoscal-server")
}

// BuildCtl compiles a static release binary for xoscal-ctl CLI and returns it.
func (m *Xoscal) BuildCtl(source *dagger.Directory) *dagger.File {
	ldflags := fmt.Sprintf("-ldflags=-s -w -X main.version=%s", m.Version)
	return m.base(source).
		WithExec([]string{"go", "build", ldflags, "-o", "/bin/xoscal-ctl", "./server/cmd/xoscal-ctl"}).
		File("/bin/xoscal-ctl")
}

// BuildBackup compiles a static release binary for xoscal-backup and returns it.
func (m *Xoscal) BuildBackup(source *dagger.Directory) *dagger.File {
	ldflags := fmt.Sprintf("-ldflags=-s -w -X main.version=%s", m.Version)
	return m.base(source).
		WithExec([]string{"go", "build", ldflags, "-o", "/bin/xoscal-backup", "./server/cmd/xoscal-backup"}).
		File("/bin/xoscal-backup")
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
		From(goBuilderImage).
		WithEnvVariable("GOMAXPROCS", "2").
		WithEnvVariable("GOFLAGS", "-p=2 -mod=readonly").
		WithEnvVariable("GOWORK", "off").
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "git", "curl", "python3"}).
		WithExec([]string{"sh", "-c", pinnedBufInstall()}).
		WithExec([]string{"go", "install", fmt.Sprintf("golang.org/x/vuln/cmd/govulncheck@%s", govulncheckVersion)}).
		WithExec([]string{"go", "install", fmt.Sprintf("github.com/securego/gosec/v2/cmd/gosec@%s", gosecVersion)})
}

// Lint runs code, proto, design, and accessibility checks.
func (m *Xoscal) Lint(source *dagger.Directory) *dagger.Container {
	c := m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"buf", "lint"})
	c = c.WithExec([]string{"go", "vet", "./..."})
	c = c.WithExec([]string{"sh", "-c", "gofmt -d . | tee /tmp/gofmt.diff; test -s /tmp/gofmt.diff && exit 1 || true"})
	c = c.WithExec([]string{"python3", "scripts/design-lint.py"})
	c = c.WithExec([]string{"python3", "scripts/a11y-lint.py"})
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

// ProtoCheck is the SDK-drift gate: it snapshots the supplied proto tree,
// regenerates from the protos, and fails if generation changes the tree. The
// before/after comparison is intentional: git diff is not a valid oracle when
// Dagger is run from a dirty development checkout containing intended changes.
func (m *Xoscal) ProtoCheck(source *dagger.Directory) *dagger.Container {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"buf", "lint"}).
		WithExec([]string{"sh", "-c", "rm -rf /tmp/proto-before && mkdir -p /tmp/proto-before && cp -a proto/. /tmp/proto-before/"}).
		WithExec([]string{"sh", "-c", "find /tmp/proto-before/oscal/gen/java -type f -name '*.java' -exec sed -i 's/[[:space:]]*$//' {} +"}).
		WithExec([]string{"buf", "generate"}).
		// The Buf Java plugin emits platform-dependent trailing whitespace;
		// normalize that generated-only noise before the source-relative drift
		// comparison and SDK packaging.
		WithExec([]string{"sh", "-c", "find proto/oscal/gen/java -type f -name '*.java' -exec sed -i 's/[[:space:]]*$//' {} +"}).
		WithExec([]string{"sh", "-c",
			"diff -qr /tmp/proto-before proto > /tmp/proto.diff || true; " +
				"if [ -s /tmp/proto.diff ]; then cat /tmp/proto.diff >&2; echo 'SDK drift: buf generate changed the supplied proto tree' >&2; exit 1; fi"}).
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

// SdkBundles returns tested language distributions and source bundles.
func (m *Xoscal) SdkBundles(source *dagger.Directory) *dagger.Directory {
	return m.SdkPackages(source)
}

// OscalFrameworks builds and runs xoscal-export-frameworks over the manifest,
// emitting one OSCAL catalog per framework. Needs network (fetches upstream).
func (m *Xoscal) OscalFrameworks(source *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/export-fw", "./server/cmd/xoscal-export-frameworks"}).
		WithExec([]string{"/bin/export-fw", "-manifest", "data/frameworks/manifest.yaml", "-out", "/out", "-dsn", "/tmp/fw.db"}).
		Directory("/out")
}

// OscalSchemaValidation builds and runs xoscal-validate-schema against every
// framework catalog produced by OscalFrameworks. Fails if any catalog does
// not conform to the pinned OSCAL JSON schema.
func (m *Xoscal) OscalSchemaValidation(source *dagger.Directory, frameworks *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/validate-schema", "./server/cmd/xoscal-validate-schema"}).
		WithDirectory("/frameworks", frameworks).
		WithExec([]string{"sh", "-c",
			"fail=0; count=0; for f in /frameworks/*/catalog.json /frameworks/*/profile.json; do " +
				"[ -f \"$f\" ] || continue; " +
				"kind=$(basename \"$f\" .json); count=$((count+1)); if ! /bin/validate-schema -file \"$f\" -kind \"$kind\"; then " +
				"echo \"FAIL: $f\" >&2; fail=1; fi; done; " +
				"if [ $fail -eq 0 ] && [ $count -gt 0 ]; then echo 'oscal-schema-ok' > /tmp/schema.ok; else exit 3; fi"})
}

// OscalConstraintValidation builds and runs xoscal-validate-constraints against
// every framework catalog and profile. This is tier-2 validation: Metaschema constraint
// checking via oscal-cli (allowed-values, has-cardinality, index-has-key,
// is-unique, matches). Any constraint or tool failure blocks admission; raw
// per-file logs and statuses remain independently exportable.
func (m *Xoscal) OscalConstraintValidation(source *dagger.Directory, frameworks *dagger.Directory) *dagger.Container {
	return m.OscalConstraintReports(source, frameworks).
		WithExec([]string{"sh", "-ec", "test \"$(cat /tmp/constraint-evidence/result.status)\" = 0; echo 'oscal-constraints-ok' > /tmp/constraints.ok"})
}

// OscalConstraintReports collects each actual catalog/profile result without
// granting admission. Use OscalConstraintValidation for the enforcing gate.
func (m *Xoscal) OscalConstraintReports(source *dagger.Directory, frameworks *dagger.Directory) *dagger.Container {
	const oscalCLIVersion = "1.0.3"
	const oscalCLIHash = "c90166b6f94a8b32a4c0012bb3f3d32dfa54b950aed31ac1a4902ce75fc6a57e"
	validator := m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/validate-constraints", "./server/cmd/xoscal-validate-constraints"}).File("/bin/validate-constraints")
	// Reuse the digest-pinned Java SDK producer rather than installing another
	// mutable APT runtime. The JDK's jar tool extracts the checksum-pinned CLI.
	return sdkToolchain("java").
		WithExec([]string{"sh", "-ec", "apt-get update && apt-get install -y --no-install-recommends python3"}).
		WithExec([]string{"sh", "-ec",
			fmt.Sprintf(
				"curl -fsSL -o /tmp/oscal-cli.zip "+
					"https://repo1.maven.org/maven2/gov/nist/secauto/oscal/tools/oscal-cli/cli-core/%s/cli-core-%s-oscal-cli.zip && "+
					"echo '%s  /tmp/oscal-cli.zip' | sha256sum -c - && "+
					"mkdir -p /opt/oscal-cli && cd /opt/oscal-cli && jar xf /tmp/oscal-cli.zip && "+
					"echo '65b7e116fc4991ccaf8f6e841d4af955313efbc6aad3b1de4472e0408c352dda  /opt/oscal-cli/lib/gov.nist.secauto.oscal.liboscal-java-3.0.3.jar' | sha256sum -c - && "+
					"chmod +x /opt/oscal-cli/bin/oscal-cli",
				oscalCLIVersion, oscalCLIVersion, oscalCLIHash)}).
		WithFile("/bin/validate-constraints", validator).
		WithFile("/tools/oscal-semantic-scope.py", source.File("scripts/oscal-semantic-scope.py")).
		WithDirectory("/frameworks", frameworks).
		WithExec([]string{"sh", "-c",
			"mkdir -p /tmp/constraint-evidence; : > /tmp/constraint-evidence/results.tsv; fail=0; count=0; python3 /tools/oscal-semantic-scope.py /frameworks /tmp/constraint-evidence/semantic-scope.json || fail=1; for f in /frameworks/*/catalog.json /frameworks/*/profile.json; do " +
				"[ -f \"$f\" ] || continue; " +
				"kind=$(basename \"$f\" .json); name=$(basename \"$(dirname \"$f\")\"); count=$((count+1)); status=0; " +
				"/bin/validate-constraints -file \"$f\" -kind \"$kind\" -oscal-cli /opt/oscal-cli/bin/oscal-cli > \"/tmp/constraint-evidence/${name}-${kind}.log\" 2>&1 || status=$?; " +
				"cat \"/tmp/constraint-evidence/${name}-${kind}.log\"; sum=$(sha256sum \"$f\"); sum=${sum%% *}; printf '%s\\t%s\\t%s\\t%s\\n' \"${f#/frameworks/}\" \"$kind\" \"$status\" \"$sum\" >> /tmp/constraint-evidence/results.tsv; " +
				"if [ \"$status\" -ne 0 ]; then fail=1; fi; done; " +
				"if [ \"$count\" -eq 0 ]; then fail=1; fi; printf '%s\\n' \"$fail\" > /tmp/constraint-evidence/result.status"})
}
