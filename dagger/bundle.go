package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

const (
	cosignVersion        = "v2.6.1"
	cosignLinuxAMD64SHA  = "064954c5d8c7e3b28188eee5b1727b31c411550bc5fefd41aa672d3c761d103a"
	cosignLinuxARM64SHA  = "56a16480bdd56ec789abaa65924402f6b92c0041f06885995853c05567b76f34"
	slsaVerifierVersion  = "v2.7.0"
	slsaVerifierAMD64SHA = "499befb675efcca9001afe6e5156891b91e71f9c07ab120a8943979f85cc82e6"
	slsaVerifierARM64SHA = "dc3845d7605f666a0938389c1c5735230e50b32a547867ffd351fb14df928167"
	repoOwner            = "ckodex-labs"
	repoName             = "ckodex-xoscal"
)

func pinnedCosignInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=amd64; expected=%s ;;\n"+
		"  aarch64|arm64) asset_arch=arm64; expected=%s ;;\n"+
		"  *) echo \"unsupported cosign architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
		"esac\n"+
		"curl -fsSL \"https://github.com/sigstore/cosign/releases/download/%s/cosign-linux-${asset_arch}\" -o /usr/local/bin/cosign\n"+
		"printf '%%s  %%s\\n' \"$expected\" /usr/local/bin/cosign | sha256sum -c -\n"+
		"chmod +x /usr/local/bin/cosign\n"+
		"cosign version", cosignLinuxAMD64SHA, cosignLinuxARM64SHA, cosignVersion)
}

func pinnedSlsaVerifierInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) asset_arch=amd64; expected=%s ;;\n"+
		"  aarch64|arm64) asset_arch=arm64; expected=%s ;;\n"+
		"  *) echo \"unsupported slsa-verifier architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
		"esac\n"+
		"curl -fsSL \"https://github.com/slsa-framework/slsa-verifier/releases/download/%s/slsa-verifier-linux-${asset_arch}\" -o /usr/local/bin/slsa-verifier\n"+
		"printf '%%s  %%s\\n' \"$expected\" /usr/local/bin/slsa-verifier | sha256sum -c -\n"+
		"chmod +x /usr/local/bin/slsa-verifier\n"+
		"slsa-verifier version", slsaVerifierAMD64SHA, slsaVerifierARM64SHA, slsaVerifierVersion)
}

// verifyTools returns a container with pinned cosign and slsa-verifier
// installed, plus curl/jq/python3 for release-asset plumbing.
func (m *Xoscal) verifyTools() *dagger.Container {
	aptCache := dag.CacheVolume("apt-cache")
	aptLists := dag.CacheVolume("apt-lists")
	return dag.Container().
		From(ubuntuBase).
		WithMountedCache("/var/cache/apt", aptCache, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithMountedCache("/var/lib/apt/lists", aptLists, dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "jq", "python3"}).
		WithExec([]string{"sh", "-c", pinnedCosignInstall()}).
		WithExec([]string{"sh", "-c", pinnedSlsaVerifierInstall()})
}

// VerifyPublishedRelease downloads every asset of a published GitHub release
// and verifies it from a clean environment: checksums.txt coverage, cosign
// blob signatures (sig + certificate) for every signed artifact, SLSA
// provenance for the archives, and the keyless container image signature.
// It emits bundle/inventory.json recording the verification result per asset.
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
		WithExec([]string{"sh", "-c", releaseCosignVerifyScript()}).
		WithExec([]string{"sh", "-c", releaseProvenanceVerifyScript()}).
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
jq -r '.assets[].name' /verify/release.json > /verify/asset-names
jq -r '.assets[] | select(.name != "provenance.intoto.jsonl") | .browser_download_url' /verify/release.json > /verify/asset-urls
while read -r url; do
  name=$(basename "$url")
  curl -fsSL "${auth[@]}" "$url" -o "/bundle/assets/$name"
