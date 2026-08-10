#!/usr/bin/env python3
"""Serve a static site directory and exercise its local delivery surface."""

from __future__ import annotations

import argparse
import functools
import http.server
import threading
import urllib.error
import urllib.request
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path("site"))
    args = parser.parse_args()
    root = args.root.resolve()
    if not root.is_dir():
        print(f"site-smoke: missing root {root}")
        return 1

    handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=str(root))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    base = f"http://127.0.0.1:{server.server_port}"
    routes = ["index.html", "portal.html", "docs.html", "downloads.html", "sbom-validation.html", "transparency.html", "styles.css"]
    errors: list[str] = []
    try:
        for route in routes:
            try:
                with urllib.request.urlopen(f"{base}/{route}", timeout=5) as response:
                    body = response.read()
                    if response.status != 200 or not body:
                        errors.append(f"{route}: HTTP {response.status} or empty body")
            except (urllib.error.URLError, OSError) as error:
                errors.append(f"{route}: {error}")
        for route in ("portal.html", "docs.html", "downloads.html", "sbom-validation.html", "transparency.html"):
            body = urllib.request.urlopen(f"{base}/{route}", timeout=5).read().decode("utf-8")
            for marker in ('<main id="main"', '<aside class="ck-shell__margin', '<footer class="ck-shell__footer'):
                if marker not in body:
                    errors.append(f"{route}: missing rendered marker {marker}")
    finally:
        server.shutdown()
        thread.join(timeout=5)

    if errors:
        print("site-smoke: FAIL")
        print("\n".join(f"- {error}" for error in errors))
        return 1
    print(f"site-smoke: PASS ({len(routes)} HTTP routes served from {root})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
