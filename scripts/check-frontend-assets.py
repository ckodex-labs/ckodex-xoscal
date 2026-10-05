#!/usr/bin/env python3
"""Replay frontend admission against exact served files before presentation."""
import argparse
from pathlib import Path

from release_requirements import load_module, replay_gate


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--exceptions', type=Path, required=True)
    args = parser.parse_args()
    gate = load_module('presentation_analysis_gate', 'analysis-gate.py')
    replay_gate(args.root, 'evidence/frontend', 'frontend-release', gate,
                args.exceptions, subject='scalar.js')
    replay_gate(args.root, 'evidence/frontend-sbom', 'frontend-sbom', gate,
                args.exceptions)
    print('frontend served-byte and strict SBOM admission: PASS')


if __name__ == '__main__':
    main()
