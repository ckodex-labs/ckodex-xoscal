"""Exercise real packaging and strict extraction with shared source inodes."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest

SCRIPTS = Path(__file__).parent
PRODUCER = SCRIPTS / 'package-candidate.py'
sys.path.insert(0, str(SCRIPTS))
from release_inventory import create, validate

spec = importlib.util.spec_from_file_location('download_release', SCRIPTS / 'download-release.py')
consumer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(consumer)


class PackageCandidateTests(unittest.TestCase):
    def fixture(self, parent, name, linked):
        root = parent / name
        first = root / 'evidence/first/producer.txt'
        second = root / 'evidence/second/producer.txt'
        first.parent.mkdir(parents=True)
        second.parent.mkdir(parents=True)
        first.write_bytes(b'unsigned packaging test fixture\n')
        if linked:
            os.link(first, second)
        else:
            second.write_bytes(first.read_bytes())
        manifest = create(root, 'dev', '0' * 40, final=True)
        (root / 'release-manifest.json').write_text(json.dumps(manifest, sort_keys=True, indent=2) + '\n')
        return root

    def package(self, root, output):
        subprocess.run([sys.executable, str(PRODUCER), str(root), str(output)],
                       check=True, capture_output=True, text=True)

    def test_shared_inodes_round_trip_through_strict_consumer(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            root = self.fixture(parent, 'source', linked=True)
            output = parent / 'transport.tar.gz'
            self.package(root, output)
            with tarfile.open(output) as archive:
                self.assertTrue(all(member.isfile() for member in archive))
            extracted = parent / 'extracted'
            consumer.safe_extract(output, extracted)
            manifest = json.loads((extracted / 'release-manifest.json').read_text())
            validate(extracted, manifest, final=True)
            for relative in ('evidence/first/producer.txt', 'evidence/second/producer.txt'):
                self.assertEqual((root / relative).read_bytes(), (extracted / relative).read_bytes())

    def test_transport_bytes_do_not_depend_on_inode_sharing(self):
        with tempfile.TemporaryDirectory() as temporary:
            parent = Path(temporary)
            linked = self.fixture(parent, 'linked', linked=True)
            separate = self.fixture(parent, 'separate', linked=False)
            linked_output = parent / 'linked.tar.gz'
            separate_output = parent / 'separate.tar.gz'
            self.package(linked, linked_output)
            self.package(separate, separate_output)
            self.assertEqual(linked_output.read_bytes(), separate_output.read_bytes())


if __name__ == '__main__':
    unittest.main()
