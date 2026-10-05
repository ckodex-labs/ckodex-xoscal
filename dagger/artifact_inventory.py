#!/usr/bin/env python3
"""Replay dual Trivy analysis of the same safely extracted artifact bytes."""
import argparse
import hashlib
import json
from pathlib import Path


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError('duplicate JSON property: ' + key)
        value[key] = item
    return value


def replay_merge(root):
    results, reports = [], {}
    for mode in ('fs', 'rootfs'):
        status = (root / f'trivy-{mode}.status').read_text().strip()
        if status != '0':
            raise ValueError(f'Trivy {mode} operational failure (exit {status})')
        raw = (root / f'trivy-{mode}-results.json').read_bytes()
        document = json.loads(raw, object_pairs_hook=unique_object)
        if not isinstance(document, dict) or document.get('SchemaVersion') != 2 or document.get('Trivy', {}).get('Version') != '0.72.0' or document.get('ArtifactType') != 'filesystem' or document.get('ArtifactName') != '/input/unpacked':
            raise ValueError(f'malformed Trivy {mode} report')
        targets = document.get('Results', [])
        if not isinstance(targets, list):
            raise ValueError(f'malformed Trivy {mode} targets')
        for result in targets:
            if not isinstance(result, dict) or not result.get('Target') or 'XoscalScanMode' in result:
                raise ValueError(f'malformed Trivy {mode} target')
            # Preserve each observed target and finding; mode identifies any
            # overlap without dropping or conflating either raw observation.
            results.append(dict(result, XoscalScanMode=mode))
        reports[mode] = hashlib.sha256(raw).hexdigest()
    return {'SchemaVersion': 2, 'Trivy': {'Version': '0.72.0'},
            'ArtifactName': '/input/unpacked', 'ArtifactType': 'filesystem',
            'XoscalRawReports': reports, 'Results': results}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--evidence', type=Path, required=True)
    args = parser.parse_args()
    output = replay_merge(args.evidence)
    (args.evidence / 'trivy-results.json').write_text(json.dumps(output, sort_keys=True, indent=2) + '\n')
    print('combined both exact-artifact Trivy modes; empty combined inventory remains inadmissible')


if __name__ == '__main__':
    main()
