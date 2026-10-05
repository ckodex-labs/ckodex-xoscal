#!/usr/bin/env python3
"""Extract analysis inputs without trusting archive paths, links, or devices."""
import argparse
import shutil
import stat
import tarfile
import zipfile
from pathlib import Path, PurePosixPath

MAX_FILES = 100000
MAX_BYTES = 1024 * 1024 * 1024


def safe_name(name):
    while name.startswith('./'):
        name = name[2:]
    parts = PurePosixPath(name).parts
    if not parts or name.startswith('/') or '\\' in name or any(x in ('.', '..') for x in name.split('/')) or ':' in parts[0]:
        raise ValueError(f'unsafe archive path: {name!r}')
    return Path(*parts)


def extract(source, destination, allow_file=False):
    destination.mkdir(parents=True, exist_ok=False)
    seen, files, size = set(), 0, 0

    def target(name, length, directory=False):
        nonlocal files, size
        relative = safe_name(name.rstrip('/'))
        if relative in seen:
            raise ValueError(f'duplicate archive path: {name!r}')
        seen.add(relative)
        if len(seen) > MAX_FILES:
            raise ValueError('archive exceeds analysis entry limit')
        if not directory:
            files += 1
            size += length
            if length < 0 or files > MAX_FILES or size > MAX_BYTES:
                raise ValueError('archive exceeds analysis extraction limits')
        output = destination / relative
        output.parent.mkdir(parents=True, exist_ok=True)
        if directory:
            output.mkdir(exist_ok=True)
        return output

    if zipfile.is_zipfile(source):
        with zipfile.ZipFile(source) as archive:
            for member in archive.infolist():
                mode = member.external_attr >> 16
                kind = stat.S_IFMT(mode)
                if kind not in (0, stat.S_IFREG, stat.S_IFDIR) or member.flag_bits & 1:
                    raise ValueError(f'unsupported archive member: {member.filename!r}')
                output = target(member.filename, member.file_size, member.is_dir())
                if not member.is_dir():
                    with archive.open(member) as incoming, output.open('xb') as outgoing:
                        shutil.copyfileobj(incoming, outgoing)
    elif tarfile.is_tarfile(source):
        with tarfile.open(source, mode='r:*') as archive:
            for member in archive:
                if not (member.isfile() or member.isdir()) or member.issparse():
                    raise ValueError(f'unsupported archive member: {member.name!r}')
                if member.isdir() and member.name in ('.', './'):
                    continue
                output = target(member.name, member.size, member.isdir())
                if member.isfile():
                    with archive.extractfile(member) as incoming, output.open('xb') as outgoing:
                        shutil.copyfileobj(incoming, outgoing)
    elif allow_file:
        output = target('artifact', source.stat().st_size)
        with source.open('rb') as incoming, output.open('xb') as outgoing:
            shutil.copyfileobj(incoming, outgoing)
    else:
        raise ValueError('unsupported artifact archive format')
    if not files:
        raise ValueError('archive has no file content to analyze')
    return files, size


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('destination', type=Path)
    parser.add_argument('--allow-file', action='store_true')
    args = parser.parse_args()
    count, size = extract(args.source, args.destination, args.allow_file)
    print(f'extracted {count} files ({size} bytes) from exact supplied artifact')


if __name__ == '__main__':
    main()
