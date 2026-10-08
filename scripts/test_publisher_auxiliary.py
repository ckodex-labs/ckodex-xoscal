#!/usr/bin/env python3
"""Synthetic auxiliary replay refusal controls; not artifact production proof."""
import tempfile
import unittest
from pathlib import Path
from test_release_requirements import make_candidate
import release_requirements as requirements


class PublisherAuxiliaryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        make_candidate(self.root)

    def replay(self, stage, subject=None):
        gate = requirements.load_module('publisher_auxiliary_gate', 'analysis-gate.py')
        requirements.replay_gate(self.root, stage, 'sbom', gate,
                                 self.root / 'evidence/policy/analysis-exceptions.json', subject)

    def test_go_mod_exact_original_archive_is_accepted(self):
        self.replay('evidence/sdk-go-sbom', 'sdk/go.zip')

    def test_go_mod_symlink_to_exact_bytes_is_rejected(self):
        path = self.root / 'evidence/sdk-go-sbom/publishers/go.mod'
        target = self.root / 'same-go.mod'
        target.write_bytes(path.read_bytes())
        path.unlink()
        path.symlink_to(target)
        with self.assertRaisesRegex(ValueError, 'publisher auxiliary differs'):
            self.replay('evidence/sdk-go-sbom', 'sdk/go.zip')

    def test_go_mod_requires_original_sdk_subject(self):
        (self.root / 'evidence/binary-sbom/publishers/go.mod').write_bytes(b'module example.com/unbound\n')
        with self.assertRaisesRegex(ValueError, 'publisher raw file set differs'):
            self.replay('evidence/binary-sbom')

    def test_unlisted_publisher_file_still_rejects(self):
        (self.root / 'evidence/sdk-go-sbom/publishers/unlisted').write_bytes(b'not in ledger')
        with self.assertRaisesRegex(ValueError, 'publisher raw file set differs'):
            self.replay('evidence/sdk-go-sbom', 'sdk/go.zip')


if __name__ == '__main__':
    unittest.main()
