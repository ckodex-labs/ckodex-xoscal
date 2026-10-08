"""Synthetic browser receipt refusal tests; these do not run a browser."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

import presentation_browser_verify as browser

SOURCE = Path(__file__).resolve().parents[1]


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value) + "\n")


def make_fixture(root):
    stage, tools = stage_fixture_tools(root)
    fixture_pages(root)
    receipt = {"schema_version": 1, "status": "passed", "failures": [], "modes": [],
               "negative_controls": [], "checks": [{"id": "inventory: read-only same-origin resources", "passed": True}],
               "tools": tools, "inventory_observations": make_log("inventory"), "proxy_denials": []}
    for mode in sorted(browser.MODES):
        receipt["modes"].append(fixture_mode(mode))
        receipt["checks"] += [{"id": name, "passed": True} for name in sorted(browser.mandatory_checks(mode))]
    fixture_negative_controls(receipt)
    names = browser.subjects(root)
    receipt["subjects"] = {name: browser.digest(root / name) for name in names}
    receipt["subject_sizes"] = {name: (root / name).stat().st_size for name in names}
    write_json(stage / browser.FILES[0], receipt)
    return receipt


def stage_fixture_tools(root):
    stage = root / browser.DIRECTORY
    stage.mkdir(parents=True, exist_ok=True)
    sources = {"check-presentation-browser.mjs": SOURCE / "scripts/check-presentation-browser.mjs",
               "check-api-landmarks.mjs": SOURCE / "scripts/check-api-landmarks.mjs"}
    for name in ("package.json", "package-lock.json", "browser-toolchain.json"):
        sources[name] = SOURCE / "dagger/presentation-browser" / ("toolchain.json" if name == "browser-toolchain.json" else name)
    for name, source in sources.items():
        (stage / name).write_bytes(source.read_bytes())
    config = json.loads((stage / "browser-toolchain.json").read_text())
    tools = {key: config[key] for key in ("image", "playwright", "chromium")}
    tools.update(node="v22.0.0", npm="10.0.0")
    for key, name in (("runner_digest", "check-presentation-browser.mjs"),
                      ("api_helper_digest", "check-api-landmarks.mjs"),
                      ("package_lock_digest", "package-lock.json"),
                      ("toolchain_digest", "browser-toolchain.json"),
                      ("package_digest", "package.json")):
        tools[key] = browser.digest(stage / name)
    return stage, tools


def fixture_pages(root):
    for route in browser.ROUTES:
        (root / route).write_text('<main id="main"><h1>Synthetic fixture</h1><section>Fixture</section></main>')
    for name in ("openapi.json", "downloads.json", "provenance.json", "sbom-assessment-results.json"):
        if not (root / name).exists():
            (root / name).write_text("{}\n")
    if not (root / "THIRD-PARTY-NOTICES.txt").exists():
        (root / "THIRD-PARTY-NOTICES.txt").write_text("Synthetic notices\n")
    installer = root / "scripts/install.sh"
    installer.parent.mkdir(exist_ok=True)
    if not installer.exists():
        installer.write_text("#!/bin/sh\n")


def fixture_mode(mode):
    log = make_log(mode)
    log.update(javaScriptEnabled=mode.endswith("-js"), observations={})
    for route in browser.ROUTES | {"/"}:
        observation = {"initial": {"mains": 1, "canonical": 1, "duplicates": [], "unresolved": []},
                       "native": {"sections": 1, "headings": 1, "links": 0},
                       "geometry": {"h1": {"top": 0}, "firstSection": 20}}
        if route == "docs.html":
            observation["api"] = {"operations": 96, "models": 395}
            observation["api_native_contracts"] = 491
        if route == "portal.html":
            observation["blueprint"] = {"anchors": list(range(6)), "views": list(range(6)), "containers": 15}
        log["observations"][route] = observation
    return log


def fixture_negative_controls(receipt):
    for mode in ("desktop-js", "mobile-js"):
        for failure in ("404", "tamper", "script"):
            log = make_log(mode + ":" + failure)
            if failure != "tamper":
                status = 404 if failure == "404" else 503
                url = "http://127.0.0.1:8081" + ("/openapi.json" if failure == "404" else "/scalar.js")
                log["http_failures"] = [{"url": url, "status": status, "intentional": True}]
                log["console"] = [{"type": "error", "text": "Synthetic intentional HTTP " + str(status)}]
            receipt["negative_controls"].append(log)
            receipt["checks"].append({"id": log["id"] + ": complete Plain HTML is the initial or failed-enhancement view", "passed": True})


def make_log(identifier):
    return {"id": identifier, "requests": [{"url": "http://127.0.0.1:8081/index.html", "method": "GET"}],
            "console": [], "network": [], "network_failures": [], "page_errors": [], "http_failures": []}


class BrowserReceiptTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.receipt = make_fixture(self.root)

    def save(self, value):
        write_json(self.root / browser.DIRECTORY / browser.FILES[0], value)

    def test_complete_synthetic_fixture(self):
        self.assertEqual(browser.verify(self.root)["status"], "passed")

    def test_failed_missing_and_truncated_observations_refused(self):
        mutations = [lambda r: r.update(status="failed"), lambda r: r.update(checks=[]),
                     lambda r: r.update(checks=[c for c in r["checks"] if c["id"] != "desktop-js: native contract 490: keyboard collapse"]),
                     lambda r: r["checks"].pop(), lambda r: r["checks"][0].update(passed=1),
                     lambda r: r["checks"].append(r["checks"][0]), lambda r: r["modes"].pop(),
                     lambda r: r["modes"][0]["observations"].pop("docs.html"),
                     lambda r: r["modes"][0]["observations"]["docs.html"]["api"].update(operations=0),
                     lambda r: r["modes"][0]["observations"]["docs.html"].update(api_native_contracts=490),
                     lambda r: r["modes"][0]["observations"]["portal.html"]["native"].update(sections=0),
                     lambda r: r["negative_controls"].pop(), lambda r: r["modes"][0]["page_errors"].append("error"),
                     lambda r: r["modes"][0]["requests"][0].update(method="POST"),
                     lambda r: r["modes"][0]["requests"][0].update(url="https://example.com/"),
                     lambda r: r.pop("inventory_observations"),
                     lambda r: r["inventory_observations"]["requests"][0].update(method="POST"),
                     lambda r: r["proxy_denials"].append({"url": "https://example.com/"}),
                     lambda r: r["negative_controls"][0]["http_failures"][0].update(url="http://127.0.0.1:8081/other.json"),
                     lambda r: r["negative_controls"][0]["console"].append({"text": "Unexpected runtime error"})]
        for mutation in mutations:
            with self.subTest(mutation=mutation):
                receipt = copy.deepcopy(self.receipt)
                mutation(receipt)
                self.save(receipt)
                with self.assertRaises(ValueError):
                    browser.verify(self.root)

    def test_changed_subject_and_producer_bytes_refused(self):
        for name in ("docs.html", "scripts/install.sh", browser.DIRECTORY + "/check-api-landmarks.mjs"):
            with self.subTest(name=name):
                path = self.root / name
                original = path.read_bytes()
                path.write_bytes(original + b" altered")
                with self.assertRaises(ValueError):
                    browser.verify(self.root)
                path.write_bytes(original)

    def test_missing_sources_and_mutable_tooling_refused(self):
        stage = self.root / browser.DIRECTORY
        for name in browser.FILES[1:]:
            with self.subTest(name=name):
                path = stage / name
                path.rename(stage / (name + ".missing"))
                with self.assertRaises(ValueError):
                    browser.verify(self.root)
                (stage / (name + ".missing")).rename(path)
        receipt = copy.deepcopy(self.receipt)
        receipt["tools"]["image"] = "mcr.microsoft.com/playwright:latest"
        self.save(receipt)
        with self.assertRaises(ValueError):
            browser.verify(self.root)

    def test_only_exact_correlated_negative_aborts_allowed(self):
        receipt = copy.deepcopy(self.receipt)
        control = next(log for log in receipt["negative_controls"] if log["id"] == "desktop-js:script")
        url = "http://127.0.0.1:8081/scalar.js"
        control["http_failures"][0]["url"] = url
        control["network_failures"] = [{"url": url, "error": "net::ERR_ABORTED", "intentional": True}]
        self.save(receipt)
        self.assertEqual(browser.verify(self.root)["status"], "passed")
        for field, value in (("url", "http://127.0.0.1:8081/unexpected.js"),
                             ("error", "net::ERR_CONNECTION_REFUSED"), ("intentional", False)):
            with self.subTest(field=field):
                altered = copy.deepcopy(receipt)
                item = next(log for log in altered["negative_controls"] if log["id"] == "desktop-js:script")
                item["network_failures"][0][field] = value
                self.save(altered)
                with self.assertRaises(ValueError):
                    browser.verify(self.root)
