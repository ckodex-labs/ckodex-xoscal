package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

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
		From(goBuilderImage).
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "git", "curl"}).
		WithExec([]string{"sh", "-c", pinnedGoReleaserInstall()}).
		WithExec([]string{"sh", "-c", pinnedSyftInstall()})
}

func pinnedBufInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=x86_64; expected=%s ;;\n"+
		"  aarch64|arm64) asset_arch=aarch64; expected=%s ;;\n"+
		"  *) echo \"unsupported Buf architecture: $(uname -m)\" >&2; exit 1 ;;\n"+"esac\n"+"archive=\"/tmp/buf\"\n"+"curl -fsSL \"https://github.com/bufbuild/buf/releases/download/%s/buf-Linux-${asset_arch}\" -o \"$archive\"\n"+"printf '%%s  %%s\\n' \"$expected\" \"$archive\" | sha256sum -c -\n"+"install -m 0755 \"$archive\" /usr/local/bin/buf\n"+"test -x /usr/local/bin/buf", bufLinuxAMD64SHA, bufLinuxARM64SHA, bufVersion)
}

func pinnedGoReleaserInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=x86_64; expected=a99bbc7ae0d8d897b07c4c497a9b62f222558804715ef219d1af05a7e417bc80 ;;\n"+
		"  aarch64|arm64) asset_arch=arm64; expected=702f03769ac8bcb0e47839c82243cc614ae995633599a98c63062e13ea85f829 ;;\n"+
		"  *) echo \"unsupported GoReleaser architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
		"esac\n"+
		"archive=\"/tmp/goreleaser.tar.gz\"\n"+
		"curl -fsSL \"https://github.com/goreleaser/goreleaser/releases/download/%s/goreleaser_Linux_${asset_arch}.tar.gz\" -o \"$archive\"\n"+
		"printf '%%s  %%s\\n' \"$expected\" \"$archive\" | sha256sum -c -\n"+
		"tar -xzf \"$archive\" -C /usr/local/bin goreleaser\n"+
		"test -x /usr/local/bin/goreleaser", goreleaserVersion)
}

func pinnedSyftInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=amd64; expected=%s ;;\n"+
		"  aarch64|arm64) asset_arch=arm64; expected=%s ;;\n"+
		"  *) echo \"unsupported Syft architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
		"esac\n"+
		"archive=\"/tmp/syft.tar.gz\"\n"+
		"curl -fsSL \"https://github.com/anchore/syft/releases/download/%s/syft_%s_linux_${asset_arch}.tar.gz\" -o \"$archive\"\n"+
		"printf '%%s  %%s\\n' \"$expected\" \"$archive\" | sha256sum -c -\n"+
		"tar -xzf \"$archive\" -C /usr/local/bin syft\n"+
		"test -x /usr/local/bin/syft", syftLinuxAMD64SHA, syftLinuxARM64SHA, syftVersion, syftVersion)
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
		WithSecretVariable("GITHUB_TOKEN", githubToken)
}

// Release runs GoReleaser release --clean (publishes GitHub release, archives, SBOMs).
// SLSA L3 provenance is generated separately by slsa-github-generator.
// Container image signing is done separately by cosign in the image job.
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

// All runs lint, test, race, security, proto-drift, spec-registry,
// OSCAL schema validation (tier 1, blocking), and OSCAL Metaschema constraint
// validation (tier 2, non-blocking) checks in parallel branches.
// Each branch shares the cached base container. The returned directory
// contains outputs from all parallel checks.
