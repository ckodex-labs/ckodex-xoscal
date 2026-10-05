#!/usr/bin/env python3
"""Fail closed outside the reviewed framework catalog/profile semantic subset.

This is an applicability gate, not a replacement for schema or CLI validation.
"""
import argparse
import hashlib
import json
from pathlib import Path

SCOPE = 'framework-catalog-profile-legacy-semantics-compatibility'
PRODUCER = {
    'cli_version': '1.0.3',
    'cli_distribution_sha256': 'c90166b6f94a8b32a4c0012bb3f3d32dfa54b950aed31ac1a4902ce75fc6a57e',
    'binding_library': 'liboscal-java',
    'binding_library_version': '3.0.3',
    'binding_library_sha256': '65b7e116fc4991ccaf8f6e841d4af955313efbc6aad3b1de4472e0408c352dda',
    'semantic_model_version': '1.1.2',
    'structural_model_version': '1.2.3',
}
MODEL_DIGESTS = {
    '1.1.2': {
        'oscal_catalog_metaschema.xml': '31330afb81453fa628d6b09150e701dbf5812565d4ed8f289e3c4306fd699761',
        'oscal_profile_metaschema.xml': '4d8f048525f0e1175b5a17723d960bf625bb599d982bdcae9d22ea7ac2df17ea',
        'oscal_control-common_metaschema.xml': '09b67d9d3804f803f782d95aaeeaa6609e96e57cfd37319473bd010cc65049be',
        'oscal_metadata_metaschema.xml': '0bea27e467bea722b1bf850b1e8dbd502096f1585602e7b07507004b6963ef99',
    },
    '1.2.3': {
        'oscal_catalog_metaschema.xml': '2242a51cb3e307975a677e26816167df1d3dec759101074e58fcb3f3bca67d67',
        'oscal_profile_metaschema.xml': '6d7473bc0a77239fb9eb48c055c7642c93cb0733242dbee6bb232a0c87405335',
        'oscal_control-common_metaschema.xml': 'e6903981455b839e6043cb02adaf9bb4b3a3f2ace84c24f23980ee3c0ab2d46c',
        'oscal_metadata_metaschema.xml': '2e87a638e604f79c7d26a5fbed236f2c1b045a6baf5ac366fdec74e3404b7bb9',
    },
}
FEATURES = ('hashes', 'locations', 'location-uuids', 'resource-fragment', 'links',
            'status-property', 'with-child-controls')


def model_sources():
    result = []
    for version, files in MODEL_DIGESTS.items():
        for name, digest in files.items():
            result.append({'version': version, 'path': name, 'sha256': digest,
                           'url': 'https://raw.githubusercontent.com/usnistgov/OSCAL/v' + version + '/src/metaschema/' + name})
        result.append({'version': version, 'path': 'shared-constraints/allowed-values-control-group-property-name.ent',
                       'sha256': '6e714787b64f18964aa42fde3763005b2277818dd9c99637870653072f61718d',
                       'url': 'https://raw.githubusercontent.com/usnistgov/OSCAL/v' + version + '/src/metaschema/shared-constraints/allowed-values-control-group-property-name.ent'})
    return result


def affected_features(document):
    counts = dict.fromkeys(FEATURES, 0)

    def visit(value):
        if isinstance(value, dict):
            for key, child in value.items():
                if key in counts:
                    counts[key] += 1
                if key == 'props' and isinstance(child, list):
                    counts['status-property'] += sum(isinstance(prop, dict) and prop.get('name') == 'status' for prop in child)
                visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)
    visit(document)
    return counts


def evaluate(root):
    root = Path(root)
    subjects, findings = [], []
    totals = {'catalog': 0, 'profile': 0}
    for path in sorted(root.glob('*/*.json')):
        if path.name not in ('catalog.json', 'profile.json'):
            continue
        if path.is_symlink() or path.parent.is_symlink() or not path.is_file():
            raise ValueError('semantic subject must be a regular file: ' + str(path))
        data = path.read_bytes()
        document = json.loads(data)
        kind = path.stem
        name = path.relative_to(root).as_posix()
        if not isinstance(document, dict) or set(document) - {'$schema', kind} or not isinstance(document.get(kind), dict):
            raise ValueError('semantic subject has unexpected OSCAL root: ' + name)
        version = document[kind].get('metadata', {}).get('oscal-version')
        if version != PRODUCER['structural_model_version']:
            findings.append({'path': name, 'reason': 'framework requires explicit OSCAL 1.2.3 metadata'})
        counts = affected_features(document[kind])
        for feature, count in counts.items():
            if count:
                findings.append({'path': name, 'feature': feature, 'count': count,
                                 'reason': 'unsupported legacy-model semantic scope'})
        subjects.append({'path': name, 'kind': kind, 'sha256': hashlib.sha256(data).hexdigest(),
                         'size': len(data), 'excluded_feature_counts': counts})
        totals[kind] += 1
    if totals != {'catalog': 35, 'profile': 1}:
        findings.append({'reason': 'scope requires exactly 35 catalogs and one profile', 'actual_counts': totals})
    return {'schema_version': 1, 'scope': SCOPE, 'status': 'failed' if findings else 'passed',
            'producer': dict(PRODUCER), 'model_sources': model_sources(),
            'subject_counts': totals, 'subjects': subjects, 'findings': findings,
            'assurance': 'OSCAL 1.2.3 structural schema plus pinned OSCAL 1.1.2 semantic-model validation within the reviewed framework subset; not general OSCAL 1.2.3 semantic validation'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('frameworks', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    result = evaluate(args.frameworks)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
    if result['status'] != 'passed':
        raise SystemExit('unsupported legacy-model semantic scope; see ' + str(args.output))


if __name__ == '__main__':
    main()
