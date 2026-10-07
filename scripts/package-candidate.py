#!/usr/bin/env python3
"""Deterministic transport envelope of frozen, integrity-checked site bytes."""
import gzip
import json
import tarfile
import sys
from pathlib import Path
from release_inventory import files, validate

root, output = Path(sys.argv[1]), Path(sys.argv[2])
manifest = json.loads((root / 'release-manifest.json').read_text())
validate(root, manifest, final=True)
with output.open('wb') as destination:
    with gzip.GzipFile(fileobj=destination, filename='', mtime=0) as compressed:
        with tarfile.open(fileobj=compressed, mode='w', dereference=True) as archive:
            for name in sorted(files(root)):
                info = archive.gettarinfo(str(root / name), arcname=name)
                if not info.isfile():
                    raise ValueError('transport producer requires regular file headers: ' + name)
                info.uid = info.gid = info.mtime = 0
                info.uname = info.gname = ''
                with (root / name).open('rb') as stream:
                    archive.addfile(info, stream)
print('package-candidate: frozen site transport written')
