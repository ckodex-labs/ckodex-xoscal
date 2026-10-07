"""Exercise the promotion gate's strict consumer with real unsigned archives."""
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from release_inventory import create

SCRIPTS = Path(__file__).parent
spec = importlib.util.spec_from_file_location('verify_transport', SCRIPTS / 'verify-transport.py')
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class VerifyTransportTests(unittest.TestCase):
    def fixture(self, parent):
        root = parent / 'source'
        first = root / 'evidence/first/producer.txt'
        second = root / 'evidence/second/producer.txt'
        first.parent.mkdir(parents=True)
        second.parent.mkdir(parents=True)
        first.write_bytes(b'unsigned transport contract fixture\n')
        os.link(first, second)
        manifest = create(root, 'dev', '0' * 40, final=True)
        (root / 'release-manifest.json').write_text(json.dumps(manifest, sort_keys=True, indent=2) + '\n')
        return root

    def package(self, root, output):
        subprocess.run([sys.executable, str(SCRIPTS / 'package-candidate.py'), str(root), str(output)],
                       check=True, capture_output=True, text=True)

    def test_regular_transport_passes_structure_without_claiming_signature(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            source = self.fixture(parent)
            output = parent / 'transport.tar.gz'
            self.package(source, output)
            manifest = gate.unpack_transport(output, parent / 'extracted', 'dev', '0' * 40)
            self.assertEqual(manifest['scope'], 'final')
            self.assertEqual(manifest['release_tag'], 'dev')

    def test_legacy_hardlinked_transport_fails_before_extraction(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            source = self.fixture(parent)
            output = parent / 'legacy.tar.gz'
            with tarfile.open(output, 'w:gz') as archive:
                for path in sorted(source.rglob('*')):
                    if path.is_file():
                        info = archive.gettarinfo(str(path), arcname=path.relative_to(source).as_posix())
                        with path.open('rb') as stream:
                            archive.addfile(info, stream)
            extracted = parent / 'extracted'
            with self.assertRaisesRegex(ValueError, 'noncanonical or non-file entry'):
                gate.unpack_transport(output, extracted, 'dev', '0' * 40)
            self.assertFalse(extracted.exists())

    def test_tampered_regular_member_fails_manifest_integrity(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            source = self.fixture(parent)
            output = parent / 'tampered.tar.gz'
            with tarfile.open(output, 'w:gz', dereference=True) as archive:
                for path in sorted(source.rglob('*')):
                    if path.is_file():
                        raw = path.read_bytes()
                        if path.name == 'producer.txt':
                            raw += b'tampered\n'
                        info = archive.gettarinfo(str(path), arcname=path.relative_to(source).as_posix())
                        info.size = len(raw)
                        archive.addfile(info, io.BytesIO(raw))
            with self.assertRaises(ValueError):
                gate.unpack_transport(output, parent / 'extracted', 'dev', '0' * 40)

    def test_embedded_identity_must_match_expected_release(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            source = self.fixture(parent)
            output = parent / 'transport.tar.gz'
            self.package(source, output)
            with self.assertRaisesRegex(ValueError, 'identity differs'):
                gate.unpack_transport(output, parent / 'extracted', 'v0.3.2', '0' * 40)


if __name__ == '__main__':
    unittest.main()
