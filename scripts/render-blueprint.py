#!/usr/bin/env python3
"""Build complete static Blueprint HTML before analysis, manifests or signing."""

import argparse
import json
import sys
from pathlib import Path

from blueprint_render import INPUTS, ROOT, asset_digests, render, verify_assets


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, help="Dagger output path; defaults to both tracked previews")
    parser.add_argument("--check", action="store_true", help="reject generated HTML drift without writing")
    args = parser.parse_args()
    if args.check or args.output:
        verify_assets(ROOT / "site")
    targets = [(args.output, False)] if args.output else [(ROOT / "site/portal.html", False), (ROOT / "portal-preview.html", True)]
    for path, preview in targets:
        expected = render(preview=preview)
        if args.check:
            if not path.is_file() or path.read_text() != expected:
                raise ValueError("generated Blueprint HTML differs: " + str(path))
        else:
            path.write_text(expected)
    if not args.check and not args.output:
        (INPUTS / "enhancement-sha256.json").write_text(json.dumps(asset_digests(ROOT / "site"), indent=2) + "\n")
    print(f"render-blueprint: {'checked' if args.check else 'generated'} {len(targets)} complete HTML surface(s)")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, OSError) as error:
        print("render-blueprint: " + str(error), file=sys.stderr)
        sys.exit(1)
