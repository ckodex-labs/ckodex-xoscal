#!/usr/bin/env python3
"""Flatten only signed inventory names into GitHub attachment staging."""
import json
import shutil
import sys
from pathlib import Path
from release_inventory import RESERVED_ASSETS, digest, safe_path, validate

root, output = Path(sys.argv[1]), Path(sys.argv[2])
manifest = json.loads((root / 'release-manifest.json').read_text())
validate(root, manifest, final=True)
output.mkdir(parents=True, exist_ok=True)
seen = set()
for item in manifest['artifacts']:
    name = item.get('release_asset_name')
    if not name:
        continue
    if str(safe_path(name)) != name or '/' in name or name in seen or name in RESERVED_ASSETS:
        raise ValueError('duplicate or unsafe release attachment name: ' + name)
    seen.add(name)
    shutil.copyfile(root / item['path'], output / name)
    if digest(output / name) != item['digest']:
        raise ValueError('staged attachment integrity mismatch: ' + name)
for source, name in [('release-manifest.json', 'release-manifest.json'),
                     ('proofs/release-manifest.sigstore.json', 'release-manifest.sigstore.json')]:
    shutil.copyfile(root / source, output / name)
print(f'promotion-assets: {len(seen)} exact payload attachments staged')
