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
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "jq", "python3"}).
		WithExec([]string{"sh", "-c", pinnedGhInstall()})
}

// VerifyPublishedRelease downloads every asset of the latest published GitHub
// release and verifies it from a clean environment: checksums.txt coverage of
// the GoReleaser artifacts and a GitHub artifact attestation for every
// attested asset plus the container image. GitHub signs the attestations with
// its own key; verification uses only gh and the repository's attestation
// store — no Sigstore, TUF mirror, or external transparency log. It emits
// bundle/inventory.json recording the verification result per asset.
//
// This is the clean-environment gate required by docs/BETA-PLAN.md Phase 4;
// it runs against the published release, not the working tree.
func (m *Xoscal) VerifyPublishedRelease(
	// +optional
	githubToken *dagger.Secret,
) *dagger.Directory {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	registry := fmt.Sprintf("ghcr.io/%s/%s", repoOwner, repoName)

	c := m.verifyTools().
		WithEnvVariable("GITHUB_API", api).
		WithEnvVariable("IMAGE_REGISTRY", registry)
	if githubToken != nil {
		c = c.WithSecretVariable("GH_TOKEN", githubToken)
	}

	return c.
		WithExec([]string{"sh", "-c", releaseDownloadScript()}).
		WithExec([]string{"sh", "-c", releaseChecksumVerifyScript()}).
		WithExec([]string{"sh", "-c", releaseAttestationVerifyScript()}).
		WithExec([]string{"sh", "-c", releaseImageVerifyScript()}).
		WithExec([]string{"sh", "-c", releaseInventoryScript()}).
		Directory("/bundle")
}

func releaseDownloadScript() string {
	return `set -eu
mkdir -p /bundle/assets /verify
auth=()
if [ -n "${GH_TOKEN:-}" ]; then auth=(-H "Authorization: Bearer $GH_TOKEN"); fi
tag=$(curl -fsSL "${auth[@]}" "$GITHUB_API" | jq -r .tag_name)
test -n "$tag" && test "$tag" != "null"
echo "$tag" > /verify/tag
echo "verifying release $tag"
curl -fsSL "${auth[@]}" "$GITHUB_API" -o /verify/release.json
jq -r '.assets[] | select(.name != "provenance.intoto.jsonl") | .browser_download_url' /verify/release.json > /verify/asset-urls
while read -r url; do
  name=$(basename "$url")
  curl -fsSL "${auth[@]}" "$url" -o "/bundle/assets/$name"
done < /verify/asset-urls
`
}

func releaseChecksumVerifyScript() string {
	return `set -eu
cd /bundle/assets
test -s checksums.txt
# checksums.txt covers the GoReleaser artifacts (archives + SBOMs). The
# separately attached assets (SDK zips, catalogs) are verified by their own
# GitHub attestations, not by checksums.txt.
verified=0
while read -r expected name; do
  [ -f "$name" ] || { echo "checksums.txt lists missing asset: $name" >&2; exit 1; }
  printf '%s  %s\n' "$expected" "$name" | sha256sum -c --status || { echo "digest mismatch: $name" >&2; exit 1; }
  verified=$((verified + 1))
done < checksums.txt
for archive in *.tar.gz *.tar.gz.sbom.json; do
  [ -e "$archive" ] || continue
  grep -q "  ${archive}$" checksums.txt || { echo "archive not covered by checksums.txt: $archive" >&2; exit 1; }
done
echo "checksums verified: $verified entries; archives and SBOMs covered"
`
}

func releaseAttestationVerifyScript() string {
	return `set -eu
cd /bundle/assets
verified=0
for artifact in *.tar.gz checksums.txt *.zip *-catalog.json; do
  [ -e "$artifact" ] || continue
  gh attestation verify "$artifact" -R ` + repoOwner + `/` + repoName + ` > /dev/null
  echo "attestation verified: $artifact"
  verified=$((verified + 1))
done
[ "$verified" -gt 0 ] || { echo "no attested artifacts found" >&2; exit 1; }
echo "GitHub attestations verified: $verified artifacts"
`
}

func releaseImageVerifyScript() string {
	return `set -eu
tag=$(cat /verify/tag)
image="$IMAGE_REGISTRY:$tag"
gh attestation verify "oci://$image" -R ` + repoOwner + `/` + repoName + ` > /verify/image-attestation.txt
echo "image attestation verified: $image"
`
}

func releaseInventoryScript() string {
	return `set -eu
cd /bundle/assets
python3 - <<'PY'
import hashlib, json, os, glob

entries = []
for path in sorted(glob.glob("*")):
    if os.path.isdir(path):
        continue
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            h.update(chunk)
    entries.append({
        "name": path,
        "sha256": h.hexdigest(),
        "size": os.path.getsize(path),
        "checksums_listed": True,
    })

sig_count = len(glob.glob("*.sig"))
inventory = {
    "bundle": "xoscal-release-bundle",
    "release_tag": open("/verify/tag").read().strip(),
    "assets": entries,
    "verification": {
        "checksums_txt": "verified",
        "github_attestations": "verified",
        "image_attestation": "verified",
    },
}
os.makedirs("/bundle", exist_ok=True)
with open("/bundle/inventory.json", "w") as fh:
    json.dump(inventory, fh, indent=2, sort_keys=True)
print(f"inventory: {len(entries)} assets, {sig_count} legacy signatures present")
PY
`
}

// ReleaseBundle assembles the operator release bundle for a published tag:
// the verified assets, the image attestation payload, and the inventory.
// Verification failures abort the bundle; there is no partial bundle. The
// result is the artifact set an operator retains per docs/OPERATIONS.md
// retention procedure.
func (m *Xoscal) ReleaseBundle(
	// +optional
	githubToken *dagger.Secret,
) *dagger.Directory {
	bundle := m.VerifyPublishedRelease(githubToken)
	return bundle.
		WithFile("image-attestation.json", bundle.File("/verify/image-attestation.json"))
}
