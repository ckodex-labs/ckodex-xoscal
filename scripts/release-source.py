#!/usr/bin/env python3
"""Derive production inputs from a validated Git commit, without caller metadata."""
import argparse
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile
import tempfile

from release_inventory import identity

PUBLIC_ORIGIN = 'https://github.com/ckodex-labs/ckodex-xoscal.git'


def git_environment():
    # Do not inherit injected Git configuration, global hooks, attributes or
    # alternate Git directories. No caller checkout credentials are copied.
    env = {key: value for key, value in os.environ.items() if not key.startswith('GIT_')}
    env.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL='/dev/null',
               GIT_ATTR_NOSYSTEM='1', GIT_TERMINAL_PROMPT='0')
    return env


def git(root, *arguments):
    return subprocess.check_output(['git', '-C', str(root), *arguments],
                                   env=git_environment(), text=True).strip()


def extract_tree(archive, destination):
    with tarfile.open(archive, 'r:') as source:
        for member in source:
            path = PurePosixPath(member.name)
            if (not path.parts or path.is_absolute() or str(path) != member.name
                    or any(part in ('.', '..') for part in path.parts)
                    or path.parts[0] == '.git' or '\\' in member.name
                    or not (member.isdir() or member.isfile())
                    or member.issym() or member.islnk()):
                raise ValueError('unsupported tracked source entry: ' + member.name)
            target = destination.joinpath(*path.parts)
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            stream = source.extractfile(member)
            if stream is None:
                raise ValueError('tracked source entry has no contents')
            with stream, target.open('xb') as output:
                while block := stream.read(1024 * 1024):
                    output.write(block)
            target.chmod(member.mode & 0o777)


def production_source(root, tag, revision, version, destination, output):
    # The same explicit tag is promoted to OCI; no implicit tag rewriting may
    # change the release identity. SemVer build metadata uses unsupported '+'.
    if not re.fullmatch(r'[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}', tag):
        raise ValueError('production release tag must be OCI-compatible (at most 128 ASCII characters; no SemVer build metadata)')
    identity(tag, revision)
    if tag == 'dev' or version != tag:
        raise ValueError('producer version must equal explicit production release tag')
    if (git(root, 'rev-parse', 'HEAD') != revision
            or git(root, 'rev-parse', 'refs/tags/' + tag + '^{commit}') != revision):
        raise ValueError('checked-out source or tag differs from independently expected revision')
    if git(root, 'status', '--porcelain', '--untracked-files=all'):
        raise ValueError('production release source must be clean')
    if any(entry.startswith('160000 ') for entry in git(root, 'ls-tree', '-r', '-z', revision).split('\0')):
        raise ValueError('production source requires explicit submodule materialization support')
    destination, output = Path(destination), Path(output)
    if os.path.lexists(destination):
        raise ValueError('production source destination must not exist')
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.release-source-', dir=destination.parent) as temp:
        clean = Path(temp) / 'tree'
        # Local object copying never runs caller upload-pack configuration;
        # --no-hardlinks gives the derived tree independent Git object files.
        subprocess.run(['git', 'clone', '--quiet', '--local', '--no-hardlinks', '--dissociate',
                        '--no-checkout', '--template=', str(Path(root).resolve()), str(clean)],
                       env=git_environment(), check=True)
        git(clean, 'remote', 'set-url', 'origin', PUBLIC_ORIGIN)
        git(clean, 'config', 'core.attributesFile', '/dev/null')
        attributes = clean / '.git/info/attributes'
        attributes.parent.mkdir(parents=True, exist_ok=True)
        attributes.write_text('* -export-ignore -export-subst\n')
        git(clean, 'update-ref', '--no-deref', 'HEAD', revision)
        git(clean, 'read-tree', revision)
        archive = Path(temp) / 'source.tar'
        with archive.open('wb') as stream:
            subprocess.run(['git', '-C', str(clean), 'archive', '--format=tar', revision],
                           env=git_environment(), stdout=stream, check=True)
        extract_tree(archive, clean)
        if git(clean, 'status', '--porcelain', '--untracked-files=all'):
            raise ValueError('archive differs from exact tracked commit')
        if git(clean, 'rev-parse', 'HEAD') != revision:
            raise ValueError('derived Git metadata differs from expected revision')
        clean.rename(destination)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({'schema_version': 1, 'release_tag': tag,
                                  'source_revision': revision, 'version': version,
                                  'scope': 'clean-tagged-source', 'status': 'passed'}, sort_keys=True) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('tag')
    parser.add_argument('revision')
    parser.add_argument('version')
    parser.add_argument('output', type=Path)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    production_source(Path.cwd(), args.tag, args.revision, args.version, args.destination, args.output)


if __name__ == '__main__':
    main()
