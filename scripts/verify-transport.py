#!/usr/bin/env python3
"""Verify exact frozen transport under the same hosted manifest identity."""
import json
import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
from attestation_policy import verification_flags
from release_inventory import digest, validate, write


def unpack_transport(transport, root, tag, revision):
    """Use the actual public consumer's strict extractor and manifest policy."""
    spec = importlib.util.spec_from_file_location('download_release', Path(__file__).with_name('download-release.py'))
    consumer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(consumer)
    consumer.safe_extract(Path(transport), root)
    manifest = json.loads((root / 'release-manifest.json').read_text())
    if (manifest.get('release_tag'), manifest.get('source_revision')) != (tag, revision):
        raise ValueError('transport manifest identity differs from expected release')
    validate(root, manifest, final=True)
    return manifest


def main():
    transport, bundle, tag, revision, receipt = sys.argv[1:]
    result = subprocess.run(['gh', 'attestation', 'verify', transport] + verification_flags(bundle, tag, revision),
                            check=True, capture_output=True, text=True)
    verified = json.loads(result.stdout)
    if not isinstance(verified, list) or not verified:
        raise ValueError('verifier supplied no successful transport attestation')
    with tempfile.TemporaryDirectory(prefix='xoscal-transport-admission-') as temporary:
        root = Path(temporary) / 'extracted'
        unpack_transport(transport, root, tag, revision)
        subprocess.run([sys.executable, str(Path(__file__).with_name('release-contract.py')), 'verify',
                        '--root', str(root), '--bundle', bundle, '--tag', tag, '--revision', revision,
                        '--final', '--required'], check=True)
        manifest_digest = digest(root / 'release-manifest.json')
    write(Path(receipt), {'scope': 'exact-transport-verification', 'release_tag': tag,
                         'source_revision': revision, 'transport_digest': digest(Path(transport)),
                         'bundle_digest': digest(Path(bundle)), 'results': verified,
                         'consumer_admission': 'passed', 'final_manifest_digest': manifest_digest})


if __name__ == '__main__':
    main()
