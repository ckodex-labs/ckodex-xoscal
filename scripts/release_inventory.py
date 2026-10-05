#!/usr/bin/env python3
"""Exact-byte release inventory. No trust decisions derive from a JSON boolean."""

import hashlib
import json
import re
from pathlib import Path, PurePosixPath

SCHEMA = "xoscal-release-v1"
PROOF_ENVELOPE = {"proofs/release-manifest.sigstore.json", "proofs/verification.json"}
PAYLOAD_ROOTS = {"sdk", "frameworks", "release", "evidence", "scripts"}
PAYLOAD_FILES = {"openapi.json", "sbom-assessment-results.json", "scalar.js", "scalar.js.map", "scalar.css", "scalar.css.map", "THIRD-PARTY-NOTICES.txt"}
RESERVED_ASSETS = {"release-manifest.json", "release-manifest.sigstore.json", "release-site.tar.gz"}


def digest(path):
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            value.update(block)
    return "sha256:" + value.hexdigest()


def safe_path(value):
    path = PurePosixPath(value)
    if not value or path.is_absolute() or str(path) != value:
        raise ValueError(f"noncanonical artifact path: {value}")
    if any(part in (".", "..") for part in path.parts) or "\\" in value:
        raise ValueError(f"unsafe artifact path: {value}")
    return path


def files(root):
    result = set()
    for path in root.rglob("*"):
        if path.is_symlink():
            raise ValueError(f"symlinks prohibited: {path}")
        if path.is_file():
            result.add(path.relative_to(root).as_posix())
    return result


def identity(tag, revision):
    if not re.fullmatch(r"(?:v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?|dev)", tag):
        raise ValueError("release tag must be explicit semver or dev")
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("source revision must be a full Git SHA")


def metadata(path):
    name = PurePosixPath(path).name
    if path in ("scalar.js", "scalar.js.map", "scalar.css", "scalar.css.map", "THIRD-PARTY-NOTICES.txt"):
        return "frontend-runtime", "FrontendAssets", "component-definition"
    if path.startswith("evidence/frontend-sbom/"):
        return "analysis", "SbomArtifactAnalysis", "assessment-results"
    if path.startswith("evidence/frontend/"):
        return "analysis", "FrontendAssets", "assessment-results"
    if path == "payload-manifest.json":
        return "release-evidence", "ReleaseCandidate", "back-matter resource"
    if path == "proofs/payload.sigstore.json":
        return "attestation", "HostedAttestation", "back-matter resource"
    if path == "proofs/payload-verification.json":
        return "analysis", "FinalizeCandidate", "assessment-results"
    if path.startswith("frameworks/"):
        kind = "oscal-profile" if name == "profile.json" else "oscal-catalog" if name == "catalog.json" else "release-evidence"
        return kind, "OscalFrameworks", "profile" if kind.endswith("profile") else "catalog"
    if path.startswith("sdk/smoke/"):
        return "analysis", "SdkConsumer", "assessment-results"
    if path.startswith("sdk/"):
        kind = "sdk-source" if re.fullmatch(r"sdk/(go|python|java|csharp|ts|swift)\.zip", path) else "sdk-package" if path.startswith("sdk/packages/") else "sdk-metadata"
        return kind, "SdkPackages", "component-definition"
    if path.startswith("release/"):
        producer = "ImageArchive" if name in ("image.tar", "image-amd64.tar", "image-arm64.tar") else "InspectImageArchive" if name.startswith("image-") else "Build" if name == "xoscal-server" else "Release"
        return "cli-archive" if name.endswith(".tar.gz") else "release-evidence", producer, "back-matter resource"
    if path == "evidence/release-source.json":
        return "analysis", "ReleaseIdentity", "assessment-results"
    if path == "evidence/presentation.ok":
        return "analysis", "PresentationAnalysis", "assessment-results"
    if path.startswith("evidence/") or name.startswith("sbom-"):
        stage = PurePosixPath(path).parts[1] if path.startswith("evidence/") else "binary-sbom"
        if stage == "policy":
            return "release-policy", "ReleaseCandidate", "assessment-plan"
        producer = "All" if stage == "ci" else "SecurityAnalysis" if stage == "security" else "ImageSbomAnalysis" if stage.startswith("image-") and stage.endswith("-sbom") else "ImageAnalysis" if stage.startswith("image-") else "ArtifactSbomAnalysis" if stage.endswith("-sbom") else "ArtifactAnalysis"
        return "analysis", producer, "assessment-results"
    if path.startswith("scripts/"):
        return "installer" if path == "scripts/install.sh" else "release-evidence", "ReleaseCandidate", "back-matter resource"
    if path == "openapi.json":
        return "api-spec", "Openapi", "component-definition"
    return "site-asset", "FinalizeCandidate", "back-matter resource"


