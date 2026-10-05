package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

const (
	ghCLIVersion    = "v2.100.0"
	ghLinuxAMD64SHA = "e4d4bb4498e8d007abe545b6568926793ace1b6447da598294a610018cb164be"
	ghLinuxARM64SHA = "ea4e7a581a32ccad6cc7923cb1576ac5859ba4b9a16ab22eb8f8a96e78e2e961"
	repoOwner       = "ckodex-labs"
	repoName        = "ckodex-xoscal"
)

func pinnedGhInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=amd64; expected=%s ;;\n"+
		"  aarch64|arm64) asset_arch=arm64; expected=%s ;;\n"+
		"  *) echo \"unsupported gh architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
		"esac\n"+
		"curl -fsSL \"https://github.com/cli/cli/releases/download/%s/gh_2.100.0_linux_${asset_arch}.tar.gz\" -o /tmp/gh.tar.gz\n"+
		"printf '%%s  %%s\\n' \"$expected\" /tmp/gh.tar.gz | sha256sum -c -\n"+
		"tar -xzf /tmp/gh.tar.gz -C /tmp\n"+
		"install -m 0755 \"/tmp/gh_2.100.0_linux_${asset_arch}/bin/gh\" /usr/local/bin/gh\n"+
		"gh --version", ghLinuxAMD64SHA, ghLinuxARM64SHA, ghCLIVersion)
}

// verifyTools returns a container with the pinned gh CLI for attestation
// verification, plus curl/jq/python3 for release-asset plumbing.
func (m *Xoscal) verifyTools() *dagger.Container {
	aptCache := dag.CacheVolume("apt-cache")
	aptLists := dag.CacheVolume("apt-lists")
	return dag.Container().
		From(ubuntuBase).
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "jq", "python3", "git"}).
		WithExec([]string{"sh", "-c", pinnedGhInstall()})
}

// VerifyPublishedRelease retrieves one independently selected tag and revision,
// verifies signed transport before extraction, then validates every final byte.
func (m *Xoscal) VerifyPublishedRelease(source *dagger.Directory, releaseTag string, sourceRevision string) *dagger.Directory {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").
		WithExec([]string{"python3", "scripts/download-release.py", "--tag", releaseTag, "--revision", sourceRevision, "--root", "/out"}).Directory("/out")
}

// ReleaseBundle is the immutable operator retention bundle for an explicit release.
func (m *Xoscal) ReleaseBundle(source *dagger.Directory, releaseTag string, sourceRevision string) *dagger.Directory {
	return m.VerifyPublishedRelease(source, releaseTag, sourceRevision)
}

// PackageCandidate serializes frozen site bytes before hosted attestation of
// the transport and final manifest. It does not add its own digest to that manifest.
func (m *Xoscal) PackageCandidate(source *dagger.Directory, candidate *dagger.Directory) *dagger.File {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").WithDirectory("/candidate", candidate).
		WithExec([]string{"python3", "scripts/package-candidate.py", "/candidate", "/release-site.tar.gz"}).File("/release-site.tar.gz")
}

// VerifyTransport reuses the exact hosted identity contract before handing
// a separately signed transport to a consumer or promotion boundary.
func (m *Xoscal) VerifyTransport(source *dagger.Directory, transport *dagger.File, bundle *dagger.File, releaseTag, sourceRevision string) *dagger.File {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").
		WithFile("/input/transport.tar.gz", transport).WithFile("/input/bundle.json", bundle).
		WithExec([]string{"python3", "scripts/verify-transport.py", "/input/transport.tar.gz", "/input/bundle.json", releaseTag, sourceRevision, "/transport-verification.json"}).File("/transport-verification.json")
}
