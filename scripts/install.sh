#!/bin/sh
# Reviewed-source bootstrap. Release provenance is verified before archive parsing.
set -eu
command -v python3 >/dev/null 2>&1 || { echo 'install: Python 3 is required' >&2; exit 1; }
exec python3 - "$@" <<'PY'
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import urllib.request

REPO = "ckodex-labs/ckodex-xoscal"
WORKFLOW = REPO + "/.github/workflows/release.yml"
MAX_ARCHIVE = 1024 * 1024 * 1024
MAX_BINARY = 512 * 1024 * 1024

def fail(message):
    raise ValueError(message)

def unique_object(pairs):
    obj = {}
    for key, value in pairs:
        if key in obj:
            fail("duplicate JSON key: " + key)
        obj[key] = value
    return obj

def safe_path(value):
    if not isinstance(value, str) or not value or "\\" in value:
        fail("invalid artifact path")
    path = PurePosixPath(value)
    if path.is_absolute() or any(p in ("", ".", "..") for p in value.split("/")):
        fail("unsafe artifact path: " + value)
    return path

def bounded_copy(source, target, limit):
    count = 0
    with open(target, "wb") as out:
        while True:
            block = source.read(1024 * 1024)
            if not block:
                break
            count += len(block)
            if count > limit:
                fail("input exceeds size limit")
            out.write(block)

