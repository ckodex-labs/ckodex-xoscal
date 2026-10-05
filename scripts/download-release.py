#!/usr/bin/env python3
"""Retrieve one explicit release; admit verified bytes atomically without overwrites."""
import argparse
import ctypes
import errno
import gzip
import json
import os
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path, PurePosixPath
from attestation_policy import verification_flags
from release_inventory import identity

# Fixed admission ceilings accommodate the multiarch image and all SDK packages.
MAX_TRANSPORT_BYTES = 2 * 1024 ** 3
MAX_EXPANDED_BYTES = 8 * 1024 ** 3
MAX_MANIFEST_BYTES = 32 * 1024 ** 2
MAX_BUNDLE_BYTES = 64 * 1024 ** 2
MAX_FILES = 100000
DOWNLOAD_TIMEOUT_SECONDS = 1800
REPOSITORY = 'ckodex-labs/ckodex-xoscal'


def absent(destination):
    if os.path.lexists(destination):
        raise ValueError('destination must not exist: ' + str(destination))


def download(url, destination, limit):
    subprocess.run(['curl', '-fsSL', '--proto', '=https', '--proto-redir', '=https',
                    '--connect-timeout', '30', '--max-time', str(DOWNLOAD_TIMEOUT_SECONDS),
                    '--max-filesize', str(limit), url, '-o', str(destination)], check=True,
                   timeout=DOWNLOAD_TIMEOUT_SECONDS + 30)
    if not destination.is_file() or destination.stat().st_size > limit:
        raise ValueError('download exceeds admission limit: ' + destination.name)


class BoundedTarStream:
    """Cap tar metadata reads and seeks too, before gzip allocates their contents."""
    def __init__(self, stream, limit):
        self.stream, self.limit = stream, limit

    def tell(self):
        return self.stream.tell()

    def read(self, size=-1):
        if size < 0 or self.tell() + size > self.limit:
            raise ValueError('transport exceeds expanded stream admission limit')
        return self.stream.read(size)

    def seek(self, offset, whence=0):
        position = offset if whence == 0 else self.tell() + offset if whence == 1 else -1
        if position < 0 or position > self.limit:
            raise ValueError('transport exceeds expanded stream admission limit')
        return self.stream.seek(position)


def safe_extract(transport, destination, max_bytes=MAX_EXPANDED_BYTES, max_files=MAX_FILES):
    """Inspect all headers before writing; never honor archive links or modes."""
    if transport.stat().st_size > MAX_TRANSPORT_BYTES:
        raise ValueError('transport exceeds compressed admission limit')
    absent(destination)
    with gzip.open(transport, 'rb') as compressed, tarfile.open(
            fileobj=BoundedTarStream(compressed, max_bytes), mode='r:') as archive:
        members, names, total = [], set(), 0
        for member in archive:
            name = member.name
            path = PurePosixPath(name)
            if (not member.isfile() or member.issym() or member.islnk() or member.issparse()
                    or path.is_absolute() or str(path) != name or not path.parts
                    or any(part in ('.', '..') for part in path.parts)
                    or '\\' in name or any(ord(c) < 32 or ord(c) == 127 for c in name)):
                raise ValueError('transport contains a noncanonical or non-file entry: ' + repr(name))
            if name in names:
                raise ValueError('transport contains duplicate file: ' + name)
            if member.size < 0:
                raise ValueError('transport contains an invalid size')
            total += member.size
            if total > max_bytes or len(members) >= max_files:
                raise ValueError('transport exceeds expanded admission limit')
            names.add(name)
            members.append(member)
        for name in names:
            if any(str(parent) in names for parent in PurePosixPath(name).parents if str(parent) != '.'):
                raise ValueError('transport contains conflicting file and directory names')
        # Detached final proofs are supplied independently; archive copies cannot replace them.
        if names.intersection({'proofs/release-manifest.sigstore.json', 'proofs/verification.json'}):
            raise ValueError('transport contains reserved detached final proof')
        destination.mkdir()
        for member in members:
            target = destination.joinpath(*PurePosixPath(member.name).parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            source = archive.extractfile(member)
            if source is None:
                raise ValueError('transport file has no contents')
            remaining = member.size
            with source, target.open('xb') as output:
                while remaining:
                    data = source.read(min(1024 ** 2, remaining))
                    if not data:
                        raise ValueError('transport file is truncated')
                    output.write(data)
                    remaining -= len(data)
            target.chmod(0o644)


def atomic_promote(staging, destination):
    """OS no-replace rename keeps even a concurrently created destination intact."""
    absent(destination)
    libc = ctypes.CDLL(None, use_errno=True)
    if sys.platform == 'darwin':
        rename = libc.renamex_np
        rename.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_uint]
        result = rename(os.fsencode(staging), os.fsencode(destination), 0x00000004)  # RENAME_EXCL
    elif sys.platform.startswith('linux') and hasattr(libc, 'renameat2'):
        rename = libc.renameat2
        rename.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
        result = rename(-100, os.fsencode(staging), -100, os.fsencode(destination), 1)  # RENAME_NOREPLACE
    else:
        raise ValueError('atomic no-overwrite admission is unsupported on this platform')
    if result:
        error = ctypes.get_errno()
        if error in (errno.EEXIST, errno.ENOTEMPTY):
            raise ValueError('destination appeared during admission; refuse overwrite')
        raise OSError(error, os.strerror(error), str(destination))


def retrieve(tag, revision, destination):
    identity(tag, revision)
    if tag == 'dev':
        raise ValueError('development is not a published release')
    destination = destination.absolute()
    absent(destination)
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.xoscal-release-', dir=destination.parent) as temporary:
        incoming = Path(temporary)
        staging = incoming / 'verified'
        base = 'https://github.com/' + REPOSITORY + '/releases/download/' + tag + '/'
        for name, limit in (('release-site.tar.gz', MAX_TRANSPORT_BYTES),
                            ('release-manifest.json', MAX_MANIFEST_BYTES),
                            ('release-manifest.sigstore.json', MAX_BUNDLE_BYTES)):
            download(base + name, incoming / name, limit)
        bundle = incoming / 'release-manifest.sigstore.json'
        common = verification_flags(bundle, tag, revision)
        result = subprocess.run(['gh', 'attestation', 'verify', str(incoming / 'release-site.tar.gz')] + common,
                                check=True, capture_output=True, text=True)
        verified = json.loads(result.stdout)
        if not isinstance(verified, list) or not verified:
            raise ValueError('verifier supplied no successful transport attestation')
        safe_extract(incoming / 'release-site.tar.gz', staging)
        if (staging / 'release-manifest.json').read_bytes() != (incoming / 'release-manifest.json').read_bytes():
            raise ValueError('transport manifest differs from downloaded manifest')
        proof = staging / 'proofs/release-manifest.sigstore.json'
        proof.parent.mkdir(parents=True, exist_ok=True)
        proof.write_bytes(bundle.read_bytes())
        subprocess.run(['python3', str(Path(__file__).with_name('release-contract.py')), 'verify',
                        '--root', str(staging), '--bundle', str(proof), '--tag', tag,
                        '--revision', revision, '--final', '--required'], check=True)
        atomic_promote(staging, destination)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    retrieve(args.tag, args.revision, args.root)


if __name__ == '__main__':
    main()
