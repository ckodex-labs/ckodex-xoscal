#!/usr/bin/env python3
"""Synthetic OCI descriptor tests; these are not built/published container proofs."""
import copy
import gzip
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('oci_inspector_test', Path(__file__).with_name('inspect-image-archive.py'))
oci = importlib.util.module_from_spec(spec)
spec.loader.exec_module(oci)
INDEX = 'application/vnd.oci.image.index.v1+json'
MANIFEST = 'application/vnd.oci.image.manifest.v1+json'
CONFIG = 'application/vnd.oci.image.config.v1+json'
LAYER = 'application/vnd.oci.image.layer.v1.tar+gzip'


def json_bytes(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def add_blob(files, value, media_type):
    raw = value if isinstance(value, bytes) else json_bytes(value)
    sha = hashlib.sha256(raw).hexdigest()
    files['blobs/sha256/' + sha] = raw
    return {'digest': 'sha256:' + sha, 'size': len(raw), 'mediaType': media_type}


def make_layout(arches=('amd64', 'arm64'), *, os_name='linux', config_os=None, duplicate=False, bad_size=False, bad_type=False, missing_child=False, direct=False):
    files = {'oci-layout': json_bytes({'imageLayoutVersion': '1.0.0'})}
    images = []
    for arch in arches:
        config = add_blob(files, {'architecture': arch, 'os': config_os or os_name, 'rootfs': {'type': 'layers', 'diff_ids': []}}, CONFIG)
        layer = add_blob(files, gzip.compress(('fixture-layer-' + arch).encode(), mtime=0), LAYER)
        if bad_size:
            layer['size'] += 1
        if bad_type:
            layer['mediaType'] = 'application/x-unknown-layer'
        image = add_blob(files, {'schemaVersion': 2, 'mediaType': MANIFEST, 'config': config, 'layers': [layer]}, MANIFEST)
        image['platform'] = {'os': os_name, 'architecture': arch}
        if missing_child:
            del files['blobs/sha256/' + config['digest'].split(':')[1]]
        images.append(image)
    if duplicate:
        images[-1]['platform'] = dict(images[0]['platform'])
    root = add_blob(files, {'schemaVersion': 2, 'mediaType': INDEX, 'manifests': images}, INDEX)
    if direct:
        files['index.json'] = files.pop('blobs/sha256/' + root['digest'].split(':')[1])
    elif len(arches) == 1:
        del files['blobs/sha256/' + root['digest'].split(':')[1]]
        files['index.json'] = json_bytes({'schemaVersion': 2, 'manifests': images})
    else:
        files['index.json'] = json_bytes({'schemaVersion': 2, 'manifests': [root]})
    return files


def hybrid_layout():
    """Match observed BuildKit single-image Docker/OCI hybrid metadata."""
    files = make_layout(('arm64',))
    index = json.loads(files['index.json'])
    index['manifests'][0].pop('platform')
    descriptor = index['manifests'][0]
    manifest = json.loads(files['blobs/sha256/' + descriptor['digest'].split(':')[1]])
    files['index.json'] = json_bytes(index)
    files['manifest.json'] = json_bytes([{
        'Config': 'blobs/sha256/' + manifest['config']['digest'].split(':')[1], 'RepoTags': None,
        'Layers': ['blobs/sha256/' + layer['digest'].split(':')[1] for layer in manifest['layers']],
    }])
    return files


def write_archive(path, files, extra=None):
    with tarfile.open(path, 'w') as stream:
        for name, raw in files.items():
            info = tarfile.TarInfo(name)
            info.size = len(raw)
            stream.addfile(info, io.BytesIO(raw))
        if extra:
            name, raw, kind = extra
            info = tarfile.TarInfo(name)
            info.type = kind
            info.size = len(raw) if kind == tarfile.REGTYPE else 0
            info.linkname = 'index.json' if kind in (tarfile.SYMTYPE, tarfile.LNKTYPE) else ''
            stream.addfile(info, io.BytesIO(raw) if kind == tarfile.REGTYPE else None)


class OciTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='oci-contract-test-')
        self.addCleanup(self.temp.cleanup)
        self.archive = Path(self.temp.name) / 'image.tar'

    def test_complete_two_platform_graph(self):
        write_archive(self.archive, make_layout())
        raw, report = oci.inspect(self.archive)
        self.assertEqual(report['status'], 'passed')
        self.assertEqual({p['arch'] for p in report['platforms']}, {'amd64', 'arm64'})
        self.assertEqual(report['root_digest'], 'sha256:' + hashlib.sha256(raw).hexdigest())

    def test_direct_layout_and_single_platform(self):
        write_archive(self.archive, make_layout(direct=True))
        oci.inspect(self.archive)
        write_archive(self.archive, make_layout(('arm64',)))
        oci.inspect(self.archive, {('linux', 'arm64')})
        with self.assertRaises(ValueError):
            oci.inspect(self.archive)

    def test_buildkit_hybrid_optional_platform_descriptor(self):
        write_archive(self.archive, hybrid_layout())
        _, report = oci.inspect(self.archive, {('linux', 'arm64')})
        self.assertEqual(report['platforms'][0]['arch'], 'arm64')

    def test_docker_metadata_cannot_point_to_other_graph(self):
        for change in ('config', 'layer', 'path', 'duplicate'):
            with self.subTest(change=change):
                files = hybrid_layout()
                metadata = json.loads(files['manifest.json'])
                if change == 'config':
                    metadata[0]['Config'] = 'blobs/sha256/' + 'b' * 64
                elif change == 'layer':
                    metadata[0]['Layers'] = ['blobs/sha256/' + 'b' * 64]
                elif change == 'path':
                    metadata[0]['Config'] = '../config.json'
                else:
                    metadata.append(copy.deepcopy(metadata[0]))
                files['manifest.json'] = json_bytes(metadata)
                write_archive(self.archive, files)
                with self.assertRaisesRegex(ValueError, 'Docker compatibility'):
                    oci.inspect(self.archive, {('linux', 'arm64')})

    def test_missing_child_blob_rejected(self):
        write_archive(self.archive, make_layout(missing_child=True))
        with self.assertRaisesRegex(ValueError, 'missing OCI blob'):
            oci.inspect(self.archive)

    def test_layer_hash_mismatch_rejected(self):
        files = make_layout()
        name = next(n for n, value in files.items() if value.startswith(b'\x1f\x8b'))
        files[name] += b'tamper'
        write_archive(self.archive, files)
        with self.assertRaisesRegex(ValueError, 'hash mismatch'):
            oci.inspect(self.archive)

    def test_descriptor_size_mismatch_rejected(self):
        write_archive(self.archive, make_layout(bad_size=True))
        with self.assertRaisesRegex(ValueError, 'size mismatch'):
            oci.inspect(self.archive)

    def test_unknown_layer_media_type_rejected(self):
        write_archive(self.archive, make_layout(bad_type=True))
        with self.assertRaisesRegex(ValueError, 'media type'):
            oci.inspect(self.archive)

    def test_windows_platform_rejected(self):
        write_archive(self.archive, make_layout(os_name='windows'))
        with self.assertRaisesRegex(ValueError, 'platform'):
            oci.inspect(self.archive)

    def test_duplicate_platform_rejected(self):
        write_archive(self.archive, make_layout(duplicate=True))
        with self.assertRaisesRegex(ValueError, 'platform'):
            oci.inspect(self.archive)

    def test_config_platform_mismatch_rejected(self):
        write_archive(self.archive, make_layout(config_os='windows'))
        with self.assertRaisesRegex(ValueError, 'config differs'):
            oci.inspect(self.archive)

    def test_unsafe_duplicate_and_link_members_rejected(self):
        for extra in (('../escape', b'x', tarfile.REGTYPE), ('/absolute', b'x', tarfile.REGTYPE),
                      ('index.json', b'x', tarfile.REGTYPE), ('link', b'', tarfile.SYMTYPE), ('hardlink', b'', tarfile.LNKTYPE)):
            with self.subTest(member=extra[0]):
                write_archive(self.archive, make_layout(), extra)
                with self.assertRaises(ValueError):
                    oci.inspect(self.archive)

    def test_unreferenced_blob_rejected(self):
        files = make_layout()
        add_blob(files, b'not in image graph', LAYER)
        write_archive(self.archive, files)
        with self.assertRaisesRegex(ValueError, 'unreferenced'):
            oci.inspect(self.archive)

    def test_same_platform_graph_from_combined_and_single_bytes(self):
        combined = make_layout()
        write_archive(self.archive, combined)
        _, report = oci.inspect(self.archive)
        single = Path(self.temp.name) / 'single.tar'
        write_archive(single, make_layout(('amd64',)))
        _, one = oci.inspect(single, {('linux', 'amd64')})
        self.assertEqual(one['platforms'], [report['platforms'][0]])


if __name__ == '__main__':
    print('Synthetic OCI graph validation only; no container build, scan or hosted proof.', flush=True)
    unittest.main(verbosity=2)
