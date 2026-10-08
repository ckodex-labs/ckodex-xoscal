#!/usr/bin/env python3
"""Reject any Blueprint candidate whose complete HTML was not prebuilt."""

import argparse
from pathlib import Path

from blueprint_render import ROOT, render, verify_assets


def check(root):
    path = root / "portal.html"
    if path.read_text() != render():
        raise ValueError("Blueprint HTML differs from the complete deterministic build")
    verify_assets(root)
    print("check-blueprint: complete prebuilt HTML and exact enhancement assets passed")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT / "site")
    args = parser.parse_args()
    try:
        check(args.root)
    except (ValueError, OSError) as error:
        parser.exit(1, "check-blueprint: " + str(error) + "\n")
