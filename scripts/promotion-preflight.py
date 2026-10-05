#!/usr/bin/env python3
"""Read-only refusal gate for immutable public release identities."""
import argparse
import re
import subprocess
from release_inventory import identity

REPOSITORY = 'ckodex-labs/ckodex-xoscal'


def preflight(tag, revision, run=subprocess.run):
    identity(tag, revision)
    if tag == 'dev':
        raise ValueError('development cannot be promoted')
    release = run(['gh', 'api', '--include', 'repos/' + REPOSITORY + '/releases/tags/' + tag],
                  capture_output=True, text=True)
    status = re.search(r'^HTTP/[^ ]+ ([0-9]{3})\b', release.stdout, re.M)
    if release.returncode == 0 or not status or status[1] != '404':
        raise ValueError('release exists or absence could not be established; refuse promotion')
    image = run(['skopeo', 'inspect', '--raw', 'docker://ghcr.io/' + REPOSITORY + ':' + tag],
                capture_output=True, text=True)
    # Skopeo emits the registry's structured OCI error code in its diagnostic.
    # Authorization, network and other failures do not establish absence.
    if image.returncode == 0 or not re.search(r'\bmanifest unknown\b|\bMANIFEST_UNKNOWN\b', image.stderr):
        raise ValueError('registry tag exists or absence could not be established; refuse promotion')
    print('promotion-preflight: both immutable destinations are absent')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--revision', required=True)
    args = parser.parse_args()
    preflight(args.tag, args.revision)
