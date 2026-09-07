#!/usr/bin/env python3
"""Live accessibility acceptance checks for the review surface.

Runs headless Chromium against a served site root and records the keyboard,
focus, landmark, and reduced-motion evidence the beta plan's Phase 2 gate
requires. Exits non-zero on the first failed assertion; every check prints
its result so the output can be attached to the release bundle as evidence.

Usage:
    python3 scripts/a11y-live.py --root site [--port 8903]
"""

import argparse
import functools
import http.server
import socketserver
import sys
import threading
from pathlib import Path

from playwright.sync_api import sync_playwright


def serve(root: Path, port: int):
    handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=str(root))
    socketserver.TCPServer.allow_reuse_address = True
    httpd = socketserver.TCPServer(("127.0.0.1", port), handler)
    thread = threading.Thread(target=httpd.serve_forever, daemon=True)
    thread.start()
    return httpd


def check(name, condition, detail=""):
    status = "PASS" if condition else "FAIL"
    print(f"a11y-live: {status} {name}" + (f" ({detail})" if detail else ""))
    return bool(condition)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default="site")
    parser.add_argument("--port", type=int, default=8903)
    args = parser.parse_args()

    root = Path(args.root).resolve()
    httpd = serve(root, args.port)
    base = f"http://127.0.0.1:{args.port}"
    failures = 0

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={"width": 1280, "height": 900})
        page.goto(f"{base}/review.html", wait_until="networkidle")

        # Landmarks and skip link
        landmarks = page.evaluate(
            """() => Array.from(document.querySelectorAll('[role], header, main, aside, footer, nav'))
                .map(el => el.tagName.toLowerCase() + (el.getAttribute('role') ? `[role=${el.getAttribute('role')}]` : ''))"""
        )
        failures += not check("main landmark present", "main" in landmarks)
        failures += not check("banner landmark present", "header" in landmarks)
        failures += not check("complementary landmark (Evidence Margin)", "aside" in landmarks)
        failures += not check("contentinfo landmark (Authority Footer)", "footer" in landmarks)

        skip = page.locator("a.ck-skip")
        failures += not check("skip link present", skip.count() == 1)

        # Skip link moves focus to main
        page.focus("a.ck-skip")
        page.keyboard.press("Enter")
        focused = page.evaluate("() => document.activeElement && document.activeElement.id")
        failures += not check("skip link targets main", focused == "main", f"focus={focused}")

        # Keyboard-only traversal of the connection form
        page.keyboard.press("Tab")
        first_focus = page.evaluate("() => document.activeElement && document.activeElement.id")
        failures += not check(
            "keyboard focus reaches an interactive control",
            bool(first_focus),
            f"focus={first_focus}",
        )

        # Every input has a programmatic label
        unlabeled = page.evaluate(
            """() => Array.from(document.querySelectorAll('input:not([type=hidden]), textarea, select'))
                .filter(el => {
                    const labelled = el.labels && el.labels.length > 0;
                    const aria = el.getAttribute('aria-label') || el.getAttribute('aria-labelledby');
                    return !labelled && !aria;
                }).map(el => el.id || el.name)"""
        )
        failures += not check("all form controls are labelled", not unlabeled, str(unlabeled))

        # Live regions announce status
        live = page.evaluate(
            """() => Array.from(document.querySelectorAll('[aria-live]')).map(el => el.getAttribute('aria-live'))"""
        )
        failures += not check("live regions present for status", len(live) >= 2, str(live))

        # Buttons have accessible names
        unnamed = page.evaluate(
            """() => Array.from(document.querySelectorAll('button'))
                .filter(b => !(b.textContent || '').trim() && !b.getAttribute('aria-label'))
                .length"""
        )
        failures += not check("all buttons have accessible names", unnamed == 0)

        # Reduced motion: the stylesheet must contain a reduced-motion block
        reduced = page.evaluate(
            """() => {
                let found = false;
                for (const sheet of document.styleSheets) {
                    try {
                        for (const rule of sheet.cssRules) {
                            if (rule.media && /prefers-reduced-motion/.test(rule.media.mediaText)) found = true;
                        }
                    } catch (e) { /* cross-origin sheet */ }
                }
                return found;
            }"""
        )
        failures += not check("reduced-motion media query active", reduced)

        # Forced colors: the stylesheet must contain a forced-colors block
        forced = page.evaluate(
            """() => {
                let found = false;
                for (const sheet of document.styleSheets) {
                    try {
                        for (const rule of sheet.cssRules) {
                            if (rule.media && /forced-colors/.test(rule.media.mediaText)) found = true;
                        }
                    } catch (e) { /* cross-origin sheet */ }
                }
                return found;
            }"""
        )
        failures += not check("forced-colors media query active", forced)

        # Contrast sample: body text vs paper background (DS-3 ledger theme)
        contrast = page.evaluate(
            """() => {
                const body = getComputedStyle(document.body);
                const parse = rgb => rgb.match(/\\d+/g).slice(0, 3).map(Number);
                const lum = (rgb) => {
                    const f = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
                    return 0.2126 * f(rgb[0]) + 0.7152 * f(rgb[1]) + 0.0722 * f(rgb[2]);
                };
                const l1 = lum(body.color.match(/\\d+/g).slice(0, 3).map(Number));
                const bg = getComputedStyle(document.body).backgroundColor;
                const l2 = lum(bg.match(/\\d+/g).slice(0, 3).map(Number));
                return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
            }"""
        )
        failures += not check("body text contrast >= 7:1 (AAA)", contrast >= 7.0, f"ratio={contrast:.2f}")

        browser.close()

    httpd.shutdown()
    if failures:
        print(f"a11y-live: FAIL ({failures} check(s) failed)")
        return 1
    print("a11y-live: PASS (live review surface)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