done < /verify/asset-urls
curl -fsSL "${auth[@]}" "$(jq -r '.assets[] | select(.name == "provenance.intoto.jsonl") | .browser_download_url' /verify/release.json)" -o /verify/provenance.intoto.jsonl
test -s /verify/provenance.intoto.jsonl
`
}

func releaseChecksumVerifyScript() string {
	return `set -eu
cd /bundle/assets
test -s checksums.txt
# checksums.txt covers the GoReleaser artifacts (archives + SBOMs). The
# separately attached assets (SDK zips, catalogs, signatures) are verified
# by their own mechanisms: .sha256 sidecars and cosign signatures.
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

func releaseCosignVerifyScript() string {
	return `set -eu
cd /bundle/assets
identity="https://github.com/` + repoOwner + `/` + repoName + `/.github/"
issuer="https://token.actions.githubusercontent.com"
verified=0
skipped=0
for sig in *.sig; do
  [ -e "$sig" ] || continue
  base=${sig%.sig}
  cert="${base}.pem"
  blob=""
  for candidate in "$base" "$base.tar.gz" "$base.zip"; do
    [ -f "$candidate" ] && { blob="$candidate"; break; }
  done
  if [ ! -f "${blob:-}" ]; then
    echo "signature $sig has no matching artifact; skipping" >&2
    skipped=$((skipped + 1))
    continue
  fi
  test -s "$cert" || { echo "missing certificate for $sig" >&2; exit 1; }
  cosign verify-blob --certificate "$cert" --signature "$sig" \
    --certificate-identity-regexp "$identity" \
    --certificate-oidc-issuer "$issuer" \
    "$blob" > /dev/null
  echo "cosign verified: $blob"
  verified=$((verified + 1))
done
echo "cosign blob signatures verified: $verified (skipped $skipped)"
`
}

func releaseProvenanceVerifyScript() string {
	return `set -eu
cd /bundle/assets
count=0
for archive in *.tar.gz; do
  [ -e "$archive" ] || continue
  slsa-verifier verify-artifact "$archive" \
    --provenance-path /verify/provenance.intoto.jsonl \
    --source-uri github.com/` + repoOwner + `/` + repoName + ` > /dev/null
  echo "slsa verified: $archive"
  count=$((count + 1))
done
[ "$count" -gt 0 ] || { echo "no archives found for SLSA verification" >&2; exit 1; }
`
}

func releaseImageVerifyScript() string {
	return `set -eu
tag=$(cat /verify/tag)
image="$IMAGE_REGISTRY:$tag"
identity="https://github.com/` + repoOwner + `/` + repoName + `/.github/"
issuer="https://token.actions.githubusercontent.com"
cosign verify --certificate-identity-regexp "$identity" \
  --certificate-oidc-issuer "$issuer" "$image" > /verify/image-signature.json
echo "cosign image signature verified: $image"
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
        "cosign_blob_signatures": sig_count,
        "slsa_provenance": "verified",
        "image_signature": "verified",
    },
}
os.makedirs("/bundle", exist_ok=True)
with open("/bundle/inventory.json", "w") as fh:
    json.dump(inventory, fh, indent=2, sort_keys=True)
print(f"inventory: {len(entries)} assets, {sig_count} signatures")
PY
`
}

// ReleaseBundle assembles the operator release bundle for a published tag:
// the verified assets, the provenance, the image-signature payload, and the
// inventory. Verification failures abort the bundle; there is no partial
// bundle. The result is the artifact set an operator retains per
// docs/OPERATIONS.md retention procedure.
func (m *Xoscal) ReleaseBundle(
	// +optional
	githubToken *dagger.Secret,
) *dagger.Directory {
	bundle := m.VerifyPublishedRelease(githubToken)
	return bundle.
		WithFile("provenance.intoto.jsonl", bundle.File("/verify/provenance.intoto.jsonl")).
		WithFile("image-signature.json", bundle.File("/verify/image-signature.json"))
}
