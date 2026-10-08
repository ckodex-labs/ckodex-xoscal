"""Exact-manifest graph checks using a minimized actual refused SDK inventory."""
import copy
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'dagger'))
from go_module_relationships import declared_graph, parse_manifest
from enrich_sbom import enrich

FIXTURE = Path(__file__).parent / 'fixtures/go-sdk-manifest'
POLICY = {'version': 1, 'role': 'release-distributor', 'entity': {'name': 'ckodex-labs', 'url': ['https://github.com/ckodex-labs/ckodex-xoscal']}}


class GoManifestRelationships(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.publisher = Path(self.temp.name)
        self.manifest = (FIXTURE / 'go.mod').read_bytes()
        (self.publisher / 'go.mod').write_bytes(self.manifest)
        (self.publisher / 'publishers.json').write_text(json.dumps({'version': 1, 'records': [], 'unresolved': []}))
        self.raw = json.loads((FIXTURE / 'inventory.json').read_text())

    def test_actual_twelve_declared_requirements_and_roles(self):
        project, requirements = parse_manifest(self.manifest)
        self.assertEqual(project, 'github.com/mchorfa/xoscal/proto/oscal')
        self.assertEqual(len(requirements), 12)
        self.assertEqual(sum(r['indirect'] for r in requirements), 6)
        result, receipt = enrich(copy.deepcopy(self.raw), POLICY, self.publisher)
        detail = receipt['go_module_relationships']
        self.assertEqual(detail['manifest_sha256'], hashlib.sha256(self.manifest).hexdigest())
        self.assertEqual(len(detail['requirements']), 12)
        self.assertEqual(receipt['root_declares'], [detail['project_ref']])
        edge = next(e for e in result['dependencies'] if e['ref'] == detail['project_ref'])
        self.assertEqual(edge['dependsOn'], sorted(r['bom_ref'] for r in detail['requirements']))
        # Declaration edges must never manufacture supplier or installed-byte evidence.
        self.assertEqual(len(receipt['unresolved_components']), 13)
        for c in result['components']:
            if c['type'] == 'library':
                self.assertNotIn('supplier', c)
                self.assertNotIn(c['bom-ref'], receipt['root_contains'])

    def test_existing_observed_edges_are_preserved(self):
        declared, _ = declared_graph(copy.deepcopy(self.raw), self.publisher)
        file_ref = next(c['bom-ref'] for c in self.raw['components'] if c['type'] == 'file')
        self.raw['dependencies'] = [{'ref': declared[0], 'dependsOn': [file_ref]}]
        _, detail = declared_graph(self.raw, self.publisher)
        self.assertIn(file_ref, self.raw['dependencies'][0]['dependsOn'])
        self.assertEqual(len(detail['requirements']), 12)

    def test_missing_manifest_refused(self):
        (self.publisher / 'go.mod').unlink()
        with self.assertRaises(ValueError):
            declared_graph(self.raw, self.publisher)
        with self.assertRaises(ValueError):
            declared_graph(self.raw, None)

    def test_tampered_manifest_refused(self):
        (self.publisher / 'go.mod').write_bytes(self.manifest + b'\n')
        with self.assertRaises(ValueError):
            declared_graph(self.raw, self.publisher)

    def test_duplicate_and_contradictory_inventory_refused(self):
        for change in ('component', 'file', 'version', 'purl', 'location', 'property', 'missing', 'unreferenced', 'hash'):
            with self.subTest(change=change):
                raw = copy.deepcopy(self.raw)
                dependency = raw['components'][1]
                if change == 'component': raw['components'].append(copy.deepcopy(dependency))
                elif change == 'file': raw['components'].append(copy.deepcopy(raw['components'][-1]))
                elif change == 'version': dependency['version'] = 'v0.0.1'
                elif change == 'purl': dependency['purl'] += '?other=identity'
                elif change == 'location': dependency['properties'][-1]['value'] = '/other/go.mod'
                elif change == 'property': dependency['properties'].append(copy.deepcopy(dependency['properties'][0]))
                elif change == 'missing': raw['components'].pop(1)
                elif change == 'unreferenced':
                    dependency = copy.deepcopy(dependency)
                    dependency.update({'name': 'example.org/unreferenced', 'bom-ref': 'extra', 'purl': 'pkg:golang/example.org/unreferenced@v1.45.0'})
                    raw['components'].append(dependency)
                elif change == 'hash': raw['components'][-1]['hashes'].append({'alg': 'SHA-256', 'content': hashlib.sha256(self.manifest).hexdigest()})
                with self.assertRaises(ValueError): declared_graph(raw, self.publisher)

    def test_dangling_duplicate_and_self_edges_refused(self):
        project = self.raw['components'][0]['bom-ref']
        for edges in ([{'ref': project, 'dependsOn': ['absent']}], [{'ref': project, 'dependsOn': [project]}], [{'ref': project}, {'ref': project}]):
            with self.subTest(edges=edges), self.assertRaises(ValueError):
                raw = copy.deepcopy(self.raw)
                raw['dependencies'] = edges
                declared_graph(raw, self.publisher)

    def test_unsupported_and_duplicate_declarations_refused(self):
        samples = [self.manifest + b'replace example.org/a => ./a\n', self.manifest + b'module duplicate\n', self.manifest + b'require google.golang.org/grpc v1.83.2\n', self.manifest.replace(b'v1.83.2', b'latest'), self.manifest + b'require (\n', self.manifest.replace(b'// indirect', b'// manufactured', 1)]
        for data in samples:
            with self.subTest(data=data), self.assertRaises(ValueError):
                parse_manifest(data)

    def test_single_require_and_explicit_indirect_role(self):
        name, requirements = parse_manifest(b'module example.org/sdk\ngo 1.25.0\nrequire example.org/a v1.2.3 // indirect\n')
        self.assertEqual(name, 'example.org/sdk')
        self.assertEqual(requirements, [{'name': 'example.org/a', 'version': 'v1.2.3', 'indirect': True}])

    def test_go_and_toolchain_directives_have_distinct_spellings(self):
        valid = b'module example.org/sdk\ngo 1.25.0\ntoolchain go1.26.2\nrequire example.org/a v1.2.3\n'
        self.assertEqual(parse_manifest(valid)[0], 'example.org/sdk')
        for data in (valid.replace(b'go 1.25.0', b'go go1.25.0'), valid.replace(b'toolchain go1.26.2', b'toolchain 1.26.2')):
            with self.subTest(data=data), self.assertRaises(ValueError):
                parse_manifest(data)

    def test_additional_scanner_modules_require_retained_reachable_edges(self):
        dependency = copy.deepcopy(self.raw['components'][1])
        dependency.update({'name': 'example.org/observed', 'bom-ref': 'observed', 'purl': 'pkg:golang/example.org/observed@v1.45.0'})
        dependency['properties'].extend([{'name': 'syft:cpe23', 'value': 'observed-a'}, {'name': 'syft:cpe23', 'value': 'observed-b'}])
        self.raw['components'].append(dependency)
        with self.assertRaises(ValueError):
            declared_graph(copy.deepcopy(self.raw), self.publisher)
        declared_ref = self.raw['components'][1]['bom-ref']
        self.raw['dependencies'] = [{'ref': declared_ref, 'dependsOn': ['observed']}]
        _, detail = declared_graph(self.raw, self.publisher)
        self.assertEqual(detail['observed_additional_modules'][0]['bom_ref'], 'observed')
        self.assertEqual(self.raw['dependencies'][0], {'ref': declared_ref, 'dependsOn': ['observed']})
        self.assertEqual(len(detail['requirements']), 12)

    def test_actual_resolved_scan_preserves_eight_observed_modules(self):
        raw = json.loads((FIXTURE / 'observed-inventory.json').read_text())
        original_edges = copy.deepcopy(raw['dependencies'])
        result, receipt = enrich(raw, POLICY, self.publisher)
        detail = receipt['go_module_relationships']
        self.assertEqual(len(detail['requirements']), 12)
        self.assertEqual(len(detail['observed_additional_modules']), 8)
        for edge in original_edges:
            actual = next(e for e in result['dependencies'] if e['ref'] == edge['ref'])
            self.assertTrue(set(edge['dependsOn']) <= set(actual['dependsOn']))
        self.assertEqual(len(receipt['unresolved_components']), 21)

    def test_observed_extras_require_exact_valid_module_identity(self):
        for name, version in (('example.org/module', 'UNKNOWN'), ('example.org/module', None), ('example.org/module', 'latest'), ('example.org/../module', 'v1.2.3'), (None, 'v1.2.3')):
            with self.subTest(name=name, version=version):
                raw = json.loads((FIXTURE / 'observed-inventory.json').read_text())
                extra = raw['components'][0]
                extra.update(name=name, version=version, purl='pkg:golang/' + (name or 'unknown') + ('@' + version if version not in (None, 'UNKNOWN') else ''))
                with self.assertRaises(ValueError):
                    declared_graph(raw, self.publisher)


if __name__ == '__main__':
    unittest.main()
