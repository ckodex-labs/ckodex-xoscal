#!/usr/bin/env python3
"""Synthetic required-output refusal tests; not scanner or hosted release proof."""
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import shutil
import sys
import tempfile
import tarfile
import unittest
from unittest import mock
import zipfile

import release_inventory as inventory
import release_requirements as requirements
from test_inspect_image_archive import make_layout, write_archive, oci
from test_frontend_inventory import make_fixture as make_frontend_fixture, f as FRONTEND
from test_presentation_browser_verify import make_fixture as make_browser_fixture

SDK = requirements.load_module('requirements_sdk_fixture', 'sdk-verify.py')
GATE = requirements.load_module('requirements_analysis_fixture', 'analysis-gate.py')
POLICY = requirements.read_json(requirements.SOURCE_ROOT / 'docs/release-policy.json')
EXCEPTIONS = requirements.SOURCE_ROOT / 'docs/analysis-exceptions.json'
sys.path.insert(0, str(requirements.SOURCE_ROOT / 'dagger'))
ENRICH_SPEC = importlib.util.spec_from_file_location('requirements_enrichment_fixture', requirements.SOURCE_ROOT / 'dagger/enrich_sbom.py')
ENRICH = importlib.util.module_from_spec(ENRICH_SPEC)
ENRICH_SPEC.loader.exec_module(ENRICH)
MERGE_SPEC = importlib.util.spec_from_file_location('requirements_artifact_fixture', requirements.SOURCE_ROOT / 'dagger/artifact_inventory.py')
MERGE = importlib.util.module_from_spec(MERGE_SPEC)
MERGE_SPEC.loader.exec_module(MERGE)


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, sort_keys=True) + '\n')


def put(root, path, data=b'fixture'):
    target = root / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_bytes(data)
    return target


def fixture_receipt(root, stage, kind):
    directory = root / stage
    if kind == 'sbom':
        raw = GATE.sbom_gate(directory, root / 'evidence/policy/analysis-exceptions.json')
    elif kind == 'frontend-release':
        raw = GATE.frontend_gate(directory)
    elif kind == 'security':
        raw = GATE._security_gate(directory)
    elif kind == 'image-release':
        raw = GATE.image_gate(directory)
    else:
        raw = GATE.artifact_gate(directory)
    write_json(directory / 'analysis-gate.json', {'policy_version': 1, 'kind': kind, 'passed': not raw['blocked'], **raw})


def replay_fixture_enrichment(root, stage, sha):
    policy_file = root / stage / 'sbom-distributor.json'
    raw = requirements.read_json(root / stage / 'sbom.raw.cyclonedx.json')
    enriched, detail = ENRICH.enrich(raw, requirements.read_json(policy_file), root / stage / 'publishers', sha)
    write_json(root / stage / 'sbom.cyclonedx.json', enriched)
    write_json(root / stage / 'sbom-enrichment.json', {**detail, 'version': 1, 'role': 'release-distributor', 'subject_sha256': sha, 'raw_sbom_sha256': requirements.digest(root / stage / 'sbom.raw.cyclonedx.json'), 'sbom_sha256': requirements.digest(root / stage / 'sbom.cyclonedx.json'), 'policy_sha256': requirements.digest(policy_file)})


def sbom_fixture(root, stage, subject):
    sha = requirements.digest(root / subject)
    raw = {'bomFormat': 'CycloneDX', 'specVersion': '1.6', 'version': 1, 'metadata': {'component': {'name': 'fixture-root', 'bom-ref': 'fixture-root', 'type': 'file'}}, 'components': [{'name': 'fixture-component', 'bom-ref': 'fixture-component', 'version': '1.0.0', 'purl': 'pkg:npm/fixture-component@1.0.0', 'type': 'library'}]}
    if subject.startswith('sdk/') and subject.endswith('.zip'):
        with zipfile.ZipFile(root / subject) as archive:
            for name in {'go.sum', 'Package.resolved', 'requirements.txt'} & set(archive.namelist()):
                data = archive.read(name)
                put(root, stage + '/publishers/' + name, data)
                if name == 'requirements.txt':
                    raw['components'].append({'name': name, 'bom-ref': 'fixture-requirements-file', 'type': 'file', 'hashes': [{'alg': 'SHA-256', 'content': hashlib.sha256(data).hexdigest()}]})
    write_json(root / stage / 'sbom.raw.cyclonedx.json', raw)
    publisher_bytes = b'{"name":"fixture-component","version":"1.0.0","author":"Synthetic fixture author"}\n'
    publisher_sha = hashlib.sha256(publisher_bytes).hexdigest()
    put(root, stage + '/publishers/' + publisher_sha + '.metadata', publisher_bytes)
    ledger = {'version': 1, 'records': [{'bom_ref': 'fixture-component', 'purl': 'pkg:npm/fixture-component@1.0.0', 'source': 'https://registry.npmjs.org/fixture-component/1.0.0', 'field': 'author.name / author', 'author': 'Synthetic fixture author', 'raw_sha256': publisher_sha, 'file': publisher_sha + '.metadata'}], 'unresolved': []}
    write_json(root / stage / 'publishers/publishers.json', ledger)
    policy_file = put(root, stage + '/sbom-distributor.json', (requirements.SOURCE_ROOT / 'docs/SBOM-DISTRIBUTOR.json').read_bytes())
    replay_fixture_enrichment(root, stage, sha)
    write_json(root / stage / 'sbom-assessment-results.oscal.json', {'assessment-results': {'metadata': {'oscal-version': '1.1.2'}, 'results': [{'findings': [{'title': 'NTIA', 'description': 'Overall verdict for NTIA Minimum Elements: 0 error(s), 0 warning(s), 0 informational finding(s).', 'target': {'target-id': 'ntiaminimum-compliance', 'status': {'state': 'satisfied'}}}]}]}})
    put(root, stage + '/sbom-validation.status', b'0\n')
    put(root, stage + '/sbom-validation.log', b'fixture validation log\n')
    put(root, stage + '/subject.sha256', (sha + '\n').encode())
    put(root, stage + '/sbom-producer.txt', b'Syft 1.51.0 fixture\n')
    fixture_receipt(root, stage, 'sbom')


