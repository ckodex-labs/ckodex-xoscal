#!/usr/bin/env python3
"""Admission regression tests: tool errors, missing evidence and exact exceptions."""
import datetime as dt
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('analysis-gate.py')
spec = importlib.util.spec_from_file_location('analysis_gate', SCRIPT)
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class AdmissionTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.policy = self.root / 'exceptions.json'
        self.write('exceptions.json', {'version': 1, 'exceptions': []})

    def write(self, name, value):
        (self.root / name).write_text(json.dumps(value))

    def status(self, name, value=0):
        (self.root / (name + '.status')).write_text(str(value))

    def sbom(self, fail=False):
        self.write('sbom.cyclonedx.json', {'bomFormat': 'CycloneDX', 'specVersion': '1.6', 'version': 1, 'components': [{'name': 'actual-package'}]})
        state = 'not-satisfied' if fail else 'satisfied'
        self.finding = {'title': 'SBOM-NTIA-SUPPLIER', 'description': 'missing supplier', 'target': {'target-id': 'sbom-ntia-supplier', 'status': {'state': state}}}
        rollup = {'title': 'NTIA roll-up', 'description': f'Overall verdict for NTIA Minimum Elements: {int(fail)} error(s), 0 warning(s), 0 informational finding(s).', 'target': {'target-id': 'ntiaminimum-compliance', 'status': {'state': state}}}
        self.write('sbom-assessment-results.oscal.json', {'assessment-results': {'metadata': {'oscal-version': '1.1.2'}, 'results': [{'findings': [rollup, self.finding], 'observations': [{'uuid': 'observation', 'props': [{'name': 'severity', 'value': 'error'}]}] if fail else []}]}})
        if fail:
            doc=json.loads((self.root/'sbom-assessment-results.oscal.json').read_text());doc['assessment-results']['results'][0]['findings'][1]['related-observations']=[{'observation-uuid':'observation'}];self.write('sbom-assessment-results.oscal.json',doc)
        self.status('sbom-validation', int(fail))

    def exception(self, **overrides):
        expiry = (dt.datetime.now(dt.timezone.utc).date() + dt.timedelta(days=10)).isoformat()
        exception = {'id': 'approved-single-finding', 'approved_by': 'test reviewer', 'rationale': 'test exact scope', 'expires': expiry, 'sbom_sha256': hashlib.sha256((self.root / 'sbom.cyclonedx.json').read_bytes()).hexdigest(), 'finding_sha256': gate.fingerprint(self.finding)}
        exception.update(overrides)
        self.write('exceptions.json', {'version': 1, 'exceptions': [exception]})

    def security(self, issues=None):
        for name in ('govulncheck', 'gosec', 'gosec-sarif'):
            self.status(name)
        messages = [{'config': {'scan_level': 'symbol'}}, {'progress': {'message': 'Scanning your code...'}}]
        (self.root / 'govulncheck.json').write_text('\n'.join(json.dumps(x, indent=2) for x in messages))
        self.write('gosec-results.json', {'Golang errors': {}, 'Stats': {'files': 5}, 'Issues': issues or []})
        self.write('gosec-results.sarif', {'version': '2.1.0', 'runs': [{'results': issues or []}]})

    def image(self, vulns=None):
        self.status('trivy')
        (self.root / 'image.sha256').write_text('a' * 64 + '  /input/image.tar\n')
        self.write('trivy-results.json', {'SchemaVersion': 2, 'Trivy': {'Version': '0.72.0'}, 'ArtifactType': 'container_image', 'ArtifactName': '/input/image.tar', 'Results': [{'Target': 'debian', 'Packages': [{'Name': 'libc', 'Version': '1'}], 'Vulnerabilities': vulns or []}]})

    def test_sbom_passes(self):
        self.sbom()
        self.assertEqual(gate.sbom_gate(self.root, self.policy)['blocked'], [])

    def test_sbom_violations_block_even_with_report(self):
        self.sbom(True)
        self.assertTrue(gate.sbom_gate(self.root, self.policy)['blocked'])

    def test_warning_only_satisfied_rollup_still_blocks(self):
        self.sbom(True)
        doc=json.loads((self.root/'sbom-assessment-results.oscal.json').read_text());r=doc['assessment-results']['results'][0]
        r['findings'][0]['target']['status']['state']='satisfied'
        r['findings'][0]['description']='Overall verdict for NTIA Minimum Elements: 0 error(s), 1 warning(s), 0 informational finding(s).'
        r['observations'][0]['props'][0]['value']='warning'
        self.write('sbom-assessment-results.oscal.json',doc);self.status('sbom-validation')
        self.assertEqual(gate.sbom_gate(self.root,self.policy)['violations'],1)
        self.assertTrue(gate.sbom_gate(self.root,self.policy)['blocked'])

    def test_error_count_cannot_be_labeled_satisfied(self):
        self.sbom(True);doc=json.loads((self.root/'sbom-assessment-results.oscal.json').read_text())
        doc['assessment-results']['results'][0]['findings'][0]['target']['status']['state']='satisfied'
        self.write('sbom-assessment-results.oscal.json',doc)
        with self.assertRaises(gate.GateError):gate.sbom_gate(self.root,self.policy)

    def test_duplicate_verdict_properties_rejected(self):
        self.sbom()
        (self.root/'sbom.cyclonedx.json').write_text('{"bomFormat":"CycloneDX","bomFormat":"ignored","components":[]}')
        with self.assertRaises(gate.GateError):gate.sbom_gate(self.root,self.policy)

    def test_exception_covers_only_exact_bytes(self):
        self.sbom(True)
        self.exception()
        self.assertEqual(gate.sbom_gate(self.root, self.policy)['blocked'], [])
        with (self.root / 'sbom.cyclonedx.json').open('a') as stream:
            stream.write('\n')
        self.assertTrue(gate.sbom_gate(self.root, self.policy)['blocked'])

    def test_exception_covers_only_exact_finding(self):
        self.sbom(True)
        self.exception(finding_sha256='0' * 64)
        self.assertTrue(gate.sbom_gate(self.root, self.policy)['blocked'])

    def test_expired_exception_fails(self):
        self.sbom(True)
        self.exception(expires='2000-01-01')
        with self.assertRaises(gate.GateError):
            gate.sbom_gate(self.root, self.policy)

    def test_tool_operational_error_blocks(self):
        self.sbom()
        self.status('sbom-validation', 2)
        with self.assertRaises(gate.GateError):
            gate.sbom_gate(self.root, self.policy)

    def test_malformed_verdict_blocks(self):
        self.sbom()
        self.write('sbom-assessment-results.oscal.json', {'assessment-results': {'results': [{}]}})
        with self.assertRaises(gate.GateError):
            gate.sbom_gate(self.root, self.policy)

    def test_malformed_sbom_document_blocks(self):
        self.sbom()
        self.write('sbom.cyclonedx.json', {'bomFormat': 'CycloneDX', 'specVersion': '1.6', 'version': 1, 'components': [1]})
        with self.assertRaises(gate.GateError):
            gate.sbom_gate(self.root, self.policy)

    def test_security_clean_passes(self):
        self.security()
        self.assertEqual(gate._security_gate(self.root)['blocked'], [])

    def test_high_confidence_high_severity_blocks(self):
        self.security([{'severity': 'HIGH', 'confidence': 'HIGH', 'rule_id': 'G101'}])
        self.assertEqual(gate._security_gate(self.root)['blocked'], ['G101'])

    def test_lower_findings_retained_without_broadening_threshold(self):
        self.security([{'severity': 'HIGH', 'confidence': 'LOW', 'rule_id': 'G101'}])
        self.assertEqual(gate._security_gate(self.root)['findings'], 1)
        self.assertEqual(gate._security_gate(self.root)['blocked'], [])

    def test_source_suppression_requires_specific_rationale(self):
        issue = {'severity': 'HIGH', 'confidence': 'HIGH', 'rule_id': 'G703', 'suppressions': [{'kind': 'inSource', 'justification': 'Operator supplied output path, independently reviewed'}]}
        self.security([issue])
        self.assertEqual(gate._security_gate(self.root)['blocked'], [])
        self.assertEqual(gate._security_gate(self.root)['findings'], 1)
        issue['suppressions'][0]['justification'] = ''
        self.security([issue])
        self.assertEqual(gate._security_gate(self.root)['blocked'], ['G703'])

    def test_external_suppression_cannot_bypass_policy(self):
        self.security([{'severity': 'HIGH', 'confidence': 'HIGH', 'rule_id': 'G703', 'suppressions': [{'kind': 'external', 'justification': 'Global exclude'}]}])
        self.assertEqual(gate._security_gate(self.root)['blocked'], ['G703'])

    def test_golang_processing_error_blocks(self):
        self.security()
        self.write('gosec-results.json', {'Golang errors': {'broken.go': ['invalid Go']}, 'Stats': {'files': 5}, 'Issues': []})
        with self.assertRaises(gate.GateError):
            gate._security_gate(self.root)

    def test_json_govulncheck_called_finding_blocks(self):
        self.security()
        with (self.root / 'govulncheck.json').open('a') as stream:
            stream.write(json.dumps({'finding': {'osv': 'GO-TEST', 'trace': [{'module': 'example', 'function': 'Called'}]}}))
        self.assertIn('GO-TEST', gate._security_gate(self.root)['blocked'])

    def test_image_thresholds_preserved(self):
        self.image([{'Severity': 'CRITICAL', 'VulnerabilityID': 'UNFIXED'}, {'Severity': 'HIGH', 'VulnerabilityID': 'HIGH', 'FixedVersion': '2'}])
        self.assertEqual(gate.image_gate(self.root)['blocked'], [])
        self.assertEqual(gate.image_gate(self.root, ci=True)['blocked'], ['UNFIXED', 'HIGH'])
        self.image([{'Severity': 'CRITICAL', 'VulnerabilityID': 'FIXED', 'FixedVersion': '2'}])
        self.assertEqual(gate.image_gate(self.root)['blocked'], ['FIXED'])

    def test_malformed_image_blocks(self):
        self.image()
        self.write('trivy-results.json', {'SchemaVersion': 2, 'Results': []})
        with self.assertRaises(gate.GateError):
            gate.image_gate(self.root)

    def test_artifact_admission_binds_original_digest(self):
        self.image()
        self.status('extraction')
        (self.root / 'artifact.sha256').write_text('b' * 64 + '  /input/artifact\n')
        report = json.loads((self.root / 'trivy-results.json').read_text())
        report['ArtifactType'] = 'filesystem'
        report['ArtifactName'] = '/input/unpacked'
        self.write('trivy-results.json', report)
        self.assertEqual(gate.artifact_gate(self.root)['artifact_sha256'], 'b' * 64)
        self.status('extraction', 1)
        with self.assertRaises(gate.GateError):
            gate.artifact_gate(self.root)

    def test_empty_package_inventory_never_passes(self):
        self.image()
        report = json.loads((self.root / 'trivy-results.json').read_text())
        report['Results'][0]['Packages'] = []
        self.write('trivy-results.json', report)
        with self.assertRaises(gate.GateError):
            gate.image_gate(self.root)

    def test_malformed_sbom_subject_never_passes(self):
        self.sbom()
        (self.root / 'subject.sha256').write_text('not a digest')
        with self.assertRaises(gate.GateError):
            gate.sbom_gate(self.root, self.policy)

    def test_missing_evidence_cli_returns_failure_summary(self):
        output = self.root / 'summary.json'
        result = subprocess.run(['python3', str(SCRIPT), 'sbom', '--evidence', str(self.root), '--exceptions', str(self.policy), '--output', str(output)], capture_output=True)
        self.assertEqual(result.returncode, 1)
        self.assertFalse(json.loads(output.read_text())['passed'])


if __name__ == '__main__':
    unittest.main()
