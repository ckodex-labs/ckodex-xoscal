"""Applicability negatives use actual structured OSCAL fields, not text search."""
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('semantic_scope', Path(__file__).with_name('oscal-semantic-scope.py'))
scope = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scope)


class SemanticScopeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / 'frameworks'
        for index in range(35):
            self.write('framework-' + str(index) + '/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3'}}})
        self.write('framework-0/profile.json', {'profile': {'metadata': {'oscal-version': '1.2.3'}}})

    def write(self, name, document):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(document))
        return path

    def test_reviewed_subset_binds_every_exact_subject_and_versions(self):
        result = scope.evaluate(self.root)
        self.assertEqual(result['status'], 'passed')
        self.assertEqual(result['subject_counts'], {'catalog': 35, 'profile': 1})
        self.assertEqual(result['producer']['semantic_model_version'], '1.1.2')
        self.assertEqual(result['producer']['structural_model_version'], '1.2.3')
        self.assertEqual(len(result['subjects']), 36)
        for subject in result['subjects']:
            self.assertEqual(subject['sha256'], hashlib.sha256((self.root / subject['path']).read_bytes()).hexdigest())

    def test_changed_fields_are_rejected_even_when_empty_or_deeply_nested(self):
        for feature in ('hashes', 'locations', 'location-uuids', 'resource-fragment', 'links', 'with-child-controls'):
            with self.subTest(feature=feature):
                self.write('framework-0/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3'}, 'groups': [{'controls': [{'parts': [{feature: []}]}]}]}})
                result = scope.evaluate(self.root)
                self.assertEqual(result['status'], 'failed')
                self.assertTrue(any(f.get('feature') == feature for f in result['findings']))

    def test_status_property_is_rejected_in_any_namespace(self):
        self.write('framework-0/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3'}, 'controls': [{'props': [{'name': 'status', 'ns': 'https://example.invalid/custom', 'value': 'withdrawn'}]}]}})
        self.assertTrue(any(f.get('feature') == 'status-property' for f in scope.evaluate(self.root)['findings']))

    def test_text_mentions_are_not_structured_feature_use(self):
        self.write('framework-0/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3'}, 'controls': [{'title': 'hashes locations links resource-fragment status with-child-controls', 'props': [{'name': 'label', 'value': 'status'}]}]}})
        self.assertEqual(scope.evaluate(self.root)['status'], 'passed')

    def test_missing_or_additional_subject_blocks_scope(self):
        (self.root / 'framework-0/profile.json').unlink()
        self.assertEqual(scope.evaluate(self.root)['status'], 'failed')
        self.write('framework-0/profile.json', {'profile': {'metadata': {'oscal-version': '1.2.3'}}})
        self.write('framework-extra/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3'}}})
        self.assertEqual(scope.evaluate(self.root)['status'], 'failed')

    def test_profile_scope_is_checked_independently(self):
        self.write('framework-0/profile.json', {'profile': {'metadata': {'oscal-version': '1.2.3'}, 'imports': [{'include-controls': [{'with-child-controls': 'yes'}]}]}})
        result = scope.evaluate(self.root)
        self.assertEqual(result['status'], 'failed')
        self.assertTrue(any(f.get('path') == 'framework-0/profile.json' and f.get('feature') == 'with-child-controls' for f in result['findings']))

    def test_changed_subject_bytes_change_receipt_digest(self):
        before = scope.evaluate(self.root)['subjects'][0]['sha256']
        self.write('framework-0/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.2.3', 'title': 'changed'}}})
        self.assertNotEqual(scope.evaluate(self.root)['subjects'][0]['sha256'], before)

    def test_wrong_version_and_ambiguous_root_do_not_pass(self):
        self.write('framework-0/catalog.json', {'catalog': {'metadata': {'oscal-version': '1.1.2'}}})
        self.assertEqual(scope.evaluate(self.root)['status'], 'failed')
        self.write('framework-0/catalog.json', {'catalog': {}, 'profile': {}})
        with self.assertRaisesRegex(ValueError, 'unexpected OSCAL root'):
            scope.evaluate(self.root)

    def test_symlink_subject_is_rejected(self):
        path = self.root / 'framework-0/catalog.json'
        target = Path(self.temp.name) / 'outside.json'
        path.rename(target)
        path.symlink_to(target)
        with self.assertRaisesRegex(ValueError, 'regular file'):
            scope.evaluate(self.root)

    def test_failed_scope_cli_retains_raw_receipt_and_nonzero_status(self):
        (self.root / 'framework-0/profile.json').unlink()
        receipt = Path(self.temp.name) / 'scope.json'
        result = subprocess.run([sys.executable, str(Path(__file__).with_name('oscal-semantic-scope.py')), str(self.root), str(receipt)], capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(json.loads(receipt.read_text())['status'], 'failed')


if __name__ == '__main__':
    unittest.main()