def analysis_refs(root, path, kind):
    refs = []
    if kind == "frontend-runtime":
        refs = ["evidence/frontend/analysis-gate.json", "evidence/frontend/frontend-inventory.json", "evidence/frontend/sbom.cyclonedx.json", "evidence/frontend-sbom/analysis-gate.json"]
    elif kind == "sdk-source":
        language = PurePosixPath(path).stem
        refs = [f"sdk/smoke/{language}.json", "sdk/sdk-contract-check.json",
                f"evidence/sdk-{language}/analysis-gate.json", f"evidence/sdk-{language}-sbom/analysis-gate.json"]
    elif kind == "sdk-package":
        language = PurePosixPath(path).parts[2]
        refs = [f"sdk/smoke/{language}.json", "sdk/sdk-contract-check.json"]
    elif kind == "cli-archive":
        match = re.search(r"_(Darwin|Linux)_(x86_64|arm64)\.tar\.gz$", path)
        if match:
            os_name, arch = match[1].lower(), "amd64" if match[2] == "x86_64" else "arm64"
            refs = [f"evidence/archive-{os_name}_{arch}/analysis-gate.json", f"evidence/archive-{os_name}_{arch}-sbom/analysis-gate.json"]
    elif path in ("release/image-amd64.tar", "release/image-arm64.tar"):
        arch = PurePosixPath(path).stem.removeprefix("image-")
        refs = [f"evidence/image-{arch}/analysis-gate.json", f"evidence/image-{arch}-sbom/analysis-gate.json"]
    elif path == "release/image.tar":
        refs = ["release/image-platforms.json", "evidence/image-amd64/analysis-gate.json", "evidence/image-arm64/analysis-gate.json"]
    elif path == "release/xoscal-server":
        refs = ["evidence/binary-sbom/analysis-gate.json", "evidence/security/analysis-gate.json"]
    elif kind in ("oscal-catalog", "oscal-profile"):
        refs = ["evidence/ci/schema.ok", "evidence/ci/constraints.ok"]
    elif kind == "api-spec":
        refs = ["evidence/ci/openapi.ok"]
    elif kind == "installer":
        refs = ["evidence/ci/release-contract.ok"]
    elif kind == "site-asset":
        refs = ["evidence/presentation.ok"]
    return [ref for ref in refs if (root / ref).is_file()]


def record(root, path):
    kind, producer, touchpoint = metadata(path)
    if path.endswith(".sha256"):
        kind = "checksum"
    item = {"path": path, "name": PurePosixPath(path).name, "kind": kind,
            "digest": digest(root / path), "size": (root / path).stat().st_size,
            "producer": producer, "oscal_touchpoint": touchpoint,
            "evidence_refs": analysis_refs(root, path, kind)}
    if kind == "cli-archive":
        match = re.search(r"_(Darwin|Linux)_(x86_64|arm64)\.tar\.gz$", path)
        if not match:
            raise ValueError(f"unknown CLI archive platform: {path}")
        item["platform"] = {"os": match[1].lower(), "arch": "amd64" if match[2] == "x86_64" else "arm64"}
    downloadable = {"cli-archive", "sdk-source", "sdk-package", "oscal-catalog", "oscal-profile", "api-spec", "installer", "checksum"}
    if kind in downloadable and (kind != "checksum" or path.startswith(("sdk/", "frameworks/", "release/"))):
        name = PurePosixPath(path).name
        if path.startswith("frameworks/"):
            name = PurePosixPath(path).parts[1] + "-" + name
        item["release_asset_name"] = name
    return item


def selected(root, final):
    names = files(root)
    if final:
        return names - PROOF_ENVELOPE - {"release-manifest.json"}
    return {p for p in names if PurePosixPath(p).parts[0] in PAYLOAD_ROOTS or p in PAYLOAD_FILES}


def create(root, tag, revision, final=False):
    identity(tag, revision)
    names = sorted(selected(root, final))
    if not names:
        raise ValueError("empty candidate")
    entries = [record(root, p) for p in names]
    check_asset_names(entries)
    return {"schema_version": SCHEMA, "release_tag": tag, "source_revision": revision,
            "scope": "final" if final else "payload", "artifacts": entries}


def check_asset_names(entries):
    seen = set()
    for item in entries:
        name = item.get("release_asset_name")
        if name is None:
            continue
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._+-]*", name) or name in RESERVED_ASSETS or name in seen:
            raise ValueError("duplicate, reserved or unsafe release attachment name: " + str(name))
        seen.add(name)


def validate(root, manifest, final=False):
    if manifest.get("schema_version") != SCHEMA or manifest.get("scope") != ("final" if final else "payload"):
        raise ValueError("unsupported manifest schema/scope")
    identity(manifest.get("release_tag", ""), manifest.get("source_revision", ""))
    entries = manifest.get("artifacts")
    if not isinstance(entries, list) or not entries:
        raise ValueError("missing artifacts")
    found = set()
    expected_paths = selected(root, final)
    check_asset_names(entries)
    for item in entries:
        path = str(safe_path(item["path"]))
        if path in found or path not in expected_paths:
            raise ValueError(f"duplicate, missing or out-of-scope artifact: {path}")
        found.add(path)
        canonical = record(root, path)
        for key in ("name", "kind", "producer", "oscal_touchpoint", "platform", "release_asset_name", "evidence_refs"):
            if item.get(key) != canonical.get(key):
                raise ValueError(f"noncanonical artifact metadata {key}: {path}")
        if item["digest"] != canonical["digest"] or type(item["size"]) is not int or item["size"] != canonical["size"]:
            raise ValueError(f"artifact integrity mismatch: {path}")
    if found != expected_paths:
        raise ValueError(f"uncovered files: {sorted(expected_paths - found)}")
    return entries


def required_payload(root, manifest):
    from release_requirements import required_payload as admit
    return admit(root, manifest)


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")