def merge_fixture_scan(root, stage):
    write_json(root / stage / 'trivy-results.json', MERGE.replay_merge(root / stage))


def scan_fixture(root, stage, subject, image=None):
    artifact = image is None
    digest_file, source = ('artifact.sha256', '/input/artifact') if artifact else ('image.sha256', '/input/image.tar')
    put(root, stage + '/' + digest_file, (requirements.digest(root / subject) + '  ' + source + '\n').encode())
    put(root, stage + '/trivy.status', b'0\n')
    put(root, stage + '/trivy.log', b'fixture scan log\n')
    if artifact:
        put(root, stage + '/extraction.status', b'0\n')
    report = {'SchemaVersion': 2, 'ArtifactName': '/input/unpacked' if artifact else source, 'ArtifactType': 'filesystem' if artifact else 'container_image', 'Trivy': {'Version': '0.72.0'},
              'Results': [{'Target': 'fixture-target', 'Packages': [{'Name': 'fixture-package', 'Version': '1.0.0'}], 'Vulnerabilities': []}]}
    if image:
        report['Metadata'] = {'ImageID': image['config_digest'], 'ImageConfig': {'architecture': image['arch'], 'os': image['os']}}
    if artifact:
        for mode in ('fs', 'rootfs'):
            raw = dict(report, Results=report['Results'] if mode == 'fs' else [])
            write_json(root / stage / f'trivy-{mode}-results.json', raw)
            put(root, stage + f'/trivy-{mode}.status', b'0\n')
            put(root, stage + f'/trivy-{mode}.log', b'Synthetic dual-mode scanner control fixture; not actual Trivy output.\n')
        merge_fixture_scan(root, stage)
    else:
        write_json(root / stage / 'trivy-results.json', report)
    fixture_receipt(root, stage, 'artifact-release' if artifact else 'image-release')


def native_fixture(target, language):
    """Minimal package metadata only: no compilation/installability claim."""
    if target.suffix in ('.whl', '.jar', '.nupkg'):
        member, data = {
            'python': ('fixture-1.2.3.dist-info/METADATA', 'Name: fixture\nVersion: 1.2.3\n'),
            'java': ('META-INF/maven/io.ckodex/xoscal-sdk/pom.properties', 'version=1.2.3\n'),
            'csharp': ('fixture.nuspec', '<package><metadata><version>1.2.3</version></metadata></package>'),
        }[language]
        with zipfile.ZipFile(target, 'w') as archive:
            archive.writestr(member, data)
    elif target.suffix == '.pom':
        target.write_text('<project xmlns="http://maven.apache.org/POM/4.0.0"><version>1.2.3</version></project>')
    else:
        member, data = ('fixture-1.2.3/PKG-INFO', b'Name: fixture\nVersion: 1.2.3\n') if language == 'python' else ('package/package.json', b'{"version":"1.2.3"}\n')
        with tarfile.open(target, 'w:gz') as archive:
            info = tarfile.TarInfo(member);info.size = len(data)
            archive.addfile(info, io.BytesIO(data))


def frontend_loader_patch(pins):
    original_spec = importlib.util.spec_from_file_location
    def patched_spec(name, path):
        result = original_spec(name, path);loader = result.loader
        class FixturePinsLoader:
            def create_module(self, spec):
                return loader.create_module(spec)
            def exec_module(self, module):
                loader.exec_module(module)
                if name == 'frontend_inventory':
                    module.__dict__.update(pins)
        result.loader = FixturePinsLoader()
        return result
    # Only isolated tests patch producer pins. Production has no test mode.
    return mock.patch.object(importlib.util, 'spec_from_file_location', side_effect=patched_spec)


def prepare_frontend_fixture(root, pins):
    stage = root / 'evidence/frontend'
    with mock.patch.dict(vars(FRONTEND), pins):
        for name, data in FRONTEND.rebuilt(stage).items():
            (stage / name).write_bytes(data)
    write_json(stage / 'trivy-results.json', {'SchemaVersion': 2, 'ArtifactType': 'cyclonedx', 'Trivy': {'Version': '0.72.0'}, 'Results': [{'Target': 'synthetic browser fixture', 'Packages': [{'Name': 'vue', 'Version': '3.5.43'}]}]})
    fixture_receipt(root, 'evidence/frontend', 'frontend-release')
    for name in POLICY['frontend_assets']:
        shutil.copyfile(stage / name, root / name)
    assessed = root / 'evidence/frontend-sbom'
    assessed.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(stage / 'sbom.cyclonedx.json', assessed / 'sbom.cyclonedx.json')
    write_json(assessed / 'sbom-assessment-results.oscal.json', {'assessment-results': {'metadata': {'oscal-version': '1.1.2'}, 'results': [{'findings': [{'title': 'NTIA synthetic control fixture', 'description': 'Overall verdict for NTIA Minimum Elements: 0 error(s), 0 warning(s), 0 informational finding(s).', 'target': {'target-id': 'ntiaminimum-compliance', 'status': {'state': 'satisfied'}}}]}]}})
    put(root, 'evidence/frontend-sbom/sbom-validation.status', b'0\n')
    put(root, 'evidence/frontend-sbom/sbom-validation.log', b'Synthetic fixture only; no actual validator claim.\n')
    fixture_receipt(root, 'evidence/frontend-sbom', 'sbom')


