#!/usr/bin/env python3
"""Exercise actual safe extraction against malicious ZIP and tar members."""
import importlib.util
import io
from pathlib import Path
import stat
import tarfile
import tempfile
import unittest
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / 'dagger' / 'extract_archive.py'
spec = importlib.util.spec_from_file_location('extract_archive', SCRIPT)
extractor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(extractor)


class ArchiveTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.archive = self.root / 'input.zip'
        self.out = self.root / 'out'

    def test_zip_contents_are_preserved(self):
        with zipfile.ZipFile(self.archive, 'w') as archive:
            archive.writestr('pkg/metadata.json', b'actual bytes')
        self.assertEqual(extractor.extract(self.archive, self.out), (1, 12))
        self.assertEqual((self.out / 'pkg/metadata.json').read_bytes(), b'actual bytes')

    def test_traversal_and_absolute_paths_are_rejected(self):
        for path in ('../escape', '/escape', 'C:/escape', 'a\\escape', 'a/../escape'):
            with self.subTest(path=path):
                with zipfile.ZipFile(self.archive, 'w') as archive:
                    archive.writestr(path, b'bad')
                with self.assertRaises(ValueError):
                    extractor.extract(self.archive, self.root / ('out' + str(len(list(self.root.iterdir())))))
        self.assertFalse((self.root.parent / 'escape').exists())

    def test_zip_symlink_is_rejected(self):
        with zipfile.ZipFile(self.archive, 'w') as archive:
            entry = zipfile.ZipInfo('link')
            entry.create_system = 3
            entry.external_attr = (stat.S_IFLNK | 0o777) << 16
            archive.writestr(entry, '../escape')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.out)

    def test_tar_links_and_devices_are_rejected(self):
        for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.CHRTYPE):
            with self.subTest(kind=kind):
                with tarfile.open(self.archive, 'w') as archive:
                    entry = tarfile.TarInfo('link')
                    entry.type = kind
                    entry.linkname = '../escape'
                    archive.addfile(entry)
                with self.assertRaises(ValueError):
                    extractor.extract(self.archive, self.root / ('out' + str(len(list(self.root.iterdir())))))

    def test_safe_tar_dot_prefix_is_supported(self):
        with tarfile.open(self.archive, 'w') as archive:
            directory = tarfile.TarInfo('.')
            directory.type = tarfile.DIRTYPE
            archive.addfile(directory)
            entry = tarfile.TarInfo('./actual/file')
            entry.size = 3
            archive.addfile(entry, io.BytesIO(b'yes'))
        self.assertEqual(extractor.extract(self.archive, self.out), (1, 3))
        self.assertEqual((self.out / 'actual/file').read_bytes(), b'yes')

    def test_duplicate_normalized_paths_are_rejected(self):
        with zipfile.ZipFile(self.archive, 'w') as archive:
            archive.writestr('./same', b'one')
            archive.writestr('same', b'two')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.out)

    def test_empty_unsupported_or_directory_only_never_passes(self):
        with zipfile.ZipFile(self.archive, 'w') as archive:
            archive.writestr('dir/', b'')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.out)
        self.archive.write_bytes(b'not an archive')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.root / 'out2')

    def test_size_and_directory_entry_limits_are_enforced(self):
        old_bytes, old_entries = extractor.MAX_BYTES, extractor.MAX_FILES
        self.addCleanup(setattr, extractor, 'MAX_BYTES', old_bytes)
        self.addCleanup(setattr, extractor, 'MAX_FILES', old_entries)
        extractor.MAX_BYTES = 2
        with zipfile.ZipFile(self.archive, 'w') as archive:
            archive.writestr('big', b'123')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.out)
        extractor.MAX_FILES = 1
        with zipfile.ZipFile(self.archive, 'w') as archive:
            archive.writestr('a/', b'')
            archive.writestr('b/', b'')
        with self.assertRaises(ValueError):
            extractor.extract(self.archive, self.root / 'out2')


if __name__ == '__main__':
    unittest.main()
