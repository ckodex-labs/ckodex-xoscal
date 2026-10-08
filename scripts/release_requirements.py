#!/usr/bin/env python3
"""Versioned required-output and exact-subject admission for release candidates."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path, PurePosixPath
import re
import zipfile

SOURCE_ROOT = Path(__file__).resolve().parents[1]
LANGUAGES = ('go', 'python', 'java', 'csharp', 'ts', 'swift')
PLATFORMS = {('linux', 'amd64'), ('linux', 'arm64'), ('darwin', 'amd64'), ('darwin', 'arm64')}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        require(key not in value, 'duplicate JSON property: ' + key)
        value[key] = item
    return value


def read_json(path):
    value = json.loads(Path(path).read_text(), object_pairs_hook=unique_object)
    require(isinstance(value, dict), 'expected JSON object: ' + str(path))
    return value


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(block)
    return value.hexdigest()


def load_module(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def replay_gate(root, evidence_path, kind, gate, exceptions, subject=None):
    evidence = root / evidence_path
    if kind == 'security':
        computed = gate._security_gate(evidence)
    elif kind == 'sbom':
        declaration = read_json(evidence / 'sbom-distributor.json')
        expected_distributor = read_json(root / 'evidence/policy/release-policy.json').get('sbom_distributor')
        require(expected_distributor is not None and {key: declaration.get(key) for key in ('version', 'role', 'entity')} == expected_distributor, 'SBOM distributor differs from signed release policy')
        ledger = read_json(evidence / 'publishers/publishers.json')
        require(type(ledger.get('version')) is int and ledger['version'] == 1 and isinstance(ledger.get('records'), list) and isinstance(ledger.get('unresolved'), list), 'invalid publisher ledger: ' + evidence_path)
        expected_files = {'publishers.json'}
        auxiliary = {'go.mod', 'go.sum', 'Package.resolved', 'requirements.txt'}
        if subject is not None and re.fullmatch(r'sdk/(go|python|java|csharp|ts|swift)\.zip', subject):
            with zipfile.ZipFile(root / subject) as archive:
                for name in auxiliary & set(archive.namelist()):
                    retained = evidence / 'publishers' / name
                    require(retained.is_file() and not retained.is_symlink() and retained.read_bytes() == archive.read(name), 'publisher auxiliary differs from original SDK ZIP: ' + name)
                    expected_files.add(name)
        for record in ledger['records']:
            require(isinstance(record, dict) and isinstance(record.get('raw_sha256'), str) and re.fullmatch(r'[0-9a-f]{64}', record['raw_sha256']), 'invalid publisher record digest')
            filename = record['raw_sha256'] + '.metadata'
            require(record.get('file') == filename, 'unsafe publisher record file')
            raw = evidence / 'publishers' / filename
            require(raw.is_file() and not raw.is_symlink(), 'missing publisher raw metadata: ' + filename)
            require(digest(raw) == record['raw_sha256'], 'publisher raw metadata digest mismatch')
            if record.get('source') == 'artifact:SDK-PRODUCER.json':
                require(subject is not None and re.fullmatch(r'sdk/(go|python|java|csharp|ts|swift)\.zip', subject), 'SDK producer evidence lacks original source ZIP')
                with zipfile.ZipFile(root / subject) as archive:
                    require(raw.read_bytes() == archive.read('SDK-PRODUCER.json'), 'publisher SDK producer differs from original ZIP bytes')
            expected_files.add(filename)
        require({p.name for p in (evidence / 'publishers').iterdir()} == expected_files, 'publisher raw file set differs from retained ledger')
        computed = gate.sbom_gate(evidence, exceptions)
    elif kind == 'frontend-sbom':
        require(not (evidence / 'subject.sha256').exists() and not (evidence / 'sbom-producer.txt').exists(), 'frontend SBOM must identify its actual esbuild producer')
        require((evidence / 'sbom.cyclonedx.json').read_bytes() == (root / 'evidence/frontend/sbom.cyclonedx.json').read_bytes(), 'frontend assessed SBOM differs from replayed runtime inventory')
        computed = gate.sbom_gate(evidence, exceptions)
    elif kind == 'frontend-release':
        computed = gate.frontend_gate(evidence)
    elif kind == 'image-release':
        computed = gate.image_gate(evidence)
    elif kind == 'artifact-release':
        computed = gate.artifact_gate(evidence)
    else:
        raise ValueError('unsupported admission stage')
    require(computed.get('blocked') == [], 'analysis policy rejected raw evidence: ' + evidence_path)
    saved = read_json(evidence / 'analysis-gate.json')
    require(saved.get('policy_version') == 1 and saved.get('kind') == ('sbom' if kind == 'frontend-sbom' else kind) and saved.get('passed') is True and 'error' not in saved,
            'invalid saved admission receipt: ' + evidence_path)
    require(all(saved.get(key) == value for key, value in computed.items()), 'saved admission differs from replayed policy: ' + evidence_path)
    if subject is not None:
        actual = digest(root / subject)
        if kind == 'frontend-release':
            require(subject == 'scalar.js' and (evidence / 'scalar.js').read_bytes() == (root / subject).read_bytes(), 'served frontend JavaScript differs from admitted bytes')
            require((evidence / 'scalar.js.map').read_bytes() == (root / 'scalar.js.map').read_bytes(), 'served frontend source map differs from admitted bytes')
            for asset in ('scalar.css', 'scalar.css.map', 'THIRD-PARTY-NOTICES.txt'):
                require((evidence / asset).read_bytes() == (root / asset).read_bytes(), 'served frontend asset differs from admitted bytes: ' + asset)
            require((evidence / 'frontend.sha256').read_text() == actual + '  /input/scalar.js\n', 'frontend digest record differs from served artifact')
            require(saved.get('artifact_sha256') == actual and saved.get('source_map_sha256') == digest(root / 'scalar.js.map'), 'frontend admission lacks exact served subjects')
        elif kind == 'sbom':
            declared = (evidence / 'subject.sha256').read_text()
            require(re.fullmatch(r'[0-9a-f]{64}\n?', declared) and declared.strip() == actual, 'SBOM subject differs from released artifact: ' + subject)
            require(saved.get('subject_sha256') == actual, 'SBOM admission lacks exact subject: ' + subject)
        else:
            filename, source = ('image.sha256', '/input/image.tar') if kind == 'image-release' else ('artifact.sha256', '/input/artifact')
            declared = (evidence / filename).read_text()
            require(declared.strip() == actual + '  ' + source, 'scan subject differs from released artifact: ' + subject)
            require(saved.get('artifact_sha256') == actual, 'scan admission lacks exact subject: ' + subject)
    return computed


def replay_constraints(root, records, framework_ids, profile_path):
    expected = {ref + '/catalog.json': 'catalog' for ref in framework_ids}
    expected[profile_path.removeprefix('frameworks/')] = 'profile'
    directory = root / 'evidence/ci/constraints'
    require((root / 'evidence/ci/constraints.ok').read_text() == 'oscal-constraints-ok\n', 'invalid OSCAL constraints admission marker')
    require((directory / 'result.status').read_text() == '0\n', 'OSCAL constraints aggregate failed')
    expected_files = {'results.tsv', 'result.status', 'semantic-scope.json'} | {path.split('/')[0] + '-' + kind + '.log' for path, kind in expected.items()}
    require({p.name for p in directory.iterdir()} == expected_files, 'OSCAL constraints raw file set differs from policy')
    rows = (directory / 'results.tsv').read_text().splitlines()
    require(len(rows) == len(expected), 'OSCAL constraints incomplete result count')
    seen = set()
    for row in rows:
        columns = row.split('\t')
        require(len(columns) == 4, 'malformed OSCAL constraints result row')
        path, kind, status, sha = columns
        require(path in expected and path not in seen and kind == expected[path], 'duplicate or unexpected OSCAL constraints subject')
        require(status == '0', 'OSCAL constraints subject failed: ' + path)
        actual_path = 'frameworks/' + path
        require(re.fullmatch(r'[0-9a-f]{64}', sha) and sha == digest(root / actual_path) and records[actual_path]['digest'] == 'sha256:' + sha, 'OSCAL constraints subject digest differs: ' + path)
        log_path = 'evidence/ci/constraints/' + path.split('/')[0] + '-' + kind + '.log'
        require(log_path in records, 'OSCAL constraints log missing from inventory')
        log = (root / log_path).read_text()
        require('PASS: /frameworks/' + path + ' — schema + Metaschema constraints valid\n' in log and 'FAIL:' not in log, 'OSCAL constraints raw log lacks successful subject: ' + path)
        seen.add(path)
    require(seen == set(expected), 'OSCAL constraints missing catalog/profile subject')
    helper = load_module('release_oscal_semantic_scope', 'oscal-semantic-scope.py')
    semantic = helper.evaluate(root / 'frameworks')
    require(semantic.get('status') == 'passed' and semantic.get('findings') == [], 'OSCAL semantic tool scope rejected actual framework bytes')
    require(read_json(directory / 'semantic-scope.json') == semantic, 'OSCAL semantic tool scope receipt differs from exact subjects')


def required_payload(root, manifest, policy_path=None, exceptions_path=None):
    """Require complete producer outputs, replay policy, and bind their exact subjects."""
    root = Path(root)
    staged_policy = root / 'evidence/policy/release-policy.json'
    staged_exceptions = root / 'evidence/policy/analysis-exceptions.json'
    staged_frameworks = root / 'evidence/policy/framework-manifest.yaml'
    require(all(path.is_file() and not path.is_symlink() for path in (staged_policy, staged_exceptions, staged_frameworks)), 'missing required staged release policy')
    # Default admission uses only integrity-covered policy bytes. Overrides are
    # exposed for controlled tests; production callers leave both unset.
    policy = read_json(policy_path or staged_policy)
    exceptions = Path(exceptions_path or staged_exceptions)
    require(policy.get('policy_version') == 1, 'unsupported required-output policy')
    require(policy.get('sdk_languages') == list(LANGUAGES), 'incomplete SDK policy')
    ids = policy.get('framework_ids')
    require(isinstance(ids, list) and len(ids) == len(set(ids)) == 35 and all(isinstance(i, str) and re.fullmatch(r'[A-Za-z0-9_.-]+', i) for i in ids), 'invalid exact framework policy')
    require(policy.get('framework_manifest_sha256') == digest(staged_frameworks), 'framework manifest changed without updating release policy')
    require({(p['os'], p['arch']) for p in policy.get('cli_platforms', [])} == PLATFORMS, 'incomplete platform policy')
    entries = manifest.get('artifacts')
    require(isinstance(entries, list) and entries, 'missing artifact inventory')
    records = {}
    for item in entries:
        require(isinstance(item, dict) and isinstance(item.get('path'), str), 'invalid artifact record')
        path = item['path']
        require(path not in records, 'duplicate artifact path: ' + path)
        parsed = PurePosixPath(path)
        require(not parsed.is_absolute() and str(parsed) == path and '\\' not in path and not any(p in ('.', '..') for p in parsed.parts), 'unsafe artifact path')
        require(not any((root / PurePosixPath(*parsed.parts[:n])).is_symlink() for n in range(1, len(parsed.parts) + 1)), 'symlink artifact path: ' + path)
        require((root / path).is_file(), 'missing artifact bytes: ' + path)
        require(item.get('digest') == 'sha256:' + digest(root / path) and type(item.get('size')) is int and item['size'] == (root / path).stat().st_size, 'artifact bytes differ from inventory: ' + path)
        records[path] = item
    from release_inventory import metadata
    for path, item in records.items():
        kind, producer, touchpoint = metadata(path)
        if path.endswith('.sha256'):
            kind = 'checksum'
        require((item.get('kind'), item.get('producer'), item.get('oscal_touchpoint')) == (kind, producer, touchpoint), 'artifact ownership differs from declared producer: ' + path)
    paths = set(records)
    mandatory = {'scripts/install.sh', 'openapi.json', 'sbom-assessment-results.json', 'sdk/sdk-contract-check.json', 'sdk/sdk-version.txt', 'sdk/smoke/go-modules.txt', 'release/xoscal-server',
                 'release/image.tar', 'release/image-index.json', 'release/image-digest.txt', 'release/image-platforms.json',
                 'release/image-amd64.tar', 'release/image-arm64.tar', policy['profile_path'],
                 'evidence/policy/release-policy.json', 'evidence/policy/analysis-exceptions.json', 'evidence/policy/framework-manifest.yaml'}
    mandatory |= set(policy['frontend_assets'])
    mandatory |= {'evidence/frontend/' + f for f in policy['frontend_files']}
    mandatory |= {'evidence/frontend-sbom/' + f for f in policy['frontend_sbom_files']}
    mandatory |= {'evidence/ci/' + f for f in policy['ci_outputs']}
    if manifest.get('scope') == 'final':
        mandatory |= set(policy['final_outputs'])
    mandatory |= {'evidence/security/' + f for f in policy['security_files']}
    mandatory |= {'evidence/binary-sbom/' + f for f in policy['sbom_files']}
    catalogs = {'frameworks/' + i + '/catalog.json' for i in ids}
    mandatory |= catalogs
    mandatory |= {path + '.sha256' for path in catalogs | {policy['profile_path']}}
    for language in LANGUAGES:
        mandatory |= {'sdk/' + language + '.zip', 'sdk/' + language + '.zip.sha256', 'sdk/smoke/' + language + '.json'}
        mandatory |= {'evidence/sdk-' + language + '/' + f for f in policy['artifact_files']}
        mandatory |= {'evidence/sdk-' + language + '-sbom/' + f for f in policy['sbom_files']}
    for arch in policy['image_arches']:
        mandatory |= {'evidence/image-' + arch + '/' + f for f in policy['image_files']}
        mandatory |= {'evidence/image-' + arch + '-sbom/' + f for f in policy['sbom_files']}
    for os_name, arch in PLATFORMS:
        mandatory |= {'evidence/archive-' + os_name + '_' + arch + '/' + f for f in policy['artifact_files']}
        mandatory |= {'evidence/archive-' + os_name + '_' + arch + '-sbom/' + f for f in policy['sbom_files']}
    require(not mandatory - paths, 'missing required producer outputs: ' + ', '.join(sorted(mandatory - paths)))
    for path in mandatory:
        if path.endswith('/publishers/publishers.json'):
            directory = (root / path).parent
            require(all(file.relative_to(root).as_posix() in records for file in directory.iterdir() if file.is_file()), 'publisher raw metadata missing from signed inventory')
    if manifest.get('scope') == 'final':
        expected_source = {'schema_version': 1, 'release_tag': manifest.get('release_tag'),
                           'source_revision': manifest.get('source_revision'), 'version': manifest.get('release_tag'),
                           'scope': 'clean-tagged-source', 'status': 'passed'}
        require(read_json(root / 'evidence/release-source.json') == expected_source, 'production source identity differs from manifest')
        if 'evidence/presentation-browser/presentation-browser.json' in policy['final_outputs']:
            from presentation_browser_verify import verify as verify_browser
            verify_browser(root)
    require({p for p in paths if p.startswith('frameworks/') and p.endswith('/catalog.json')} == catalogs, 'catalog IDs differ from exact policy')
    require({p for p in paths if p.startswith('frameworks/') and p.endswith('/profile.json')} == {policy['profile_path']}, 'profile set differs from exact policy')
    for path in mandatory:
        if path.endswith('.sha256') and not path.startswith('evidence/'):
            subject = path[:-len('.sha256')]
            require((root / path).read_text().strip() == 'sha256:' + digest(root / subject), 'digest sidecar differs from actual bytes: ' + path)
    for path in catalogs | {policy['profile_path']}:
        document = read_json(root / path)
        kind = 'profile' if path.endswith('/profile.json') else 'catalog'
        require(isinstance(document.get(kind), dict) and document[kind], 'missing OSCAL root: ' + path)
    require((root / 'evidence/ci/tidy.ok').read_text() == 'module-tidy-ok\n', 'invalid root module tidy admission marker')
    replay_constraints(root, records, ids, policy['profile_path'])
    api = read_json(root / 'openapi.json')
    require(isinstance(api.get('paths'), dict) and api['paths'], 'empty OpenAPI operations')
    archives = [item for item in entries if item['path'].startswith('release/') and item['path'].endswith('.tar.gz')]
    require(len(archives) == 4, 'expected exactly four CLI platform archives')
    platform_archives = {}
    version = manifest.get('release_tag', '').removeprefix('v')
    require(version and version != 'dev', 'required admission needs an explicit release tag')
    for item in archives:
        platform = item.get('platform', {})
        target = (platform.get('os'), platform.get('arch'))
        require(target in PLATFORMS and target not in platform_archives, 'duplicate or unsupported CLI platform')
        filename_arch = 'x86_64' if target[1] == 'amd64' else 'arm64'
        require(PurePosixPath(item['path']).name.endswith('_' + version + '_' + target[0].title() + '_' + filename_arch + '.tar.gz'), 'CLI archive tag/platform mismatch')
        platform_archives[target] = item['path']
    require(set(platform_archives) == PLATFORMS, 'missing CLI platform')
    sdk = load_module('release_sdk_verify', 'sdk-verify.py')
    recomputed_sdk = sdk.verify(root / 'sdk')
    require(recomputed_sdk.get('version') == version, 'SDK version differs from release tag')
    require(read_json(root / 'sdk/sdk-contract-check.json') == recomputed_sdk, 'SDK aggregate receipt differs from exact consumer subjects')
    consumer_packages = {'sdk/' + subject['path'] for subject in recomputed_sdk['subjects'] if subject['path'].startswith('packages/')}
    require({path for path in paths if path.startswith('sdk/packages/')} == consumer_packages, 'native SDK package set differs from tested consumer subjects')
    for language, patterns in policy['native_patterns'].items():
        for pattern in patterns:
            matches = list((root / 'sdk/packages' / language).glob(pattern))
            require(len(matches) == 1 and matches[0].relative_to(root).as_posix() in records, 'missing inventoried native SDK package: ' + language + '/' + pattern)
    gate = load_module('release_analysis_gate', 'analysis-gate.py')
    replay_gate(root, 'evidence/frontend', 'frontend-release', gate, exceptions, 'scalar.js')
    replay_gate(root, 'evidence/frontend-sbom', 'frontend-sbom', gate, exceptions)
    replay_gate(root, 'evidence/security', 'security', gate, exceptions)
    replay_gate(root, 'evidence/binary-sbom', 'sbom', gate, exceptions, 'release/xoscal-server')
    require((root / 'sbom-assessment-results.json').read_bytes() == (root / 'evidence/binary-sbom/sbom-assessment-results.oscal.json').read_bytes(), 'displayed SBOM assessment differs from admitted raw bytes')
    for language in LANGUAGES:
        stage, subject = 'evidence/sdk-' + language, 'sdk/' + language + '.zip'
        replay_gate(root, stage, 'artifact-release', gate, exceptions, subject)
        replay_gate(root, stage + '-sbom', 'sbom', gate, exceptions, subject)
    for platform, subject in platform_archives.items():
        stage = 'evidence/archive-' + platform[0] + '_' + platform[1]
        replay_gate(root, stage, 'artifact-release', gate, exceptions, subject)
        replay_gate(root, stage + '-sbom', 'sbom', gate, exceptions, subject)
    oci = load_module('release_oci_inspect', 'inspect-image-archive.py')
    raw, image = oci.inspect(root / 'release/image.tar')
    require((root / 'release/image-index.json').read_bytes() == raw, 'stored OCI index differs from actual archive')
    require((root / 'release/image-digest.txt').read_text().strip() == image['root_digest'], 'stored OCI digest differs from actual archive')
    require(read_json(root / 'release/image-platforms.json') == image, 'stored OCI graph receipt differs from actual archive')
    for platform in image['platforms']:
        arch = platform['arch']
        subject = 'release/image-' + arch + '.tar'
        _, single = oci.inspect(root / subject, {('linux', arch)})
        require(single['platforms'] == [platform], 'scanned platform archive differs from promoted image graph: ' + arch)
        stage = 'evidence/image-' + arch
        replay_gate(root, stage, 'image-release', gate, exceptions, subject)
        report = read_json(root / stage / 'trivy-results.json')
        metadata = report.get('Metadata', {})
        require(metadata.get('ImageID') == platform['config_digest'], 'image scan config differs from promoted image graph: ' + arch)
        config = metadata.get('ImageConfig', {})
        require(config.get('architecture') == arch and config.get('os') == 'linux', 'image scan platform differs from promoted image graph: ' + arch)
        replay_gate(root, stage + '-sbom', 'sbom', gate, exceptions, subject)
    return {'policy_version': 1, 'status': 'passed', 'scope': 'required-release-subject-admission',
            'required_files': len(mandatory), 'sdk_subjects': len(recomputed_sdk['subjects']), 'image_root_digest': image['root_digest']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--manifest', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(required_payload(args.root, read_json(args.manifest)), sort_keys=True))


if __name__ == '__main__':
    main()
