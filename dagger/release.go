package main

import (
	"fmt"
	"strings"

	"dagger/xoscal/internal/dagger"
)

func (m *Xoscal) Dist(source *dagger.Directory) *dagger.Directory {
	server := m.Build(source)
	ctl := m.BuildCtl(source)
	backup := m.BuildBackup(source)
	sbom := m.Sbom(source)
	return dag.Directory().
		WithFile("xoscal-server", server).
		WithFile("xoscal-ctl", ctl).
		WithFile("xoscal-backup", backup).
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
		"test -x /usr/local/bin/syft", syftLinuxAMD64SHA, syftLinuxARM64SHA, syftVersion, strings.TrimPrefix(syftVersion, "v"))
}

// goreleaser returns a GoReleaser container with source and build caches.
func (m *Xoscal) goreleaser(source *dagger.Directory) *dagger.Container {
	modCache := dag.CacheVolume("go-mod")
	buildCache := dag.CacheVolume("go-build")
	return m.goreleaserBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", modCache).
		WithMountedCache("/root/.cache/go-build", buildCache).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithEnvVariable("GOCACHE", "/root/.cache/go-build").
		WithEnvVariable("GOMAXPROCS", "2").
		WithEnvVariable("GOFLAGS", "-p=2 -mod=readonly").
		WithEnvVariable("GOWORK", "off").
		WithEnvVariable("CGO_ENABLED", "0")
}

// Release stages archives without publishing or signing. Hosted verification
// and promotion belong to the release workflow after every Dagger gate passes.
func (m *Xoscal) Release(source *dagger.Directory) *dagger.Directory {
	return m.goreleaser(source).
		WithExec([]string{"goreleaser", "release", "--parallelism", "2", "--clean", "--skip=publish,announce,sign"}).
		Directory("/src/dist")
}

// Snapshot stages development archives without publication or credentials.
func (m *Xoscal) Snapshot(source *dagger.Directory) *dagger.Directory {
	return m.goreleaser(source).WithEnvVariable("XOSCAL_SNAPSHOT_VERSION", strings.TrimPrefix(m.Version, "v")).
		WithExec([]string{"goreleaser", "release", "--parallelism", "2", "--clean", "--snapshot", "--skip=publish,announce,sign"}).
		Directory("/src/dist")
}
