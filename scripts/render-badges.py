#!/usr/bin/env python3
"""Informational SVGs; signing and gate verdicts never come from badge artwork."""
import argparse
from html import escape
from pathlib import Path


def render(output, version):
    labels = {'release': 'candidate: ' + version, 'ci': 'CI: inspect raw evidence',
              'license': 'license: see LICENSE', 'oscal': 'OSCAL schema: 1.2.3',
              'go': 'Go root module: 1.25+', 'dagger': 'Dagger engine: 0.21.7',
              'slsa': 'provenance: verification required', 'airgap': 'static assets: local',
              'shieldcn': 'badges: informational'}
    output.mkdir(parents=True, exist_ok=True)
    for name, label in labels.items():
        safe = escape(label, quote=True)
        width = max(160, len(label) * 7 + 20)
        svg = (f'<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="{safe}" width="{width}" height="24">'
               f'<title>{safe}</title><rect width="{width}" height="24" rx="4" fill="#18181b"/>'
               f'<text x="10" y="16" fill="#fff" font-family="monospace" font-size="11">{safe}</text></svg>\n')
        (output / (name + '.svg')).write_text(svg)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--version', required=True)
    args = parser.parse_args()
    render(args.output, args.version)
