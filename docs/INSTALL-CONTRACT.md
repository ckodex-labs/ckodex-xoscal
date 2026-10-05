# Pinned CLI installation contract

`scripts/install.sh` installs **xoscal-ctl** from a reviewed source checkout and a
specific promoted release. It requires Python 3.9+ and a GitHub CLI supporting
`gh attestation verify` with the source, signer, certificate and runner policy
flags used by the script. It does not request new credentials or signing keys.
GitHub CLI may need its existing network/authentication configuration to obtain
trusted verification material. An unavailable verifier is a failure, not a bypass.

Supported release targets are Linux and macOS, each with AMD64 or ARM64 archives.
There is no universal macOS binary or Windows installer. GoReleaser creates
`<ProjectName>_<Version>_<Linux|Darwin>_<x86_64|arm64>.tar.gz`, containing
`xoscal-ctl`, `xoscal-server`, and `xoscal-backup`. The installer selects the
manifest's `cli-archive` entry for the running OS/architecture and installs only
`xoscal-ctl`; the other binaries remain available in the verified archive.
Package-manager SDK installation is a separate contract, not part of this script.

## Consumer command

Choose a release tag and its full source commit from independently reviewed
release information. Set `RELEASE_TAG` and `SOURCE_REVISION` explicitly; neither
has a `latest`, `main`, or unsigned default. Use `scripts/install.sh` from that
reviewed source revision. Do not execute a script fetched from mutable `main`.

```sh
# In a reviewed checkout of SOURCE_REVISION, fetch this release's detached proof.
curl --proto '=https' --tlsv1.2 -fSL \
  "https://github.com/ckodex-labs/ckodex-xoscal/releases/download/${RELEASE_TAG}/release-manifest.json" \
  -o release-manifest.json
curl --proto '=https' --tlsv1.2 -fSL \
  "https://github.com/ckodex-labs/ckodex-xoscal/releases/download/${RELEASE_TAG}/release-manifest.sigstore.json" \
  -o release-manifest.sigstore.json
sh scripts/install.sh \
  --release "$RELEASE_TAG" --source "$SOURCE_REVISION" \
  --manifest release-manifest.json --bundle release-manifest.sigstore.json \
  --prefix "$HOME/.local" --dry-run

# Omit --dry-run to install after the same checks.
```

Add the selected prefix's `bin` directory to your shell's PATH, then run
`xoscal-ctl help`. The installer never invokes sudo or executes the downloaded
binary. An existing destination is preserved unless `--force` is explicit.
Installation uses an atomic no-clobber operation by default.

For a staged candidate or offline archive, add `--archive /path/to/archive.tar.gz`.
The manifest and bundle are still mandatory. A local archive alone is not proof.
`--dry-run` performs the same provenance, digest, size and archive-structure gates
without creating the install directory. Truly disconnected verification also
requires GitHub CLI's trusted-root material to be available; the script does not
invent or substitute a root of trust.

## Admission and byte binding

1. Copy supplied manifest/proof into a private temporary directory.
2. Verify the **exact manifest bytes** using GitHub CLI's SLSA v1 predicate,
   GitHub OIDC issuer, `ckodex-labs/ckodex-xoscal`,
   `.github/workflows/release.yml`, the requested tag ref and full source SHA.
   Pin signer digest to the same SHA and reject self-hosted runner attestations.
3. Require `schema_version: xoscal-release-v1`, matching `release_tag` and
   `source_revision`, unique safe artifact paths, and exactly one `cli-archive`
   entry with `platform: {os: linux|darwin, arch: amd64|arm64}`.
4. Copy/download that archive into the private directory and compare its size
   and SHA-256 to the verified manifest. Signed manifest coverage binds the
   archive transitively; a checksum file alone is insufficient.
5. Validate every archive member before copying the regular top-level
   `xoscal-ctl` entry. Reject traversal, absolute paths, duplicate entries,
   symlinks, hardlinks, devices and excessive expansion. Never use `extractall`.
6. Atomically install only after every gate succeeds. Remove temporary inputs
   on success or failure; retain any pre-existing installed executable on failure.

## Dagger and hosted release integration

Dagger must stage the reviewed installer under `scripts/install.sh`, the complete
platform archives under `release/`, and cover the installer, archives and final
site bytes in `release-manifest.json`. Archive entries must include the fields
above plus the inventory's producer, OSCAL touchpoint and evidence references.
No archive or site mutation is permitted after the manifest is finalized.

Hosted Actions signs the finalized manifest using the existing release workflow
identity. The detached envelope is `proofs/release-manifest.sigstore.json`; it
is not included in its own manifest. Publish the manifest and the proof bundle
as GitHub release assets named `release-manifest.json` and
`release-manifest.sigstore.json`. Publish archive assets under their basenames,
preserving the exact Dagger bytes. All consumer-visible promotion waits for
hosted verification of the candidate; missing proofs never allow an unsigned
fallback. A Pages copy uses the same inventory's relative `release/` paths.

Run `python3 scripts/test-install.py` in Dagger's installer validation stage.
These local fixture tests exercise identity-policy arguments, digest binding,
safe archive handling, pinned inputs, refusal and atomic installation. They
deliberately use a simulated `gh` verifier and establish **no hosted signing
claim**. A hosted release must separately run the real verifier and the installer
against its actual signed manifest and actual platform archive bytes. Run the
installed binary's `help` and `version` in the corresponding clean target
environment. No such hosted release is executed by this change.
