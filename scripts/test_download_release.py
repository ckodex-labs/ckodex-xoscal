"""Filesystem and refusal tests; never simulate successful hosted cryptographic verification."""
import importlib.util
import io
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('download_release', Path(__file__).with_name('download-release.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DownloadTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.parent = Path(self.temporary.name)
        self.archive = self.parent / 'transport.tar.gz'
        self.root = self.parent / 'release'

    def transport(self, entries):
        with tarfile.open(self.archive, 'w:gz') as archive:
            for name, content, kind in entries:
                member = tarfile.TarInfo(name)
                member.type = kind
                member.size = len(content) if kind == tarfile.REGTYPE else 0
                member.linkname = '../protected'
                archive.addfile(member, io.BytesIO(content) if member.size else None)

    def test_canonical_files_only_extract_and_ignore_executable_modes(self):
        self.transport([('release-manifest.json', b'{}', tarfile.REGTYPE),
                        ('sdk/go.zip', b'actual bytes', tarfile.REGTYPE)])
        module.safe_extract(self.archive, self.root)
        self.assertEqual((self.root / 'sdk/go.zip').read_bytes(), b'actual bytes')
        self.assertEqual((self.root / 'sdk/go.zip').stat().st_mode & 0o777, 0o644)

    def test_traversal_absolute_alias_backslash_and_controls_refuse_before_writes(self):
        for name in ('../protected', '/protected', './protected', 'a//b', 'a/../protected',
                     'a/./b', 'a/', 'a\\b', 'a\nb'):
            with self.subTest(name=name):
                self.transport([(name, b'x', tarfile.REGTYPE)])
                with self.assertRaisesRegex(ValueError, 'noncanonical'):
                    module.safe_extract(self.archive, self.root)
                self.assertFalse(self.root.exists())
        self.assertFalse((self.parent / 'protected').exists())

    def test_special_files_links_and_directories_refuse(self):
        for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.DIRTYPE, tarfile.FIFOTYPE,
                     tarfile.CHRTYPE, tarfile.BLKTYPE):
            with self.subTest(kind=kind):
                self.transport([('unsafe', b'', kind)])
                with self.assertRaisesRegex(ValueError, 'non-file'):
                    module.safe_extract(self.archive, self.root)
                self.assertFalse(self.root.exists())

    def test_duplicate_and_conflicting_names_refuse_before_writes(self):
        for names in (('a', 'a'), ('a', 'a/b'), ('a/b', 'a')):
            self.transport([(name, b'x', tarfile.REGTYPE) for name in names])
            with self.assertRaisesRegex(ValueError, 'duplicate|conflicting'):
                module.safe_extract(self.archive, self.root)
            self.assertFalse(self.root.exists())

    def test_reserved_detached_proof_cannot_be_supplied_by_archive(self):
        for name in ('proofs/release-manifest.sigstore.json', 'proofs/verification.json'):
            self.transport([(name, b'forged', tarfile.REGTYPE)])
            with self.assertRaisesRegex(ValueError, 'reserved'):
                module.safe_extract(self.archive, self.root)
            self.assertFalse(self.root.exists())

    def test_expanded_bytes_and_member_count_are_bounded_before_writes(self):
        self.transport([('a', b'1234', tarfile.REGTYPE), ('b', b'5678', tarfile.REGTYPE)])
        for limits in ({'max_bytes': 7}, {'max_files': 1}):
            with self.assertRaisesRegex(ValueError, 'expanded'):
                module.safe_extract(self.archive, self.root, **limits)
            self.assertFalse(self.root.exists())

    def test_extended_header_metadata_is_bounded_before_allocation(self):
        with tarfile.open(self.archive, 'w:gz', format=tarfile.PAX_FORMAT) as archive:
            member = tarfile.TarInfo('tiny')
            member.size = 1
            member.pax_headers = {'comment': 'x' * 65536}
            archive.addfile(member, io.BytesIO(b'x'))
        with self.assertRaisesRegex(ValueError, 'expanded stream'):
            module.safe_extract(self.archive, self.root, max_bytes=8192)
        self.assertFalse(self.root.exists())

    def test_compressed_bytes_are_bounded(self):
        self.transport([('a', b'x', tarfile.REGTYPE)])
        with patch.object(module, 'MAX_TRANSPORT_BYTES', 1):
            with self.assertRaisesRegex(ValueError, 'compressed'):
                module.safe_extract(self.archive, self.root)
        self.assertFalse(self.root.exists())

    def test_existing_directory_file_and_broken_symlink_are_never_modified(self):
        for kind in ('directory', 'file', 'symlink'):
            with self.subTest(kind=kind):
                root = self.parent / kind
                if kind == 'directory':
                    root.mkdir()
                    (root / 'protected').write_bytes(b'keep')
                elif kind == 'file':
                    root.write_bytes(b'keep')
                else:
                    root.symlink_to(self.parent / 'missing')
                with patch.object(module, 'download') as download:
                    with self.assertRaisesRegex(ValueError, 'must not exist'):
                        module.retrieve('v1.2.3', 'a' * 40, root)
                    download.assert_not_called()
                if kind == 'directory':
                    self.assertEqual((root / 'protected').read_bytes(), b'keep')
                elif kind == 'file':
                    self.assertEqual(root.read_bytes(), b'keep')
                else:
                    self.assertTrue(root.is_symlink())

    def test_download_failure_cleans_unique_scratch_and_keeps_other_incoming(self):
        unrelated = self.parent / 'incoming'
        unrelated.mkdir()
        (unrelated / 'protected').write_bytes(b'keep')
        with patch.object(module, 'download', side_effect=subprocess.CalledProcessError(22, ['curl'])):
            with self.assertRaises(subprocess.CalledProcessError):
                module.retrieve('v1.2.3', 'a' * 40, self.root)
        self.assertFalse(self.root.exists())
        self.assertEqual(list(self.parent.glob('.xoscal-release-*')), [])
        self.assertEqual((unrelated / 'protected').read_bytes(), b'keep')

    def test_failed_verifier_never_extracts_or_publishes(self):
        # Failure-only substitute; no successful attestation is manufactured.
        def provide_download(url, destination, limit):
            destination.write_bytes(b'invalid unsigned bytes')
        with patch.object(module, 'download', side_effect=provide_download), \
                patch.object(module.subprocess, 'run', side_effect=subprocess.CalledProcessError(1, ['gh'])), \
                patch.object(module, 'safe_extract') as extract:
            with self.assertRaises(subprocess.CalledProcessError):
                module.retrieve('v1.2.3', 'a' * 40, self.root)
            extract.assert_not_called()
        self.assertFalse(self.root.exists())
        self.assertEqual(list(self.parent.glob('.xoscal-release-*')), [])

    def test_empty_verifier_results_cannot_admit_transport(self):
        def provide_download(url, destination, limit):
            destination.write_bytes(b'invalid unsigned bytes')
        with patch.object(module, 'download', side_effect=provide_download), \
                patch.object(module.subprocess, 'run', return_value=subprocess.CompletedProcess(['gh'], 0, '[]')), \
                patch.object(module, 'safe_extract') as extract:
            with self.assertRaisesRegex(ValueError, 'no successful'):
                module.retrieve('v1.2.3', 'a' * 40, self.root)
            extract.assert_not_called()
        self.assertFalse(self.root.exists())

    def test_download_enforces_network_and_file_bounds(self):
        target = self.parent / 'download'
        def oversized(*args, **kwargs):
            target.write_bytes(b'12345')
        with patch.object(module.subprocess, 'run', side_effect=oversized) as run:
            with self.assertRaisesRegex(ValueError, 'limit'):
                module.download('https://example.invalid/file', target, 4)
        command = run.call_args[0][0]
        self.assertIn('--max-filesize', command)
        self.assertEqual(command[command.index('--max-filesize') + 1], '4')
        self.assertIn('--max-time', command)
        self.assertIn('--connect-timeout', command)
        self.assertEqual(command.count('=https'), 2)
        self.assertEqual(run.call_args[1]['timeout'], module.DOWNLOAD_TIMEOUT_SECONDS + 30)

    def test_atomic_filesystem_promotion_and_no_overwrite(self):
        # Filesystem primitive only; not a claim that these bytes are a signed release.
        stage = self.parent / 'stage'
        stage.mkdir()
        (stage / 'file').write_bytes(b'bytes')
        module.atomic_promote(stage, self.root)
        self.assertFalse(stage.exists())
        self.assertEqual((self.root / 'file').read_bytes(), b'bytes')
        other = self.parent / 'other'
        other.mkdir()
        with self.assertRaisesRegex(ValueError, 'must not exist'):
            module.atomic_promote(other, self.root)
        self.assertTrue(other.exists())
        self.assertEqual((self.root / 'file').read_bytes(), b'bytes')

    def test_atomic_rename_rejects_destination_created_after_check(self):
        stage = self.parent / 'stage'
        stage.mkdir()
        def concurrent_creation(destination):
            destination.mkdir()
            (destination / 'protected').write_bytes(b'keep')
        with patch.object(module, 'absent', side_effect=concurrent_creation):
            with self.assertRaisesRegex(ValueError, 'appeared'):
                module.atomic_promote(stage, self.root)
        self.assertTrue(stage.exists())
        self.assertEqual((self.root / 'protected').read_bytes(), b'keep')

    def test_invalid_and_development_identity_refuse_before_download(self):
        with patch.object(module, 'download') as download:
            for tag, revision in (('dev', 'a' * 40), ('../../tag', 'a' * 40), ('v1.2.3', 'bad')):
                with self.assertRaises(ValueError):
                    module.retrieve(tag, revision, self.root)
            download.assert_not_called()


if __name__ == '__main__':
    unittest.main()
