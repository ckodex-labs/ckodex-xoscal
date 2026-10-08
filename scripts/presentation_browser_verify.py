#!/usr/bin/env python3
"""Validate exact-byte browser evidence; hosted provenance supplies producer trust."""

import argparse
import hashlib
import json
import re
from pathlib import Path
from urllib.parse import urlsplit
from html.parser import HTMLParser

DIRECTORY = "evidence/presentation-browser"
FILES = ("presentation-browser.json", "check-presentation-browser.mjs",
         "check-api-landmarks.mjs", "package.json", "package-lock.json",
         "browser-toolchain.json")
MODES = {"desktop-js", "mobile-js", "desktop-nojs", "mobile-nojs"}
ROUTES = {"index.html", "cli.html", "docs.html", "downloads.html", "review.html",
          "transparency.html", "sbom-validation.html", "portal.html"}


def require(condition, message):
    if not condition:
        raise ValueError("browser admission: " + message)


def digest(path):
    return "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()


def subjects(root):
    names = {p.name for p in root.iterdir() if p.is_file() and
             (p.suffix in {".html", ".js", ".css"} or p.name.endswith((".js.map", ".css.map")))}
    names |= {"openapi.json", "downloads.json", "provenance.json",
              "sbom-assessment-results.json", "THIRD-PARTY-NOTICES.txt", "scripts/install.sh"}
    for directory in ("assets", "fonts"):
        names |= {p.relative_to(root).as_posix() for p in (root / directory).rglob("*") if p.is_file()}
    require(all((root / name).is_file() and not (root / name).is_symlink() for name in names),
            "missing or linked presentation subject")
    return names


