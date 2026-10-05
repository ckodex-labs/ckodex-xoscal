#!/usr/bin/env python3
"""Serve final static bytes and check API, assets, inventory and no-JS disclosure."""
from __future__ import annotations

import argparse
import functools
import http.server
import importlib.util
import json
import re
import subprocess
import threading
import urllib.error
import urllib.parse
import urllib.request
import zipfile
from html.parser import HTMLParser
from pathlib import Path

from release_inventory import digest, safe_path, validate

LANGUAGES = ("go", "python", "java", "csharp", "ts", "swift")
SERVICE_COUNTS = {"OscalService": 41, "GovernanceService": 28,
                  "TransparencyExchangeService": 14, "TransparencyGraphService": 13}


class Page(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.links, self.ids, self.downloads = [], set(), []
        self.main_depth, self.depth, self.main_text = None, 0, []
        self.feed(text)

    def handle_starttag(self, tag, attributes):
        attrs = dict(attributes)
        if attrs.get("id"):
            self.ids.add(attrs["id"])
        for key in ("href", "src"):
            if attrs.get(key):
                self.links.append(attrs[key])
        if "download" in attrs and attrs.get("href"):
            self.downloads.append(attrs["href"])
        if tag == "main":
            self.main_depth = self.depth
        if tag not in ("meta", "link", "img", "br", "hr", "input", "source", "area", "embed", "wbr"):
            self.depth += 1

    def handle_endtag(self, tag):
        self.depth -= 1
        if tag == "main":
            self.main_depth = None

    def handle_data(self, data):
        if self.main_depth is not None:
            self.main_text.append(data)


def check_openapi(root):
    document = json.loads((root / "openapi.json").read_text())
    paths = document.get("paths")
    if document.get("openapi") != "3.1.0" or not paths:
        raise ValueError("OpenAPI must be JSON 3.1 with nonempty service paths")
    counts, seen = dict.fromkeys(SERVICE_COUNTS, 0), set()
    for path, methods in paths.items():
        for verb, operation in methods.items():
            if verb not in ("get", "post", "put", "patch", "delete", "head", "options", "trace"):
                continue
            identity = operation.get("operationId", "")
            parts = identity.split(".")
            if len(parts) != 5 or parts[:3] != ["oscal", "services", "v1"] or parts[3] not in counts or identity in seen:
                raise ValueError(f"invalid or duplicate API operation {identity}")
            seen.add(identity)
            counts[parts[3]] += 1
            if not operation.get("responses"):
                raise ValueError(f"API operation has no responses: {identity}")
            names = {p.get("name") for p in operation.get("parameters", []) if p.get("in") == "path" and p.get("required") is True}
            if set(re.findall(r"\{([^}]+)\}", path)) - names:
                raise ValueError(f"API operation missing path parameters: {identity}")
    if counts != SERVICE_COUNTS:
        raise ValueError(f"API service operation counts differ: {counts}")
    expected = {("/v1/component-definitions", "get"): "OscalService.ListComponentDefinitions",
                ("/v1/search", "get"): "OscalService.Search",
                ("/v1/search/semantic", "get"): "GovernanceService.SemanticSearch",
                ("/v1/transparency/claims", "post"): "TransparencyExchangeService.CreateClaim",
                ("/v1/transparency/claims/{claim_id}/receipt", "get"): "TransparencyExchangeService.ExportClaimReceipt",
                ("/v1/graph/verify-closure", "post"): "TransparencyGraphService.VerifyClosure"}
    for (path, verb), identity in expected.items():
        if paths.get(path, {}).get(verb, {}).get("operationId") != "oscal.services.v1." + identity:
            raise ValueError(f"canonical operation missing: {verb} {path}")

    def references(node):
        if isinstance(node, dict):
            if "$ref" in node:
                ref = node["$ref"]
                if not isinstance(ref, str) or not ref.startswith("#/"):
                    raise ValueError(f"nonlocal API reference: {ref}")
                target = document
                for token in ref[2:].split("/"):
                    target = target[token.replace("~1", "/").replace("~0", "~")]
            for child in node.values():
                references(child)
        elif isinstance(node, list):
            for child in node:
                references(child)
    references(document)


def check_generated(root, pages):
    manifest_path = root / "payload-manifest.json"
    manifest = json.loads(manifest_path.read_text())
    validate(root, manifest)
    index = json.loads((root / "downloads.json").read_text())
    if (index.get("release_tag"), index.get("source_revision")) != (
            manifest["release_tag"], manifest["source_revision"]):
        raise ValueError("download inventory identity differs from payload manifest")
    originals = {a["path"]: a for a in manifest["artifacts"]}
    entries = index.get("artifacts", [])
    if len(entries) != len(originals) or {a["path"] for a in entries} != set(originals):
        raise ValueError("download inventory differs from manifest artifact coverage")
    for item in entries:
        original = originals[item["path"]]
        if any(item.get(key) != original.get(key) for key in ("kind", "producer", "oscal_touchpoint", "size", "digest")):
            raise ValueError(f"download metadata differs from manifest: {item['path']}")
    verified = index.get("verification") == "verified"
    if index.get("verification") not in ("verified", "incomplete"):
        raise ValueError("unknown verification state")
    links = set(pages["downloads.html"].downloads)
    for item in entries:
        if item.get("eligible") is not verified:
            raise ValueError(f"eligibility differs from verification: {item['path']}")
        original_refs = originals[item["path"]].get("evidence_refs")
        if not isinstance(original_refs, list):
            raise ValueError("manifest evidence references must be a list")
        for reference in original_refs:
            path = str(safe_path(reference))
            if path not in originals or not (root / path).is_file():
                raise ValueError("artifact evidence reference lacks inventoried local bytes: " + path)
        expected_refs = original_refs + (["payload-manifest.json", "proofs/payload.sigstore.json"] if verified else [])
        if item.get("evidence_refs") != expected_refs:
            raise ValueError("artifact evidence references differ from manifest and verification state: " + item["path"])
    if verified:
        spec = importlib.util.spec_from_file_location("smoke_release_contract", Path(__file__).with_name("release-contract.py"))
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        receipt = json.loads((root / "proofs/payload-verification.json").read_text())
        checked = module.verify_attestation(root, manifest_path, root / "proofs/payload.sigstore.json", manifest["release_tag"], manifest["source_revision"])
        for key in ("schema_version", "manifest_digest", "bundle_digest", "release_tag", "source_revision", "repository", "workflow", "verifier"):
            if receipt.get(key) != checked[key]:
                raise ValueError(f"verification receipt differs from fresh verifier: {key}")
        if not receipt.get("results"):
            raise ValueError("verification receipt lacks results")
        downloadable = {a["path"] for a in entries
                        if a["kind"] in ("cli-archive", "release-evidence", "sdk-source", "sdk-package", "sdk-metadata", "oscal-catalog", "oscal-profile", "api-spec", "installer")
                        or (a["kind"] == "checksum" and a.get("release_asset_name"))}
        if links != downloadable:
            raise ValueError("verified no-JS download links differ from inventory")
    elif links or "Unsigned local candidate" not in "".join(pages["downloads.html"].main_text):
        raise ValueError("unsigned no-JS downloads must be unavailable and disclosed")
    for language in LANGUAGES:
        path = root / f"sdk/{language}.zip"
        with zipfile.ZipFile(path) as archive:
            if not archive.namelist() or archive.testzip() is not None:
                raise ValueError(f"empty or corrupt SDK ZIP: {language}")
            if any(name.startswith(("/", "\\")) or ".." in Path(name).parts or "\\" in name for name in archive.namelist()):
                raise ValueError(f"unsafe SDK archive path: {language}")
        check_sidecar(path)
    catalogs = sorted(root.glob("frameworks/*/catalog.json"))
    if len(catalogs) != 35:
        raise ValueError(f"expected 35 catalogs, got {len(catalogs)}")
    for path in catalogs + [root / "frameworks/cccs-medium-cloud-pbmm/profile.json"]:
        model = "profile" if path.name == "profile.json" else "catalog"
        if model not in json.loads(path.read_text()):
            raise ValueError(f"OSCAL {model} root missing: {path}")
        check_sidecar(path)
    if not (root / "scripts/install.sh").read_bytes():
        raise ValueError("installer missing or empty")
    if not (root / "scalar.js").read_bytes():
        raise ValueError("local Scalar asset missing or empty")
    if (root / "release-manifest.json").exists():
        validate(root, json.loads((root / "release-manifest.json").read_text()), final=True)


def check_sidecar(path):
    if Path(str(path) + ".sha256").read_text().strip() != digest(path):
        raise ValueError(f"digest sidecar mismatch: {path}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path("site"))
    parser.add_argument("--generated", action="store_true", help="require complete Dagger outputs and exact inventory")
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
    errors, fetched, pages = [], set(), {}

    def fetch(route):
        if route in fetched:
            return (root / route).read_bytes()
        with urllib.request.urlopen(base + "/" + urllib.parse.quote(route, safe="/"), timeout=10) as response:
            body = response.read()
            if response.status != 200 or not body:
                raise ValueError(f"{route}: HTTP {response.status} or empty body")
            if body != (root / route).read_bytes():
                raise ValueError(f"{route}: served bytes differ from staged file")
        fetched.add(route)
        return body

    try:
        for path in sorted(root.glob("*.html")):
            body = fetch(path.name).decode("utf-8")
            pages[path.name] = Page(body)
            for marker in ('<main id="main"', '<aside class="ck-shell__margin', '<footer class="ck-shell__footer'):
                if marker not in body:
                    errors.append(f"{path.name}: missing semantic landmark {marker}")
            if "releases/latest" in body:
                errors.append(f"{path.name}: mutable latest-release reference")
        for name in ("index.html", "portal.html", "review.html", "docs.html", "cli.html", "downloads.html", "sbom-validation.html", "transparency.html"):
            if name not in pages:
                errors.append(f"missing required page {name}")
        for name, page in pages.items():
            for link in page.links:
                parts = urllib.parse.urlsplit(link)
                if parts.scheme or parts.netloc:
                    if "releases/latest" in link:
                        errors.append(f"{name}: mutable release link {link}")
                    continue
                target = (Path(name).parent / urllib.parse.unquote(parts.path or name)).as_posix()
                if target in ("scalar.js", "scripts/install.sh") and not args.generated:
                    continue
                if not (root / target).resolve().is_relative_to(root):
                    errors.append(f"{name}: link escapes site: {link}")
                    continue
                try:
                    fetch(target)
                    if parts.fragment and target in pages and parts.fragment not in pages[target].ids:
                        errors.append(f"{name}: missing fragment {link}")
                except (OSError, ValueError, urllib.error.URLError) as error:
                    errors.append(f"{name}: broken local link {link}: {error}")
        css = fetch("styles.css").decode()
        for link in re.findall(r"url\(['\"]?([^)'\"]+)", css):
            if not urllib.parse.urlsplit(link).scheme:
                fetch(link)
        review = fetch("review.html").decode()
        for marker in ('<div id="review-status" role="status"', '<form id="import-form"', 'id="import-form" class="ck-quiet ck-padded ck-stack--tight" novalidate', 'data-import-field="evidence-content"'):
            if marker not in review:
                errors.append(f"review.html: missing beta interaction contract {marker}")
        review_js = fetch("review.js").decode()
        for marker in ("Import preflight blocked:", "Import failed:", "graph/projection-events", "Graph projection history", 'addReceipt("rejected"'):
            if marker not in review_js:
                errors.append(f"review.js: missing actionable rejection path {marker}")
        for path in root.glob("*.js"):
            if "releases/latest" in path.read_text():
                errors.append(f"{path.name}: mutable latest-release fetch")
        fetch("openapi.json")
        check_openapi(root)
        docs = fetch("docs.html").decode()
        for marker in ('data-url="./openapi.json"', 'src="./scalar.js"', "port 50051", "port 9090", "/v1/component-definitions", "/v1/transparency/claims/{claim_id}/receipt"):
            if marker not in docs:
                errors.append(f"docs.html: missing API contract {marker}")
        if args.generated:
            for asset in ("scalar.js", "scalar.js.map", "scalar.css", "scalar.css.map",
                          "THIRD-PARTY-NOTICES.txt", "scripts/install.sh"):
                fetch(asset)
            check_generated(root, pages)
        else:
            for name in ("downloads.html", "transparency.html", "sbom-validation.html"):
                if pages[name].downloads or "Incomplete:" not in "".join(pages[name].main_text):
                    errors.append(f"{name}: source template must disclose missing proof without download links")
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError, zipfile.BadZipFile, urllib.error.URLError) as error:
        errors.append(str(error))
    finally:
        server.shutdown()
        thread.join(timeout=5)
        server.server_close()
    if errors:
        print("site-smoke: FAIL")
        print("\n".join(f"- {error}" for error in errors))
        return 1
    print(f"site-smoke: PASS ({len(fetched)} HTTP assets; 96 API operations; {'generated inventory' if args.generated else 'source templates'} from {root})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