def make_candidate(root):
    for source, target in ((requirements.SOURCE_ROOT / 'docs/release-policy.json', 'release-policy.json'), (EXCEPTIONS, 'analysis-exceptions.json'), (requirements.SOURCE_ROOT / 'data/frameworks/manifest.yaml', 'framework-manifest.yaml')):
        put(root, 'evidence/policy/' + target, source.read_bytes())
    put(root, 'sdk/sdk-version.txt', b'1.2.3\n')
    put(root, 'sdk/smoke/go-modules.txt', ('\n'.join('go.opentelemetry.io/otel' + suffix + ' v1.45.0' for suffix in ('', '/metric', '/trace', '/sdk', '/sdk/metric')) + '\n').encode())
    for language in SDK.LANGUAGES:
        native = []
        for n, pattern in enumerate(SDK.NATIVE[language]):
            name = 'fixture-' + str(n) + pattern.removeprefix('*')
            target = put(root, 'sdk/packages/' + language + '/' + name, ('native fixture ' + language + str(n)).encode())
            native_fixture(target, language)
            native.append(target)
        archive = root / 'sdk' / (language + '.zip')
        archive.parent.mkdir(parents=True, exist_ok=True)
        with zipfile.ZipFile(archive, 'w') as stream:
            stream.writestr(SDK.METADATA[language], 'fixture package metadata')
            stream.writestr('LICENSE', 'fixture license')
            stream.writestr('SDK-VERSION', '1.2.3\n')
            stream.writestr('SDK-PRODUCER.json', json.dumps(SDK.package.producer_metadata(language, '1.2.3'), sort_keys=True) + '\n')
            if language == 'go':
                stream.writestr('go.sum', 'synthetic source checksum input\n')
            if language == 'python':
                stream.writestr('requirements.txt', '')
            locks = {'swift': 'Package.resolved', 'csharp': 'packages.lock.json', 'ts': 'package-lock.json'}
            if language in locks:
                stream.writestr(locks[language], '{}\n')
            for target in native:
                stream.write(target, target.relative_to(root / 'sdk').as_posix())
        put(root, 'sdk/' + language + '.zip.sha256', ('sha256:' + requirements.digest(archive) + '\n').encode())
        subjects = [{'path': (language + '.zip'), 'sha256': requirements.digest(archive)}]
        subjects += [{'path': p.relative_to(root / 'sdk').as_posix(), 'sha256': requirements.digest(p)} for p in native]
        write_json(root / 'sdk/smoke' / (language + '.json'), {'schema_version': 1, 'language': language, 'status': 'passed', 'scope': 'package-install-and-consumer', 'hosted_proof': False, 'subjects': subjects})
        scan_fixture(root, 'evidence/sdk-' + language, 'sdk/' + language + '.zip')
        sbom_fixture(root, 'evidence/sdk-' + language + '-sbom', 'sdk/' + language + '.zip')
    write_json(root / 'sdk/sdk-contract-check.json', SDK.verify(root / 'sdk'))
    for ref in POLICY['framework_ids']:
        target = root / 'frameworks' / ref / 'catalog.json'
        write_json(target, {'catalog': {'uuid': 'fixture-' + ref, 'metadata': {'title': 'Synthetic contract fixture', 'oscal-version': '1.2.3'}}})
        put(root, target.relative_to(root).as_posix() + '.sha256', ('sha256:' + requirements.digest(target) + '\n').encode())
    profile = root / POLICY['profile_path']
    write_json(profile, {'profile': {'uuid': 'fixture-profile', 'metadata': {'title': 'Synthetic contract fixture', 'oscal-version': '1.2.3'}}})
    put(root, POLICY['profile_path'] + '.sha256', ('sha256:' + requirements.digest(profile) + '\n').encode())
    write_json(root / 'openapi.json', {'openapi': '3.0.0', 'paths': {'/fixture': {'get': {}}}})
    put(root, 'scripts/install.sh', b'#!/bin/sh\n# fixture script only\n')
    put(root, 'release/xoscal-server', b'fixture standalone binary\n')
    sbom_fixture(root, 'evidence/binary-sbom', 'release/xoscal-server')
    shutil.copyfile(root / 'evidence/binary-sbom/sbom-assessment-results.oscal.json', root / 'sbom-assessment-results.json')
    for name in POLICY['ci_outputs']:
        put(root, 'evidence/ci/' + name, b'fixture completed gate\n')
    constraint_subjects = {ref + '/catalog.json': 'catalog' for ref in POLICY['framework_ids']}
    constraint_subjects[POLICY['profile_path'].removeprefix('frameworks/')] = 'profile'
    rows = []
    for path, kind in sorted(constraint_subjects.items()):
        rows.append('\t'.join((path, kind, '0', requirements.digest(root / 'frameworks' / path))))
        put(root, 'evidence/ci/constraints/' + path.split('/')[0] + '-' + kind + '.log', ('PASS: /frameworks/' + path + ' — schema + Metaschema constraints valid\n').encode())
    put(root, 'evidence/ci/constraints/results.tsv', ('\n'.join(rows) + '\n').encode())
    put(root, 'evidence/ci/constraints/result.status', b'0\n')
    semantic = requirements.load_module('fixture_oscal_semantic_scope', 'oscal-semantic-scope.py')
    write_json(root / 'evidence/ci/constraints/semantic-scope.json', semantic.evaluate(root / 'frameworks'))
    put(root, 'evidence/ci/constraints.ok', b'oscal-constraints-ok\n')
    put(root, 'evidence/ci/tidy.ok', b'module-tidy-ok\n')
    stage = 'evidence/security'
    put(root, stage + '/govulncheck.json', (json.dumps({'config': {'scan_level': 'symbol'}}) + '\n' + json.dumps({'progress': {'message': 'fixture completed'}}) + '\n').encode())
    for name in ('govulncheck', 'gosec', 'gosec-sarif'):
        put(root, stage + '/' + name + '.status', b'0\n')
        put(root, stage + '/' + name + '.log', b'fixture raw log\n')
    write_json(root / stage / 'gosec-results.json', {'Golang errors': {}, 'Stats': {'files': 1}, 'Issues': []})
    write_json(root / stage / 'gosec-results.sarif', {'version': '2.1.0', 'runs': [{'results': []}]})
    fixture_receipt(root, stage, 'security')
    for platform in POLICY['cli_platforms']:
        os_name, arch = platform['os'], platform['arch']
        suffix = 'x86_64' if arch == 'amd64' else 'arm64'
        path = 'release/xoscal_1.2.3_' + os_name.title() + '_' + suffix + '.tar.gz'
        put(root, path, ('synthetic platform archive ' + os_name + arch).encode())
        stage = 'evidence/archive-' + os_name + '_' + arch
        scan_fixture(root, stage, path)
        sbom_fixture(root, stage + '-sbom', path)
    combined = root / 'release/image.tar'
    write_archive(combined, make_layout())
    raw, report = oci.inspect(combined)
    put(root, 'release/image-index.json', raw)
    put(root, 'release/image-digest.txt', (report['root_digest'] + '\n').encode())
    write_json(root / 'release/image-platforms.json', report)
    for platform in report['platforms']:
        arch = platform['arch']
        path = 'release/image-' + arch + '.tar'
        write_archive(root / path, make_layout((arch,)))
        scan_fixture(root, 'evidence/image-' + arch, path, platform)
        sbom_fixture(root, 'evidence/image-' + arch + '-sbom', path)
    return inventory.create(root, 'v1.2.3', 'a' * 40)


class RequiredOutputTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='release-required-test-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / 'candidate'
        self.root.mkdir()
        pins = make_frontend_fixture(self.root / 'evidence/frontend')
        pin_patch = frontend_loader_patch(pins);pin_patch.start();self.addCleanup(pin_patch.stop)
        make_candidate(self.root)
        prepare_frontend_fixture(self.root, pins)
        self.manifest = self.refreshed()

    def refreshed(self):
        return inventory.create(self.root, 'v1.2.3', 'a' * 40)

    def reject(self, pattern=None, manifest=None):
        with self.assertRaisesRegex(ValueError, pattern or '.'):
            requirements.required_payload(self.root, manifest or self.refreshed())

    def test_complete_synthetic_contract(self):
        result = requirements.required_payload(self.root, self.manifest)
        self.assertEqual(result['status'], 'passed')
        self.assertGreater(result['required_files'], 200)

    def test_each_missing_stage_family_blocks(self):
        for stage in ('sdk/smoke/go.json', 'sdk/sdk-contract-check.json', 'release/image.tar', 'release/xoscal-server', 'evidence/security/govulncheck.json',
                      'evidence/binary-sbom/analysis-gate.json', 'evidence/sdk-swift/trivy-results.json', 'evidence/sdk-python-sbom/subject.sha256',
                      'evidence/archive-darwin_arm64/trivy.status', 'evidence/archive-linux_amd64-sbom/sbom-validation.status', 'evidence/image-arm64/image.sha256',
                      'evidence/image-amd64-sbom/sbom.cyclonedx.json', 'evidence/ci/openapi.ok'):
            with self.subTest(stage=stage):
                path = self.root / stage
                raw = path.read_bytes();path.unlink()
                self.reject('missing required')
                path.write_bytes(raw)

    def test_missing_native_package_blocks(self):
        path = next((self.root / 'sdk/packages/python').glob('*.whl'))
        path.unlink()
        self.reject('native package')

    def test_all_frontend_required_evidence_files_must_exist(self):
        paths = ['evidence/frontend/' + name for name in POLICY['frontend_files']]
        paths += ['evidence/frontend-sbom/' + name for name in POLICY['frontend_sbom_files']]
        paths += POLICY['frontend_assets']
        for name in paths:
            with self.subTest(path=name):
                path = self.root / name;original = path.read_bytes();path.unlink()
                self.reject('missing required')
                path.write_bytes(original)

    def test_frontend_served_assets_must_equal_admitted_evidence(self):
        for name in POLICY['frontend_assets']:
            with self.subTest(asset=name):
                path = self.root / name;original = path.read_bytes();path.write_bytes(original + b'tampered')
                self.reject('served frontend')
                path.write_bytes(original)

    def test_frontend_producer_inputs_cannot_be_tampered(self):
        for name in ('scalar.js', 'scalar.js.map', 'scalar.css', 'scalar.css.map', 'package-lock.json', 'entry.js', 'build.mjs', 'bundle-inputs.tar.gz', 'frontend-inventory.json', 'installed-package-manifests.json', 'landmark-transform.mjs', 'landmark-transform.test.mjs', 'landmark-transforms.json', 'landmark-transform-receipt.json'):
            with self.subTest(input=name):
                path = self.root / 'evidence/frontend' / name;original = path.read_bytes();path.write_bytes(original + b'tampered')
                self.reject()
                path.write_bytes(original)

    def test_frontend_failed_tools_cannot_keep_saved_pass(self):
        for tool in ('npm-ci', 'build', 'trivy'):
            with self.subTest(tool=tool):
                path = self.root / 'evidence/frontend' / (tool + '.status');original = path.read_bytes();path.write_text('1\n')
                self.reject('operational failure')
                path.write_bytes(original)

    def test_frontend_fixed_critical_raw_finding_blocks_saved_pass(self):
        path = self.root / 'evidence/frontend/trivy-results.json'
        report = requirements.read_json(path);report['Results'][0]['Vulnerabilities'] = [{'VulnerabilityID': 'CVE-frontend-fixture', 'Severity': 'CRITICAL', 'FixedVersion': '3.5.44'}];write_json(path, report)
        self.reject('policy rejected raw evidence')

    def test_frontend_scan_must_cover_observed_components(self):
        path = self.root / 'evidence/frontend/trivy-results.json'
        report = requirements.read_json(path);report['Results'][0]['Packages'] = [];write_json(path, report)
        self.reject('omitted observed source packages')

    def test_frontend_saved_receipt_cannot_forge_subject(self):
        path = self.root / 'evidence/frontend/analysis-gate.json'
        receipt = requirements.read_json(path);receipt['artifact_sha256'] = 'b' * 64;write_json(path, receipt)
        self.reject('saved admission differs')

    def test_frontend_sbom_must_equal_replayed_runtime_inventory(self):
        path = self.root / 'evidence/frontend-sbom/sbom.cyclonedx.json'
        sbom = requirements.read_json(path);sbom['components'][0]['version'] = '0.0.0-fake';write_json(path, sbom)
        self.reject('frontend assessed SBOM differs')

    def test_frontend_sbom_cannot_claim_syft_subject(self):
        put(self.root, 'evidence/frontend-sbom/sbom-producer.txt', b'Syft 1.51.0 fabricated\n')
        self.reject('actual esbuild producer')

    def test_frontend_sbom_raw_violation_cannot_keep_saved_pass(self):
        path = self.root / 'evidence/frontend-sbom/sbom-assessment-results.oscal.json'
        report = requirements.read_json(path);result = report['assessment-results']['results'][0]
        result['observations'] = [{'uuid': 'fixture-error', 'props': [{'name': 'severity', 'value': 'error'}]}]
        result['findings'][0]['description'] = 'Overall verdict for NTIA Minimum Elements: 1 error(s), 0 warning(s), 0 informational finding(s).'
        result['findings'][0]['target']['status']['state'] = 'not-satisfied'
        result['findings'].append({'title': 'Synthetic missing metadata', 'description': 'Synthetic actionable violation', 'target': {'target-id': 'fixture-error-target', 'status': {'state': 'not-satisfied'}}, 'related-observations': [{'observation-uuid': 'fixture-error'}]})
        write_json(path, report)
        self.reject('policy rejected raw evidence')

    def test_semantic_scope_receipt_must_exist(self):
        (self.root / 'evidence/ci/constraints/semantic-scope.json').unlink()
        self.reject('missing required')

    def test_semantic_scope_identity_must_match_pinned_model(self):
        path = self.root / 'evidence/ci/constraints/semantic-scope.json'
        receipt = requirements.read_json(path);receipt['producer']['semantic_model_version'] = '1.2.3';write_json(path, receipt)
        self.reject('semantic tool scope receipt differs')

    def test_semantic_scope_subject_digest_cannot_be_forged(self):
        path = self.root / 'evidence/ci/constraints/semantic-scope.json'
        receipt = requirements.read_json(path);receipt['subjects'][0]['sha256'] = 'b' * 64;write_json(path, receipt)
        self.reject('semantic tool scope receipt differs')

    def test_semantic_scope_unsupported_feature_blocks_forged_receipt(self):
        path = self.root / POLICY['profile_path']
        profile = requirements.read_json(path);profile['profile']['hashes'] = [];write_json(path, profile)
        put(self.root, POLICY['profile_path'] + '.sha256', ('sha256:' + requirements.digest(path) + '\n').encode())
        rows = self.constraint_rows()
        for index, row in enumerate(rows):
            columns = row.split('\t')
            if columns[1] == 'profile':
                columns[3] = requirements.digest(path);rows[index] = '\t'.join(columns)
        self.save_constraint_rows(rows)
        self.reject('semantic tool scope rejected actual framework bytes')

    def test_root_module_tidy_marker_missing_blocks(self):
        (self.root / 'evidence/ci/tidy.ok').unlink()
        self.reject('missing required')

    def test_root_module_tidy_marker_must_be_exact(self):
        put(self.root, 'evidence/ci/tidy.ok', b'module-tidy-ok\nmodule drift ignored\n')
        self.reject('invalid root module tidy admission marker')

    def constraint_rows(self):
        return (self.root / 'evidence/ci/constraints/results.tsv').read_text().splitlines()

    def save_constraint_rows(self, rows):
        put(self.root, 'evidence/ci/constraints/results.tsv', ('\n'.join(rows) + '\n').encode())

    def test_constraint_profile_result_missing_blocks(self):
        self.save_constraint_rows([row for row in self.constraint_rows() if '\tprofile\t' not in row])
        self.reject('incomplete result count')

    def test_constraint_nonzero_result_blocks_despite_passed_marker(self):
        rows = self.constraint_rows();columns = rows[0].split('\t');columns[2] = '3';rows[0] = '\t'.join(columns)
        self.save_constraint_rows(rows)
        self.reject('constraints subject failed')

    def test_constraint_duplicate_subject_blocks(self):
        rows = self.constraint_rows();rows[-1] = rows[0];self.save_constraint_rows(rows)
        self.reject('duplicate or unexpected')

    def test_constraint_extra_result_blocks(self):
        rows = self.constraint_rows();rows.append('unknown/catalog.json\tcatalog\t0\t' + 'a' * 64);self.save_constraint_rows(rows)
        self.reject('incomplete result count')

    def test_constraint_aggregate_failure_blocks(self):
        put(self.root, 'evidence/ci/constraints/result.status', b'1\n')
        self.reject('constraints aggregate failed')

    def test_constraint_marker_must_be_exact(self):
        put(self.root, 'evidence/ci/constraints.ok', b'oscal-constraints-ok\nignored failure\n')
        self.reject('invalid OSCAL constraints admission marker')

    def test_constraint_subject_digest_binds_exact_profile_bytes(self):
        profile = self.root / POLICY['profile_path']
        document = requirements.read_json(profile);document['profile']['metadata']['title'] = 'mutated after validation';write_json(profile, document)
        put(self.root, POLICY['profile_path'] + '.sha256', ('sha256:' + requirements.digest(profile) + '\n').encode())
        self.reject('constraints subject digest differs')

    def test_constraint_raw_profile_log_must_exist(self):
        path = self.root / 'evidence/ci/constraints/cccs-medium-cloud-pbmm-profile.log';path.unlink()
        self.reject('missing required')

    def test_constraint_raw_log_must_indicate_exact_subject_pass(self):
        put(self.root, 'evidence/ci/constraints/cccs-medium-cloud-pbmm-profile.log', b'FAIL: unrelated\n')
        self.reject('raw log lacks successful subject')

    def test_constraint_uninventoried_log_blocks(self):
        manifest = self.refreshed()
        path = 'evidence/ci/constraints/cccs-medium-cloud-pbmm-profile.log'
        manifest['artifacts'] = [item for item in manifest['artifacts'] if item['path'] != path]
        self.reject('missing required', manifest)

    def test_constraint_legacy_unbound_result_format_blocks(self):
        rows = ['\t'.join(row.split('\t')[:3]) for row in self.constraint_rows()];self.save_constraint_rows(rows)
        self.reject('malformed OSCAL constraints result row')

    def test_publisher_auxiliary_exact_sdk_binding(self):
        for language, filename in (('go', 'go.sum'), ('swift', 'Package.resolved'), ('python', 'requirements.txt')):
            with self.subTest(language=language):
                path = self.root / ('evidence/sdk-' + language + '-sbom/publishers/' + filename)
                original = path.read_bytes();path.write_bytes(original + b'tampered')
                self.reject('publisher auxiliary differs from original SDK ZIP')
                path.write_bytes(original)

    def test_publisher_auxiliary_missing_cannot_bypass(self):
        for language, filename in (('go', 'go.sum'), ('swift', 'Package.resolved'), ('python', 'requirements.txt')):
            with self.subTest(language=language):
                path = self.root / ('evidence/sdk-' + language + '-sbom/publishers/' + filename)
                original = path.read_bytes();path.unlink()
                self.reject('publisher auxiliary differs from original SDK ZIP')
                path.write_bytes(original)

    def test_publisher_auxiliary_requires_original_sdk_subject(self):
        put(self.root, 'evidence/binary-sbom/publishers/go.sum', b'not from SDK ZIP')
        self.reject('publisher raw file set differs')

    def test_missing_sbom_enrichment_family_blocks(self):
        stage = 'evidence/sdk-java-sbom'
        for name in ('sbom.raw.cyclonedx.json', 'sbom-distributor.json', 'sbom-enrichment.json', 'publishers/publishers.json'):
            with self.subTest(name=name):
                path = self.root / stage / name
                original = path.read_bytes();path.unlink()
                self.reject('missing required')
                path.write_bytes(original)

    def test_publisher_raw_metadata_must_be_in_inventory(self):
        manifest = self.refreshed()
        path = next((self.root / 'evidence/sdk-python-sbom/publishers').glob('*.metadata')).relative_to(self.root).as_posix()
        manifest['artifacts'] = [record for record in manifest['artifacts'] if record['path'] != path]
        self.reject('publisher raw metadata missing from signed inventory', manifest)

    def test_sdk_publisher_metadata_must_match_original_zip(self):
        stage = self.root / 'evidence/sdk-python-sbom'
        with zipfile.ZipFile(self.root / 'sdk/python.zip') as archive:
            producer = json.loads(archive.read('SDK-PRODUCER.json'))
        producer['unrelated'] = 'not present in actual ZIP'
        data = (json.dumps(producer) + '\n').encode();sha = hashlib.sha256(data).hexdigest()
        for file in (stage / 'publishers').glob('*.metadata'):
            file.unlink()
        put(self.root, 'evidence/sdk-python-sbom/publishers/' + sha + '.metadata', data)
        write_json(stage / 'publishers/publishers.json', {'version': 1, 'unresolved': [], 'records': [{'bom_ref': 'fixture-component', 'purl': 'pkg:npm/fixture-component@1.0.0', 'source': 'artifact:SDK-PRODUCER.json', 'field': 'entity (exact SDK package producer)', 'supplier': producer['entity'], 'version': producer['package']['version'], 'raw_sha256': sha, 'file': sha + '.metadata'}]})
        self.reject('publisher SDK producer differs from original ZIP bytes')

    def test_raw_sbom_tamper_blocks_enrichment_replay(self):
        path = self.root / 'evidence/binary-sbom/sbom.raw.cyclonedx.json'
        raw = requirements.read_json(path);raw['components'][0]['name'] = 'tampered-fixture';write_json(path, raw)
        self.reject('enrichment digest mismatch')

    def test_missing_publisher_metadata_blocks(self):
        next((self.root / 'evidence/sdk-python-sbom/publishers').glob('*.metadata')).unlink()
        self.reject('missing publisher raw metadata')

    def test_publisher_metadata_tamper_blocks(self):
        path = next((self.root / 'evidence/sdk-python-sbom/publishers').glob('*.metadata'))
        path.write_bytes(path.read_bytes() + b'tamper')
        self.reject('publisher raw metadata digest mismatch')

    def test_forged_publisher_author_blocks(self):
        path = self.root / 'evidence/sdk-python-sbom/publishers/publishers.json'
        ledger = requirements.read_json(path);ledger['records'][0]['author'] = 'Invented author';write_json(path, ledger)
        self.reject('publisher field mismatch')

    def test_removing_publisher_proof_cannot_keep_enriched_author(self):
        path = self.root / 'evidence/sdk-python-sbom/publishers/publishers.json'
        ledger = requirements.read_json(path);ledger['records'] = [];write_json(path, ledger)
        for item in path.parent.glob('*.metadata'):
            item.unlink()
        self.reject('enrichment does not reproduce')

    def test_enrichment_receipt_subject_tamper_blocks(self):
        path = self.root / 'evidence/sdk-python-sbom/sbom-enrichment.json'
        receipt = requirements.read_json(path);receipt['subject_sha256'] = 'b' * 64;write_json(path, receipt)
        self.reject('enrichment digest mismatch.*subject_sha256')

    def test_wrong_distributor_identity_blocks(self):
        path = self.root / 'evidence/sdk-python-sbom/sbom-distributor.json'
        policy = requirements.read_json(path);policy['entity']['name'] = 'unapproved identity';write_json(path, policy)
        self.reject('SBOM distributor differs')

    def test_missing_staged_policy_blocks(self):
        for name in ('release-policy.json', 'analysis-exceptions.json', 'framework-manifest.yaml'):
            with self.subTest(name=name):
                path = self.root / 'evidence/policy' / name
                raw = path.read_bytes();path.unlink()
                self.reject('missing required staged release policy')
                path.write_bytes(raw)

    def test_staged_framework_policy_digest_mismatch_blocks(self):
        put(self.root, 'evidence/policy/framework-manifest.yaml', b'different framework source\n')
        self.reject('framework manifest changed')

    def test_historical_release_uses_staged_framework_policy(self):
        path = self.root / 'evidence/policy/framework-manifest.yaml'
        path.write_bytes(path.read_bytes() + b'\n# historical fixture source bytes\n')
        policy_path = self.root / 'evidence/policy/release-policy.json'
        policy = requirements.read_json(policy_path)
        policy['framework_manifest_sha256'] = requirements.digest(path)
        write_json(policy_path, policy)
        self.assertEqual(requirements.required_payload(self.root, self.refreshed())['status'], 'passed')

    def test_untested_extra_native_package_blocks(self):
        put(self.root, 'sdk/packages/swift/unvalidated-package.zip', b'not consumer tested')
        self.reject('native SDK package set differs')

    def test_symlink_directory_artifact_path_blocks(self):
        packages = self.root / 'sdk/packages'
        external = self.root.parent / 'moved-packages'
        packages.rename(external);packages.symlink_to(external, target_is_directory=True)
        self.reject('symlink artifact path', self.manifest)

    def test_forged_consumer_subject_blocks(self):
        path = self.root / 'sdk/smoke/java.json'
        doc = requirements.read_json(path);doc['subjects'][0]['sha256'] = 'b' * 64;write_json(path, doc)
        self.reject('consumer subject tampered')

    def test_forged_sdk_aggregate_receipt_blocks(self):
        path = self.root / 'sdk/sdk-contract-check.json'
        doc = requirements.read_json(path);doc['subjects'] = [];write_json(path, doc)
        self.reject('aggregate receipt')

    def test_invalid_sidecar_content_blocks(self):
        put(self.root, 'frameworks/nist-csf-2.0/catalog.json.sha256', b'not a digest\n')
        self.reject('sidecar differs')

    def test_fake_catalog_id_with_same_count_blocks(self):
        (self.root / 'frameworks/nist-csf-2.0').rename(self.root / 'frameworks/fake-framework')
        self.reject('missing required')

    def test_duplicate_platform_records_block(self):
        manifest = self.refreshed()
        for item in manifest['artifacts']:
            if item['kind'] == 'cli-archive':
                item['platform'] = {'os': 'linux', 'arch': 'amd64'}
        self.reject('platform', manifest)

    def test_wrong_producer_blocks(self):
        self.manifest['artifacts'][0]['producer'] = 'wrong-producer'
        self.reject('ownership differs', self.manifest)

    def test_retagged_archive_blocks(self):
        path = next((self.root / 'release').glob('*Darwin_arm64.tar.gz'))
        path.rename(path.with_name(path.name.replace('1.2.3', '1.2.4')))
        self.reject('tag/platform mismatch')

    def test_saved_passed_boolean_cannot_hide_raw_scan_failure(self):
        path = self.root / 'evidence/sdk-go/trivy-fs-results.json'
        report = requirements.read_json(path)
        report['Results'][0]['Vulnerabilities'] = [{'Severity': 'CRITICAL', 'FixedVersion': '2', 'VulnerabilityID': 'fixture-blocking-advisory'}]
        write_json(path, report)
        merge_fixture_scan(self.root, 'evidence/sdk-go')
        self.reject('policy rejected raw')

    def test_operational_scan_error_blocks(self):
        put(self.root, 'evidence/image-arm64/trivy.status', b'2\n')
        self.reject('operational failure')

    def test_saved_receipt_cannot_hide_empty_raw_inventory(self):
        path = self.root / 'evidence/sdk-swift/trivy-fs-results.json'
        report = requirements.read_json(path);report['Results'][0]['Packages'] = [];write_json(path, report)
        merge_fixture_scan(self.root, 'evidence/sdk-swift')
        self.reject('no identifiable')

    def test_both_artifact_modes_raw_files_are_mandatory(self):
        for stage in ('evidence/sdk-go', 'evidence/archive-linux_amd64'):
            for mode in ('fs', 'rootfs'):
                for suffix in ('-results.json', '.status', '.log'):
                    with self.subTest(stage=stage, mode=mode, suffix=suffix):
                        path = self.root / stage / ('trivy-' + mode + suffix)
                        original = path.read_bytes();path.unlink()
                        self.reject('missing required')
                        path.write_bytes(original)

    def test_raw_artifact_report_tamper_without_new_merge_blocks(self):
        path = self.root / 'evidence/sdk-go/trivy-rootfs-results.json'
        report = requirements.read_json(path)
        report['Results'] = [{'Target': 'synthetic observed executable', 'Packages': [{'Name': 'module', 'Version': '1.0.0'}]}]
        write_json(path, report)
        self.reject('combined report differs from raw scanner evidence')

    def test_combined_artifact_report_tamper_blocks(self):
        path = self.root / 'evidence/archive-linux_amd64/trivy-results.json'
        report = requirements.read_json(path)
        report['XoscalRawReports']['rootfs'] = 'a' * 64
        write_json(path, report)
        self.reject('combined report differs from raw scanner evidence')

    def test_rootfs_finding_cannot_hide_behind_successful_fs_scan(self):
        path = self.root / 'evidence/archive-linux_amd64/trivy-rootfs-results.json'
        report = requirements.read_json(path)
        report['Results'] = [{'Target': 'synthetic observed executable', 'Packages': [{'Name': 'module', 'Version': '1.0.0'}],
                              'Vulnerabilities': [{'Severity': 'CRITICAL', 'FixedVersion': '2', 'VulnerabilityID': 'synthetic-rootfs-blocker'}]}]
        write_json(path, report)
        merge_fixture_scan(self.root, 'evidence/archive-linux_amd64')
        self.reject('policy rejected raw')

    def test_artifact_mode_operational_failure_blocks(self):
        for mode in ('fs', 'rootfs'):
            with self.subTest(mode=mode):
                path = self.root / f'evidence/sdk-go/trivy-{mode}.status'
                path.write_text('2\n')
                self.reject('operational failure')
                path.write_text('0\n')

    def test_scan_subject_digest_mismatch_blocks(self):
        path = self.root / 'evidence/sdk-go/artifact.sha256'
        put(self.root, path.relative_to(self.root).as_posix(), ('b' * 64 + '  /input/artifact\n').encode())
        fixture_receipt(self.root, 'evidence/sdk-go', 'artifact-release')
        self.reject('scan subject differs')

    def test_sbom_subject_digest_mismatch_blocks(self):
        put(self.root, 'evidence/image-amd64-sbom/subject.sha256', ('b' * 64 + '\n').encode())
        replay_fixture_enrichment(self.root, 'evidence/image-amd64-sbom', 'b' * 64)
        fixture_receipt(self.root, 'evidence/image-amd64-sbom', 'sbom')
        self.reject('SBOM subject differs')

    def test_image_scan_config_mismatch_blocks(self):
        path = self.root / 'evidence/image-arm64/trivy-results.json'
        report = requirements.read_json(path);report['Metadata']['ImageID'] = 'sha256:' + 'b' * 64;write_json(path, report)
        fixture_receipt(self.root, 'evidence/image-arm64', 'image-release')
        self.reject('scan config differs')

    def test_displayed_sbom_raw_mismatch_blocks(self):
        put(self.root, 'sbom-assessment-results.json', b'{}\n')
        self.reject('displayed SBOM')

    def test_oci_receipt_cannot_forge_graph(self):
        path = self.root / 'release/image-platforms.json'
        report = requirements.read_json(path);report['platforms'] = [];write_json(path, report)
        self.reject('OCI graph receipt')

    def test_final_requires_presentation_admission(self):
        manifest = self.refreshed();manifest['scope'] = 'final'
        self.reject('missing required.*presentation.ok', manifest)
        put(self.root, 'evidence/presentation.ok', b'presentation-analysis-ok\n')
        write_json(self.root / 'evidence/release-source.json', {'schema_version': 1, 'release_tag': 'v1.2.3', 'source_revision': 'a' * 40, 'version': 'v1.2.3', 'scope': 'clean-tagged-source', 'status': 'passed'})
        make_browser_fixture(self.root)
        manifest = inventory.create(self.root, 'v1.2.3', 'a' * 40, final=True)
        self.assertEqual(requirements.required_payload(self.root, manifest)['status'], 'passed')

    def test_final_source_identity_must_match_manifest(self):
        put(self.root, 'evidence/presentation.ok', b'presentation-analysis-ok\n')
        make_browser_fixture(self.root)
        for field, wrong in (('source_revision', 'b' * 40), ('release_tag', 'v1.2.4'), ('version', 'v1.2.4'), ('scope', 'preview'), ('status', 'incomplete')):
            with self.subTest(field=field):
                doc = {'schema_version': 1, 'release_tag': 'v1.2.3', 'source_revision': 'a' * 40, 'version': 'v1.2.3', 'scope': 'clean-tagged-source', 'status': 'passed'}
                doc[field] = wrong
                write_json(self.root / 'evidence/release-source.json', doc)
                manifest = inventory.create(self.root, 'v1.2.3', 'a' * 40, final=True)
                self.reject('production source identity differs', manifest)

    def test_sdk_version_must_match_release(self):
        manifest = self.refreshed();manifest['release_tag'] = 'v1.2.4'
        for item in manifest['artifacts']:
            if item['kind'] == 'cli-archive':
                source = self.root / item['path']
                source.rename(source.with_name(source.name.replace('1.2.3', '1.2.4')))
        manifest = inventory.create(self.root, 'v1.2.4', 'a' * 40)
        self.reject('SDK version differs', manifest)

    def test_empty_openapi_blocks(self):
        write_json(self.root / 'openapi.json', {'openapi': '3.0.0', 'paths': {}})
        self.reject('empty OpenAPI')


if __name__ == '__main__':
    print('Synthetic admission fixtures only; no real compiler/scanner/hosted proof claimed.', flush=True)
    unittest.main(verbosity=2)
