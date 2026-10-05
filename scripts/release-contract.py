#!/usr/bin/env python3
"""Build and verify manifest subjects before any promotion side effect."""

import argparse
import json
import os
import re
import subprocess
from pathlib import Path

from attestation_policy import verification_flags
from release_inventory import create, digest, identity, required_payload, validate, write

REPOSITORY = "ckodex-labs/ckodex-xoscal"
WORKFLOW = REPOSITORY + "/.github/workflows/release.yml"


def verifier_diagnostic(output):
    """Retain verifier errors without exposing authentication material."""
    text = output or "(empty)"
    for name in ("GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"):
        secret = os.environ.get(name)
        if secret:
            text = text.replace(secret, "[REDACTED]")
    text = re.sub(r"(?i)(authorization\s*:\s*(?:bearer|token|basic)\s+)\S+", r"\1[REDACTED]", text)
    return re.sub(r"\b(?:gh[pousr]_[A-Za-z0-9]+|github_pat_[A-Za-z0-9_]+)\b", "[REDACTED]", text)


def verify_attestation(root, manifest_path, bundle, expected_tag, expected_revision):
    identity(expected_tag, expected_revision)
    document = json.loads(manifest_path.read_text())
    if document["release_tag"] != expected_tag or document["source_revision"] != expected_revision:
        raise ValueError("manifest differs from independently expected release identity")
    command = ["gh", "attestation", "verify", str(manifest_path)] + verification_flags(bundle, expected_tag, expected_revision)
    try:
        result = subprocess.run(command, check=True, capture_output=True, text=True)
    except subprocess.CalledProcessError as error:
        raise ValueError(
            f"gh attestation verify failed (exit {error.returncode})\n"
            f"stdout:\n{verifier_diagnostic(error.stdout)}\n"
            f"stderr:\n{verifier_diagnostic(error.stderr)}"
        ) from None
    verified = json.loads(result.stdout)
    if not isinstance(verified, list) or not verified:
        raise ValueError("verifier supplied no successful attestation")
    return {"schema_version": "xoscal-verification-v1", "manifest_digest": digest(manifest_path),
            "bundle_digest": digest(bundle), "release_tag": expected_tag,
            "source_revision": expected_revision, "repository": REPOSITORY,
            "workflow": WORKFLOW, "verifier": "gh attestation verify",
            "results": verified}


def build(args):
    document = create(args.root, args.tag, args.revision, args.final)
    if args.required:
        required_payload(args.root, document)
    name = "release-manifest.json" if args.final else "payload-manifest.json"
    write(args.root / name, document)
    print(f"release-contract: inventoried {len(document['artifacts'])} exact files ({name})")


def verify(args):
    name = "release-manifest.json" if args.final else "payload-manifest.json"
    manifest_path = args.root / name
    document = json.loads(manifest_path.read_text())
    validate(args.root, document, args.final)
    if args.required:
        required_payload(args.root, document)
    if not args.bundle or not args.tag or not args.revision:
        raise ValueError("signature verification requires bundle and independent expected tag/revision")
    receipt = verify_attestation(args.root, manifest_path, args.bundle, args.tag, args.revision)
    destination = "proofs/verification.json" if args.final else "proofs/payload-verification.json"
    write(args.root / destination, receipt)
    print(f"release-contract: verified signature and {len(document['artifacts'])} exact files")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=("create", "verify", "integrity"))
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--tag")
    parser.add_argument("--revision")
    parser.add_argument("--bundle", type=Path)
    parser.add_argument("--final", action="store_true")
    parser.add_argument("--required", action="store_true")
    args = parser.parse_args()
    if args.operation == "create":
        build(args)
    elif args.operation == "verify":
        verify(args)
    else:
        name = "release-manifest.json" if args.final else "payload-manifest.json"
        document = json.loads((args.root / name).read_text())
        validate(args.root, document, args.final)
        if args.required:
            required_payload(args.root, document)
        print("release-contract: exact-byte integrity passed; signature not checked")


if __name__ == "__main__":
    main()
