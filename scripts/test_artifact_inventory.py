#!/usr/bin/env python3
"""Both pinned scan modes and their exact raw observations remain mandatory."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('inventory', Path(__file__).resolve().parents[1] / 'dagger/artifact_inventory.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


class InventoryTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        for mode in ('fs', 'rootfs'):
            self.report(mode, [])
            (self.root / f'trivy-{mode}.status').write_text('0\n')

    def report(self, mode, targets):
        data = {'SchemaVersion': 2, 'Trivy': {'Version': '0.72.0'}, 'ArtifactName': '/input/unpacked', 'ArtifactType': 'filesystem', 'Results': targets}
        (self.root / f'trivy-{mode}-results.json').write_text(json.dumps(data))

    def test_preserves_both_modes_all_findings_and_overlapping_targets(self):
        target = {'Target': 'binary', 'Packages': [{'Name': 'actual', 'Version': '1'}], 'Vulnerabilities': [{'VulnerabilityID': 'FIXED', 'Severity': 'CRITICAL', 'FixedVersion': '2'}]}
        self.report('fs', [target]); self.report('rootfs', [target])
        output = helper.replay_merge(self.root)
        self.assertEqual([x['XoscalScanMode'] for x in output['Results']], ['fs', 'rootfs'])
        self.assertEqual(len(output['Results']), 2)
        self.assertEqual(output['Results'][1]['Vulnerabilities'], target['Vulnerabilities'])
        self.assertEqual(set(output['XoscalRawReports']), {'fs', 'rootfs'})

    def test_missing_or_failed_mode_never_produces_merged_report(self):
        (self.root / 'trivy-rootfs.status').write_text('2\n')
        with self.assertRaises(ValueError): helper.replay_merge(self.root)
        (self.root / 'trivy-rootfs.status').unlink()
        with self.assertRaises(FileNotFoundError): helper.replay_merge(self.root)

    def test_wrong_version_subject_or_malformed_targets_rejected(self):
        for field, value in (('Trivy', {'Version': '0.71.0'}), ('ArtifactName', '/other'), ('Results', {}), ('Results', [{'Target': 'a', 'XoscalScanMode': 'forged'}])):
            with self.subTest(field=field, value=value):
                self.report('rootfs', [])
                path = self.root / 'trivy-rootfs-results.json'
                data = json.loads(path.read_text()); data[field] = value; path.write_text(json.dumps(data))
                with self.assertRaises(ValueError): helper.replay_merge(self.root)

    def test_empty_reports_remain_empty_for_strict_admission(self):
        self.assertEqual(helper.replay_merge(self.root)['Results'], [])

    def test_raw_metadata_changes_alter_binding(self):
        before = helper.replay_merge(self.root)
        path = self.root / 'trivy-fs-results.json'; path.write_text(path.read_text() + '\n')
        after = helper.replay_merge(self.root)
        self.assertNotEqual(before['XoscalRawReports'], after['XoscalRawReports'])


if __name__ == '__main__':
    unittest.main()
