#!/usr/bin/env python3
"""Fail-closed static checks for the xOSCAL DS-3 delivery surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
HTML_FILES = sorted((ROOT / "site").glob("*.html")) + [ROOT / "portal-preview.html"]
FORBIDDEN_GLYPHS = ("✓", "✗", "•", "·")


def fail(errors: list[str], path: Path, message: str) -> None:
    errors.append(f"{path.relative_to(ROOT)}: {message}")


def check_page(path: Path, errors: list[str]) -> None:
    text = path.read_text(encoding="utf-8")
    rel = path.relative_to(ROOT)

    required = {
        'class="ck-skip" href="#main"': "skip link",
        "<header": "header landmark",
        "<nav": "navigation landmark",
        '<main id="main"': "main landmark",
        '<aside class="ck-shell__margin': "Evidence Margin landmark",
        '<footer class="ck-shell__footer': "Authority Footer landmark",
    }
    if not re.search(r'<html\b[^>]*lang="en"[^>]*data-theme="ledger"', text):
        fail(errors, path, "ledger root theme")
    for marker, label in required.items():
        if marker not in text:
            fail(errors, path, f"missing {label}")

    if not re.search(r'<link[^>]+href="(?:site/)?styles\.css"', text):
        fail(errors, path, "canonical styles.css is not linked")
    if "fonts.googleapis.com" in text or "fonts.gstatic.com" in text:
        fail(errors, path, "external font runtime is not allowed")
    for glyph in FORBIDDEN_GLYPHS:
        if glyph in text:
            fail(errors, path, f"forbidden glyph {glyph!r} appears in product markup")
    if path.parent == ROOT / "site" and path.name != "portal.html" and 'href="#"' in text:
        fail(errors, path, "placeholder href is not allowed outside the sample portal")
    if path.name in {"portal.html", "portal-preview.html"}:
        if 'href="#"' in text and "ck-action-unwired" not in text:
            fail(errors, path, "sample action placeholders lack an inert-action guard")
        if "illustrative data" not in text:
            fail(errors, path, "sample data boundary is not disclosed")


def main() -> int:
    errors: list[str] = []
    if not HTML_FILES:
        print("design-lint: no HTML delivery surfaces found", file=sys.stderr)
        return 1

    stylesheet = ROOT / "site" / "styles.css"
    if not stylesheet.is_file():
        errors.append("site/styles.css: canonical stylesheet is missing")
    else:
        css = stylesheet.read_text(encoding="utf-8")
        for marker, label in (
            ('[data-theme="vault"]', "vault theme"),
            ('[data-theme="hc"]', "high-contrast theme"),
            ("@media (forced-colors: active)", "forced-colors mode"),
            ("@media (prefers-reduced-motion: reduce)", "reduced-motion mode"),
            ("@font-face", "self-hosted evidence font"),
        ):
            if marker not in css:
                errors.append(f"site/styles.css: missing {label}")
        if "fonts.googleapis.com" in css or "@import" in css:
            errors.append("site/styles.css: external font or import runtime is not allowed")
        for font in ("JetBrainsMono-VariableFont_wght.ttf", "JetBrainsMono-Italic-VariableFont_wght.ttf"):
            if not (ROOT / "site" / "fonts" / font).is_file():
                errors.append(f"site/fonts/{font}: self-hosted font asset is missing")

    for page in HTML_FILES:
        check_page(page, errors)

    if errors:
        print("design-lint: FAIL")
        print("\n".join(f"- {error}" for error in errors))
        return 1

    print(f"design-lint: PASS ({len(HTML_FILES)} HTML surfaces, DS-3 themes and landmarks present)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
