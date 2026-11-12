#!/usr/bin/env python3
"""Build or verify deterministic embedded_catalogs.tar.gz from manifest.yaml and OSCAL catalogs."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import json
import os
import re
import sys
import tarfile
from pathlib import Path


def _unquote_yaml(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and ((value[0] == '"' and value[-1] == '"') or (value[0] == "'" and value[-1] == "'")):
        if value[0] == '"':
            return json.loads(value)
        return value[1:-1]
    return value


def parse_manifest_frameworks(manifest_path: Path) -> list[dict]:
    """Parse entries under `frameworks:` in data/frameworks/manifest.yaml using stdlib only."""
    entries: list[dict] = []
    current: dict | None = None
    in_frameworks = False
    for raw_line in manifest_path.read_text(encoding="utf-8").splitlines():
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if stripped == "frameworks:":
            in_frameworks = True
            continue
        if not in_frameworks:
            continue
        if not raw_line.startswith(" ") and not raw_line.startswith("-"):
            break
        if stripped.startswith("- "):
            if current is not None:
                entries.append(current)
            current = {}
            stripped = stripped[2:].strip()
        if current is None or ":" not in stripped:
            continue
        key, val = stripped.split(":", 1)
        key = key.strip()
        val = val.strip()
        if key == "aliases":
            if val.startswith("[") and val.endswith("]"):
                current[key] = json.loads(val)
            else:
                current[key] = [_unquote_yaml(val)] if val else []
        else:
            current[key] = _unquote_yaml(val)
    if current is not None:
        entries.append(current)
    return entries


def count_controls(obj: dict) -> int:
    total = 0
    for ctrl in obj.get("controls") or []:
        total += 1 + count_controls(ctrl)
    for grp in obj.get("groups") or []:
        total += count_controls(grp)
    return total


def _add_tar_bytes(tf: tarfile.TarFile, arcname: str, data: bytes) -> None:
    info = tarfile.TarInfo(name=arcname)
    info.size = len(data)
    info.mtime = 0
    info.uid = 0
    info.gid = 0
    info.uname = ""
    info.gname = ""
    info.mode = 0o644
    info.type = tarfile.REGTYPE
    tf.addfile(info, io.BytesIO(data))


def build_archive_bytes(repo_root: Path) -> tuple[bytes, int]:
    manifest_path = repo_root / "data" / "frameworks" / "manifest.yaml"
    data_dir = repo_root / "data" / "frameworks"

    entries = [
        e for e in parse_manifest_frameworks(manifest_path)
        if e.get("jurisdiction") != "canada" and str(e.get("filename", "")).endswith(".yaml")
    ]
    entries.sort(key=lambda e: e["ref_id"])

    index_specs: list[dict] = []
    catalog_blobs: list[tuple[str, bytes]] = []
    for entry in entries:
        fw_id = entry["ref_id"]
        if not re.fullmatch(r"[A-Za-z0-9_.-]+", fw_id):
            raise ValueError(f"invalid framework ID: {fw_id!r}")
        cat_path = data_dir / fw_id / "catalog.json"
        if not cat_path.is_file():
            raise FileNotFoundError(f"missing catalog: {cat_path}")
        raw = cat_path.read_bytes()
        if not raw:
            raise ValueError(f"empty catalog: {cat_path}")
        catalog = json.loads(raw).get("catalog")
        if not isinstance(catalog, dict):
            raise ValueError(f"invalid OSCAL catalog root in {cat_path}")
        metadata = catalog.get("metadata") or {}
        spec = {
            "id": fw_id,
            "aliases": list(entry.get("aliases") or []),
            "title": str(metadata.get("title") or fw_id),
            "version": str(metadata.get("version") or "1"),
            "jurisdiction": str(entry.get("jurisdiction") or ""),
            "authority": str(entry.get("authority") or ""),
            "control_count": count_controls(catalog),
        }
        index_specs.append(spec)
        catalog_blobs.append((f"{fw_id}/catalog.json", raw))

    index_bytes = (json.dumps(index_specs, indent=2, ensure_ascii=False, sort_keys=True) + "\n").encode("utf-8")

    tar_buf = io.BytesIO()
    with tarfile.open(fileobj=tar_buf, mode="w", format=tarfile.PAX_FORMAT) as tf:
        _add_tar_bytes(tf, "index.json", index_bytes)
        for arcname, raw in catalog_blobs:
            _add_tar_bytes(tf, arcname, raw)

    gz_buf = io.BytesIO()
    with gzip.GzipFile(filename="", mode="wb", fileobj=gz_buf, compresslevel=9, mtime=0) as gz:
        gz.write(tar_buf.getvalue())

    return gz_buf.getvalue(), len(index_specs)


def build_deterministic_archive(repo_root: Path) -> tuple[Path, str, int]:
    out_path = repo_root / "server" / "internal" / "frameworks" / "embedded_catalogs.tar.gz"
    archive_bytes, count = build_archive_bytes(repo_root)
    tmp_path = out_path.with_suffix(".tar.gz.tmp")
    tmp_path.write_bytes(archive_bytes)
    os.replace(tmp_path, out_path)
    digest = hashlib.sha256(archive_bytes).hexdigest()
    return out_path, digest, count


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify embedded_catalogs.tar.gz matches deterministic output")
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parent.parent
    out_path = repo_root / "server" / "internal" / "frameworks" / "embedded_catalogs.tar.gz"

    if args.check:
        expected, count = build_archive_bytes(repo_root)
        actual = out_path.read_bytes() if out_path.is_file() else b""
        if actual != expected:
            print(f"[FAIL] {out_path.relative_to(repo_root)} is out of sync with data/frameworks/manifest.yaml and catalogs", file=sys.stderr)
            return 1
        print(f"[OK] Verified {count} embedded catalogs ({len(actual)} bytes, sha256={hashlib.sha256(actual).hexdigest()})")
        return 0

    out_path, digest, count = build_deterministic_archive(repo_root)
    print(
        f"[OK] Packaged {count} catalogs into {out_path.relative_to(repo_root)} "
        f"({out_path.stat().st_size} bytes, sha256={digest})"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