def main():
    parser = argparse.ArgumentParser(description="Install xoscal-ctl from an explicitly pinned, attested release. Never uses latest or unsigned fallback.")
    parser.add_argument("--release", required=True, help="release tag, e.g. v1.2.3")
    parser.add_argument("--source", required=True, help="expected full 40-character source commit SHA")
    parser.add_argument("--manifest", required=True, type=Path, help="local release-manifest.json")
    parser.add_argument("--bundle", required=True, type=Path, help="local proofs/release-manifest.sigstore.json")
    parser.add_argument("--archive", type=Path, help="local GoReleaser archive; otherwise download from the pinned GitHub release")
    parser.add_argument("--prefix", type=Path, default=Path.home() / ".local", help="install under PREFIX/bin (default ~/.local)")
    parser.add_argument("--dry-run", action="store_true", help="verify provenance, bytes and archive structure without installing")
    parser.add_argument("--force", action="store_true", help="replace an existing xoscal-ctl after verification")
    args = parser.parse_args()
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", args.release):
        fail("--release must be an explicit vMAJOR.MINOR.PATCH tag (optional prerelease)")
    if not re.fullmatch(r"[0-9a-f]{40}", args.source):
        fail("--source must be a full lowercase commit SHA")
    os_name = {"Darwin": "darwin", "Linux": "linux"}.get(platform.system())
    arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine().lower())
    if os_name is None or arch is None:
        fail("supported platforms: Linux/macOS AMD64/ARM64; no universal binary")
    if shutil.which("gh") is None:
        fail("GitHub CLI with gh attestation verify is required")
    if not args.manifest.is_file() or not args.bundle.is_file():
        fail("manifest and signed provenance bundle are required local files")

    # Copy inputs into a private directory to avoid verify/use races on caller files.
    with tempfile.TemporaryDirectory(prefix="xoscal-install-") as work:
        work = Path(work)
        manifest = work / "release-manifest.json"
        bundle = work / "release-manifest.sigstore.json"
        for source, target in ((args.manifest, manifest), (args.bundle, bundle)):
            with source.open("rb") as stream:
                bounded_copy(stream, target, 32 * 1024 * 1024)
        command = ["gh", "attestation", "verify", str(manifest), "--bundle", str(bundle),
                   "--hostname", "github.com", "--repo", REPO,
                   "--signer-digest", args.source,
                   "--source-digest", args.source, "--source-ref", "refs/tags/" + args.release,
                   "--cert-identity", "https://github.com/" + WORKFLOW + "@refs/tags/" + args.release,
                   "--cert-oidc-issuer", "https://token.actions.githubusercontent.com",
                   "--predicate-type", "https://slsa.dev/provenance/v1", "--deny-self-hosted-runners"]
        result = subprocess.run(command, check=False)
        if result.returncode:
            fail("release manifest provenance verification failed; nothing installed")
        data = json.loads(manifest.read_text(encoding="utf-8"), object_pairs_hook=unique_object)
        if not isinstance(data, dict):
            fail("manifest must be a JSON object")
        if data.get("scope") != "final":
            fail("installer requires the final admitted release manifest")
        if data.get("schema_version") != "xoscal-release-v1":
            fail("unsupported release manifest schema")
        if data.get("release_tag") != args.release or data.get("source_revision") != args.source:
            fail("manifest does not match the requested release and source")
        artifacts = data.get("artifacts")
        if not isinstance(artifacts, list):
            fail("manifest artifacts must be an array")
        seen = set()
        matches = []
        for item in artifacts:
            if not isinstance(item, dict):
                fail("malformed artifact entry")
            path = safe_path(item.get("path"))
            if str(path) in seen:
                fail("duplicate artifact path: " + str(path))
            seen.add(str(path))
            if item.get("kind") == "cli-archive" and item.get("platform") == {"os": os_name, "arch": arch}:
                matches.append(item)
        if len(matches) != 1:
            fail("manifest must contain exactly one CLI archive for this platform")
        artifact = matches[0]
        path = safe_path(artifact["path"])
        suffix = "_" + args.release[1:] + "_" + os_name.title() + "_" + ("x86_64" if arch == "amd64" else "arm64") + ".tar.gz"
        if not path.name.endswith(suffix):
            fail("archive filename does not match the GoReleaser release/platform convention")
        digest = artifact.get("digest", "")
        size = artifact.get("size")
        if not isinstance(digest, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
            fail("CLI archive requires a SHA-256 manifest digest")
        if type(size) is not int or size <= 0 or size > MAX_ARCHIVE:
            fail("invalid CLI archive size")
        archive = work / "archive.tar.gz"
        if args.archive:
            with args.archive.open("rb") as stream:
                bounded_copy(stream, archive, size)
        else:
            url = "https://github.com/" + REPO + "/releases/download/" + args.release + "/" + path.name
            with urllib.request.urlopen(url, timeout=60) as response:
                if not response.geturl().startswith("https://"):
                    fail("archive download redirected outside HTTPS")
                bounded_copy(response, archive, size)
        actual = hashlib.sha256()
        with archive.open("rb") as stream:
            for block in iter(lambda: stream.read(1024 * 1024), b""):
                actual.update(block)
        if archive.stat().st_size != size or "sha256:" + actual.hexdigest() != digest:
            fail("archive digest/size mismatch; nothing installed")
        # Do not extractall: validate all members, then copy the one regular binary.
        with tarfile.open(archive, mode="r:gz") as package:
            binary = None
            names = set()
            total = 0
            for member in package:
                name = member.name.removeprefix("./")
                safe_path(name)
                if name in names or not (member.isfile() or member.isdir()):
                    fail("archive contains a duplicate, link, device or unsupported member")
                names.add(name)
                total += member.size
                if total > MAX_ARCHIVE:
                    fail("archive expands beyond the size limit")
                if name == "xoscal-ctl" and member.isfile():
                    if member.size <= 0 or member.size > MAX_BINARY:
                        fail("invalid xoscal-ctl binary size")
                    binary = member
            if binary is None:
                fail("archive is missing the regular xoscal-ctl binary")
            if args.dry_run:
                print("Verified release " + args.release + " at " + args.source + "; archive " + path.name + "; no installation requested")
                return
            bindir = args.prefix.expanduser().absolute() / "bin"
            bindir.mkdir(parents=True, exist_ok=True)
            destination = bindir / "xoscal-ctl"
            if os.path.lexists(destination) and not args.force:
                fail(str(destination) + " already exists; choose another --prefix or use --force")
            descriptor, temporary = tempfile.mkstemp(prefix=".xoscal-ctl-", dir=bindir)
            try:
                with os.fdopen(descriptor, "wb") as out, package.extractfile(binary) as stream:
                    shutil.copyfileobj(stream, out)
                    out.flush()
                    os.fsync(out.fileno())
                os.chmod(temporary, 0o755)
                if args.force:
                    os.replace(temporary, destination)
                else:
                    # Atomic no-clobber install, including a concurrent existing target.
                    os.link(temporary, destination)
                    os.unlink(temporary)
                print("Installed verified xoscal-ctl to " + str(destination))
            finally:
                if os.path.exists(temporary):
                    os.unlink(temporary)

try:
    main()
except (ValueError, OSError, tarfile.TarError, json.JSONDecodeError) as error:
    print("install: " + str(error), file=sys.stderr)
    sys.exit(1)
PY
