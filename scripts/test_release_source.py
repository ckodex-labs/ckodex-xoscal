"""Real Git-tree controls for source provenance; no hosted signing claim."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('release_source', Path(__file__).with_name('release-source.py'))
source = importlib.util.module_from_spec(spec)
spec.loader.exec_module(source)


class ProductionSourceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / 'caller'
        self.root.mkdir()
        self.run_git('init', '--quiet', '--template=')
        (self.root / '.gitignore').write_text('go.work\nvendor/\nsite/.env\nsite/frameworks/\n')
        (self.root / 'main.go').write_text('package main\nfunc main() {}\n')
        (self.root / 'site').mkdir()
        (self.root / 'site/portal.html').write_text('tracked presentation\n')
        (self.root / 'install.sh').write_text('#!/bin/sh\nexit 0\n')
        (self.root / 'install.sh').chmod(0o755)
        self.commit()
        self.destination = Path(self.temp.name) / 'production'
        self.receipt = Path(self.temp.name) / 'receipt.json'

    def run_git(self, *args):
        return subprocess.check_output(['git', '-C', str(self.root), *args],
                                       env=source.git_environment(), text=True).strip()

    def commit(self):
        self.run_git('add', '.')
        self.run_git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid',
                     'commit', '--quiet', '-m', 'tracked fixture')
        self.revision = self.run_git('rev-parse', 'HEAD')
        self.run_git('tag', '-f', 'v1.2.3')

    def produce(self, **kwargs):
        return source.production_source(self.root, kwargs.get('tag', 'v1.2.3'),
                                        kwargs.get('revision', self.revision), kwargs.get('version', 'v1.2.3'),
                                        self.destination, self.receipt)

    def test_ignored_inputs_and_caller_credentials_hooks_are_excluded(self):
        for path in ('go.work', 'vendor/modules.txt', 'vendor/attacker/injected.go',
                     'site/.env', 'site/frameworks/injected.json'):
            target = self.root / path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text('caller injection\n')
        self.run_git('config', 'http.https://github.com/.extraheader', 'FIXTURE-CHECKOUT-CREDENTIAL')
        hook = self.root / '.git/hooks/post-checkout'
        hook.parent.mkdir(parents=True, exist_ok=True)
        hook.write_text('fixture hook must not survive\n')
        (self.root / '.git/info').mkdir(exist_ok=True)
        (self.root / '.git/info/attributes').write_text('main.go export-ignore\n')
        self.assertEqual(self.run_git('status', '--porcelain', '--untracked-files=all'), '')
        self.produce()
        for path in ('go.work', 'vendor', 'site/.env', 'site/frameworks', '.git/hooks/post-checkout'):
            self.assertFalse((self.destination / path).exists(), path)
        self.assertEqual((self.destination / 'main.go').read_bytes(), (self.root / 'main.go').read_bytes())
        self.assertTrue((self.destination / 'install.sh').stat().st_mode & 0o111)
        config = (self.destination / '.git/config').read_text()
        self.assertNotIn('FIXTURE-CHECKOUT-CREDENTIAL', config)
        self.assertNotIn(str(self.root), config)
        self.assertEqual(source.git(self.destination, 'remote', 'get-url', 'origin'), source.PUBLIC_ORIGIN)
        self.assertEqual(source.git(self.destination, 'rev-parse', 'HEAD'), self.revision)
        self.assertEqual(source.git(self.destination, 'rev-parse', 'refs/tags/v1.2.3^{commit}'), self.revision)
        self.assertEqual(source.git(self.destination, 'status', '--porcelain', '--untracked-files=all'), '')
        self.assertEqual(json.loads(self.receipt.read_text())['source_revision'], self.revision)

    def test_archive_attributes_cannot_omit_or_substitute_tracked_bytes(self):
        (self.root / '.gitattributes').write_text('main.go export-ignore\nsite export-ignore\nmarker.txt export-subst\n')
        (self.root / 'marker.txt').write_text('$Format:%H$\n')
        self.commit()
        self.produce()
        for path in ('main.go', 'site/portal.html', 'marker.txt', '.gitattributes'):
            self.assertEqual((self.destination / path).read_bytes(), (self.root / path).read_bytes(), path)

    def test_dirty_and_untracked_inputs_are_rejected_before_admission(self):
        for path in ('main.go', 'untracked.go'):
            with self.subTest(path=path):
                target = self.root / path
                previous = target.read_bytes() if target.exists() else None
                target.write_text('dirty input\n')
                with self.assertRaisesRegex(ValueError, 'clean'):
                    self.produce()
                self.assertFalse(self.destination.exists())
                self.assertFalse(self.receipt.exists())
                if previous is None:
                    target.unlink()
                else:
                    target.write_bytes(previous)

    def test_identity_mismatch_is_rejected(self):
        for values in ({'revision': 'a' * 40}, {'version': 'v9.9.9'}, {'tag': 'dev'}):
            with self.subTest(values=values), self.assertRaises(ValueError):
                self.produce(**values)
        self.assertFalse(self.destination.exists())

    def test_semver_build_metadata_is_rejected_before_git_lookup(self):
        for tag in ('v1.2.3+build.1', 'v1.2.3-rc.1+build.2'):
            with self.subTest(tag=tag), self.assertRaisesRegex(ValueError, 'OCI-compatible'):
                self.produce(tag=tag, version=tag)
        self.assertFalse(self.destination.exists())
        self.assertFalse(self.receipt.exists())

    def test_oci_tag_length_limit_is_enforced_before_git_lookup(self):
        tag = 'v1.2.3-' + 'a' * 122
        self.assertEqual(len(tag), 129)
        with self.assertRaisesRegex(ValueError, '128 ASCII characters'):
            self.produce(tag=tag, version=tag)
        self.assertFalse(self.destination.exists())
        self.assertFalse(self.receipt.exists())

    def test_standard_prerelease_tag_is_preserved(self):
        tag = 'v1.2.3-rc.1'
        self.run_git('tag', tag)
        self.produce(tag=tag, version=tag)
        self.assertEqual(json.loads(self.receipt.read_text())['release_tag'], tag)
        self.assertEqual(source.git(self.destination, 'rev-parse', 'refs/tags/' + tag + '^{commit}'), self.revision)

    def test_existing_destination_is_not_overwritten(self):
        self.destination.mkdir()
        (self.destination / 'canary').write_text('preserve\n')
        with self.assertRaisesRegex(ValueError, 'must not exist'):
            self.produce()
        self.assertEqual((self.destination / 'canary').read_text(), 'preserve\n')

    def test_tracked_symlink_is_rejected_without_admission(self):
        (self.root / 'escape').symlink_to('../caller-config')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'unsupported tracked'):
            self.produce()
        self.assertFalse(self.destination.exists())
        self.assertFalse(self.receipt.exists())


if __name__ == '__main__':
    unittest.main()
