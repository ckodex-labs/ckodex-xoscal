#!/usr/bin/env python3
"""Build deterministic embedded_catalogs.tar.gz for the 31 manifest frameworks."""

from __future__ import annotations

import gzip
import hashlib
import io
import os
import re
import sys
import tarfile
from pathlib import Path


def extract_manifest_ids(provider_go: Path) -> list[str]:
    text = provider_go.read_text(encoding="utf-8")
    # Extract the manifestFrameworkSpecs slice block.
    match = re.search(
        r"var\s+manifestFrameworkSpecs\s*=\s*\[\]manifestFrameworkSpec\s*\{(.*?)\n\}",
        text,
        re.DOTALL,
    )
    if not match:
        raise RuntimeError(f"manifestFrameworkSpecs not found in {provider_go}")
    block = match.group(1)
    ids = re.findall(r'ID:\s*"([^"]+)"', block)
    if len(ids) != 31:
        raise RuntimeError(f"expected 31 manifest framework IDs, found {len(ids)}: {ids}")
    return sorted(ids)


def build_deterministic_archive(repo_root: Path) -> tuple[Path, str, int]:
    provider_go = repo_root / "server" / "internal" / "frameworks" / "manifest_provider.go"
    out_path = repo_root / "server" / "internal" / "frameworks" / "embedded_catalogs.tar.gz"
    data_dir = repo_root / "data" / "frameworks"

    framework_ids = extract_manifest_ids(provider_go)

    tar_buf = io.BytesIO()
    with tarfile.open(fileobj=tar_buf, mode="w", format=tarfile.PAX_FORMAT) as tf:
        for fw_id in framework_ids:
            cat_path = data_dir / fw_id / "catalog.json"
            if not cat_path.is_file():
                raise FileNotFoundError(f"missing catalog: {cat_path}")
            raw = cat_path.read_bytes()
            if not raw:
                raise ValueError(f"empty catalog: {cat_path}")

            arcname = f"{fw_id}/catalog.json"
            info = tarfile.TarInfo(name=arcname)
            info.size = len(raw)
            info.mtime = 0
            info.uid = 0
            info.gid = 0
            info.uname = ""
            info.gname = ""
            info.mode = 0o644
            info.type = tarfile.REGTYPE
            tf.addfile(info, io.BytesIO(raw))

    gz_buf = io.BytesIO()
    with gzip.GzipFile(filename="", mode="wb", fileobj=gz_buf, compresslevel=9, mtime=0) as gz:
        gz.write(tar_buf.getvalue())

    archive_bytes = gz_buf.getvalue()
    tmp_path = out_path.with_suffix(".tar.gz.tmp")
    tmp_path.write_bytes(archive_bytes)
    os.replace(tmp_path, out_path)

    digest = hashlib.sha256(archive_bytes).hexdigest()
    return out_path, digest, len(framework_ids)


def main() -> int:
    repo_root = Path(__file__).resolve().parent.parent
    out_path, digest, count = build_deterministic_archive(repo_root)
    print(
        f"[OK] Packaged {count} catalogs into {out_path.relative_to(repo_root)} "
        f"({out_path.stat().st_size} bytes, sha256={digest})"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
