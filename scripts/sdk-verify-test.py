#!/usr/bin/env python3
"""Negative subject-integrity gates using actual exported Dagger SDK outputs."""
import argparse
import hashlib
import importlib.util
import json
import shutil
import tempfile
from pathlib import Path

spec = importlib.util.spec_from_file_location("sdk_verify", Path(__file__).with_name("sdk-verify.py"))
sdk = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sdk)


def must_fail(root, language, description):
    try:
        sdk.verify(root, [language])
    except (ValueError, FileNotFoundError, json.JSONDecodeError):
        print("PASS: " + description)
        return
    raise AssertionError(description + " was admitted")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--language", choices=sdk.LANGUAGES, default="python")
    args = parser.parse_args()
    sdk.verify(args.directory, [args.language])
    print("PASS: original actual SDK bytes")
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp) / "sdk"
        shutil.copytree(args.directory, root)
        version_path = root / "sdk-version.txt"
        original_version = version_path.read_bytes()
        version_path.write_text("0.0.0-different-candidate\n")
        must_fail(root, args.language, "mixed candidate release version")
        version_path.write_bytes(original_version)
        archive = root / (args.language + ".zip")
        original = archive.read_bytes()
        archive.write_bytes(original + b"tampered")
        must_fail(root, args.language, "tampered archive")
        # An attacker updating the unsigned sidecar still cannot rewrite the
        # subject of the existing consumer gate.
        sidecar = root / (args.language + ".zip.sha256")
        original_sidecar = sidecar.read_bytes()
        sidecar.write_text("sha256:" + hashlib.sha256(archive.read_bytes()).hexdigest() + "\n")
        must_fail(root, args.language, "tampered archive with rewritten sidecar")
        archive.write_bytes(original)
        sidecar.write_bytes(original_sidecar)
        report_path = root / "smoke" / (args.language + ".json")
        report_bytes = report_path.read_bytes()
        report = json.loads(report_bytes)
        report["status"] = "incomplete"
        report_path.write_text(json.dumps(report))
        must_fail(root, args.language, "incomplete consumer verdict")
        report_path.unlink()
        must_fail(root, args.language, "missing consumer report")
        report_path.write_bytes(report_bytes)
        native = [s for s in json.loads(report_bytes)["subjects"] if s["path"].startswith("packages/")]
        if native:
            package = root / native[0]["path"]
            package.write_bytes(package.read_bytes() + b"tampered")
            must_fail(root, args.language, "tampered native package")


if __name__ == "__main__":
    main()