def verify_tools(stage, receipt):
    config = json.loads((stage / "browser-toolchain.json").read_text())
    tools = receipt.get("tools", {})
    require(config.get("schema_version") == 1, "unsupported toolchain schema")
    require(set(config.get("modes", [])) == MODES and set(config.get("routes", [])) == ROUTES,
            "incomplete toolchain coverage")
    require(re.fullmatch(r"mcr\.microsoft\.com/playwright:v[0-9.]+-noble@sha256:[0-9a-f]{64}", config.get("image", "")),
            "browser image must be immutable")
    for field in ("image", "playwright", "chromium"):
        require(tools.get(field) == config.get(field), "observed tool differs: " + field)
    require(re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+", tools.get("node", "")), "missing actual Node version")
    require(re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", tools.get("npm", "")), "missing actual npm version")
    for field, name in (("runner_digest", "check-presentation-browser.mjs"),
                        ("api_helper_digest", "check-api-landmarks.mjs"),
                        ("package_lock_digest", "package-lock.json"),
                        ("toolchain_digest", "browser-toolchain.json"),
                        ("package_digest", "package.json")):
        require(tools.get(field) == digest(stage / name), "producer source differs: " + name)
    package = json.loads((stage / "package.json").read_text())
    lock = json.loads((stage / "package-lock.json").read_text())
    require(package.get("dependencies") == {"playwright": config.get("playwright")}, "unexpected browser dependencies")
    for name, integrity in config.get("packages", {}).items():
        entry = lock.get("packages", {}).get("node_modules/" + name, {})
        require(entry.get("version") == config["playwright"] and entry.get("integrity") == integrity,
                "browser package lock differs: " + name)
    require(set(config.get("packages", {})) == {"playwright", "playwright-core"}, "incomplete package integrity pins")


class Counts(HTMLParser):
    def __init__(self):
        super().__init__()
        self.sections = self.headings = self.links = 0

    def handle_starttag(self, tag, attrs):
        self.sections += tag == "section"
        self.headings += tag in {"h1", "h2", "h3", "h4", "h5", "h6"}
        self.links += tag == "a" and "href" in dict(attrs)


def verify_requests(log, expected=None):
    require(isinstance(log.get("requests"), list) and log["requests"], "missing raw request observations")
    for request in log["requests"]:
        url = urlsplit(request.get("url", ""))
        require(request.get("method") in {"GET", "HEAD"} and url.scheme == "http" and
                url.hostname == "127.0.0.1" and url.port == 8081, "request exceeded the candidate boundary")
    for field in ("page_errors", "network"):
        require(log.get(field) == [], "unexpected browser " + field)
    require(isinstance(log.get("network_failures"), list), "missing raw request failures")
    for failure in log["network_failures"]:
        require(expected is not None and failure.get("intentional") is True and
                failure.get("error") == "net::ERR_ABORTED" and
                urlsplit(failure.get("url", "")).path == expected[0] and
                any(response.get("url") == failure["url"] and response.get("status") == expected[1] and
                    response.get("intentional") is True for response in log.get("http_failures", [])),
                "unexpected or uncorrelated request failure")


def verify_coverage(root, receipt, mode):
    observations = mode.get("observations", {})
    require(isinstance(observations, dict) and set(observations) == ROUTES | {"/"}, "incomplete actual route coverage")
    for route, observed in observations.items():
        parser = Counts()
        parser.feed((root / ("index.html" if route == "/" else route)).read_text())
        raw = observed.get("initial", {})
        require(raw.get("mains") == raw.get("canonical") == 1 and
                raw.get("duplicates") == raw.get("unresolved") == [], "invalid initial semantic observation")
        native = observed.get("native", {})
        require(native == {"sections": parser.sections, "headings": parser.headings, "links": parser.links},
                "native content coverage differs: " + route)
        geometry = observed.get("geometry", {})
        require(isinstance(geometry.get("h1"), dict), "missing title geometry")
        require(geometry.get("firstSection") is None or geometry["h1"]["top"] < geometry["firstSection"],
                "title follows visible content")
    require(observations["docs.html"].get("api") == {"operations": 96, "models": 395}, "incomplete API observations")
    require(observations["docs.html"].get("api_native_contracts") == 491,
            "incomplete native operation and model disclosure observations")
    blueprint = observations["portal.html"].get("blueprint", {})
    require(len(blueprint.get("anchors", [])) == len(blueprint.get("views", [])) == 6 and
            blueprint.get("containers") == 15, "incomplete Blueprint observations")


def mandatory_checks(mode):
    ids = {mode + ": fresh context no cookies",
           mode + ": all 491 native operation and model disclosures"}
    for route in ROUTES | {"/"}:
        for phase in ("initial", "after native content"):
            prefix = mode + "/" + route + ": " + phase
            ids |= {prefix + ": sole canonical main, unique IDs and ARIA", prefix + ": viewport fits native content"}
    if mode.endswith("-js"):
        for phase in ("online API lifecycle", "loaded offline API lifecycle"):
            for cycle in range(3):
                ids.add(mode + ": " + phase + "/cycle " + str(cycle) + ": Escape dismisses the API client")
    return ids


def verify_negative_controls(receipt):
    controls = receipt.get("negative_controls", [])
    expected = {mode + ":" + failure for mode in ("desktop-js", "mobile-js") for failure in ("404", "tamper", "script")}
    require(len(controls) == 6 and {item.get("id") for item in controls} == expected, "incomplete negative controls")
    ids = {item["id"] for item in receipt["checks"]}
    for log in controls:
        expected_response = (("/openapi.json", 404) if log["id"].endswith(":404") else
                             ("/scalar.js", 503) if log["id"].endswith(":script") else None)
        verify_requests(log, expected_response)
        require(log["id"] + ": complete Plain HTML is the initial or failed-enhancement view" in ids,
                "missing fail-closed fallback assertion")
        if log["id"].endswith(":tamper"):
            require(log.get("console") == log.get("http_failures") == [], "tamper control caused an unexpected runtime error")
        else:
            status = 404 if log["id"].endswith(":404") else 503
            failures = log.get("http_failures", [])
            require(len(failures) == 1 and failures[0].get("status") == status and failures[0].get("intentional") is True,
                    "missing intentional negative HTTP response")


def verify_checks(root, receipt):
    require(receipt.get("schema_version") == 1 and receipt.get("status") == "passed", "missing successful raw receipt")
    checks = receipt.get("checks")
    require(isinstance(checks, list) and len(checks) >= 100, "missing actual checks")
    ids = [item.get("id") for item in checks if isinstance(item, dict)]
    require(len(ids) == len(checks) == len(set(ids)) and all(isinstance(i, str) and i for i in ids),
            "missing or duplicate check identity")
    require(all(item.get("passed") is True for item in checks), "failed, missing or nonboolean check")
    require(receipt.get("failures") == [], "raw failures cannot admit")
    modes = receipt.get("modes", [])
    require(len(modes) == 4 and {mode.get("id") for mode in modes} == MODES, "incomplete browser modes")
    for mode in modes:
        require(mode.get("javaScriptEnabled") is mode["id"].endswith("-js"), "mode JavaScript setting differs")
        verify_requests(mode)
        require(mode.get("console") == mode.get("http_failures") == [], "unexpected console or HTTP failures")
        require(mandatory_checks(mode["id"]) <= set(ids), "missing mandatory browser assertions")
        verify_coverage(root, receipt, mode)
    verify_negative_controls(receipt)


def verify(root, stage=None):
    root = Path(root)
    stage = Path(stage) if stage else root / DIRECTORY
    require(all((stage / name).is_file() and not (stage / name).is_symlink() for name in FILES),
            "missing mandatory browser producer files")
    receipt = json.loads((stage / FILES[0]).read_text())
    verify_checks(root, receipt)
    verify_tools(stage, receipt)
    names = subjects(root)
    require(set(receipt.get("subjects", {})) == names and set(receipt.get("subject_sizes", {})) == names,
            "presentation subject set differs")
    for name in names:
        require(receipt["subjects"][name] == digest(root / name), "stale or altered subject: " + name)
        size = receipt["subject_sizes"][name]
        require(type(size) is int and size == (root / name).stat().st_size, "subject size differs: " + name)
    return {"schema_version": 1, "status": "passed", "subjects": len(names), "checks": len(receipt["checks"])}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--stage", type=Path)
    args = parser.parse_args()
    print(json.dumps(verify(args.root, args.stage), sort_keys=True))
