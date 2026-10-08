"""Synthetic browser receipt refusal tests; these do not run a browser."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

import presentation_browser_verify as browser
from presentation_browser_fixture import fixture_route_evidence

SOURCE = Path(__file__).resolve().parents[1]


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value) + "\n")


def make_fixture(root):
    stage, tools = stage_fixture_tools(root)
    fixture_pages(root)
    receipt = {"schema_version": 1, "status": "passed", "failures": [], "modes": [],
               "negative_controls": [], "checks": [{"id": "inventory: read-only same-origin resources", "passed": True}],
               "tools": tools, "inventory_observations": make_log("inventory"), "proxy_denials": [],
               "execution": {"deadline_ms": 1500000, "elapsed_ms": 1000}}
    for mode in sorted(browser.MODES):
        receipt["modes"].append(fixture_mode(mode))
        receipt["checks"] += [{"id": name, "passed": True} for name in sorted(browser.mandatory_checks(mode))]
    fixture_negative_controls(root, receipt)
    for check in receipt["checks"]:
        if check["id"].endswith(": actual visible point unoccluded"):
            check["detail"] = fixture_coverage_sample(check["id"])
        if check["id"].endswith(": cold contract: cancellation keydown occurs during loading"):
            check["detail"] = {"key": "Tab" if ":focus-return:" in check["id"] else "Enter", "phase": "loading"}
        if check["id"].endswith(": ready transition retains visible unoccluded keyboard focus"):
            sample = {"target": True, "visible": True, "unoccluded": True, "headingVisible": True,
                      "headingUnoccluded": True, "rect": {"top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20},
                      "heading": {"top": 130, "bottom": 150, "left": 10, "right": 210, "width": 200, "height": 20},
                      "viewport": {"width": 390, "height": 844} if check["id"].startswith("mobile") else {"width": 1280, "height": 900}}
            check["detail"] = [copy.deepcopy(sample) for _ in range(4)]
    fixture_route_evidence(root, receipt)
    names = browser.subjects(root)
    receipt["subjects"] = {name: browser.digest(root / name) for name in names}
    receipt["subject_sizes"] = {name: (root / name).stat().st_size for name in names}
    write_json(stage / browser.FILES[0], receipt)
    return receipt



def fixture_coverage_sample(identity):
    sample = {"rect": {"x": 10, "y": 100, "top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20},
              "viewport": {"width": 390, "height": 844} if identity.startswith("mobile") else {"width": 1280, "height": 900},
              "point": {"x": 110, "y": 110}, "hit": "PRE#fixture", "unobscured": True,
              "time": 1000, "focus": "SUMMARY#", "hash": "", "scrollY": 0, "documentHeight": 2000}
    return {**sample, "attempts": [copy.deepcopy(sample)]}


def stage_fixture_tools(root):
    stage = root / browser.DIRECTORY
    stage.mkdir(parents=True, exist_ok=True)
    sources = {"check-presentation-browser.mjs": SOURCE / "scripts/check-presentation-browser.mjs",
               "check-api-landmarks.mjs": SOURCE / "scripts/check-api-landmarks.mjs",
               "check-api-model-intent.mjs": SOURCE / "scripts/check-api-model-intent.mjs"}
    for name in ("package.json", "package-lock.json", "browser-toolchain.json"):
        sources[name] = SOURCE / "dagger/presentation-browser" / ("toolchain.json" if name == "browser-toolchain.json" else name)
    for name, source in sources.items():
        (stage / name).write_bytes(source.read_bytes())
    config = json.loads((stage / "browser-toolchain.json").read_text())
    tools = {key: config[key] for key in ("image", "playwright", "chromium")}
    tools.update(node="v22.0.0", npm="10.0.0")
    for key, name in (("runner_digest", "check-presentation-browser.mjs"),
                      ("api_helper_digest", "check-api-landmarks.mjs"),
                      ("model_intent_digest", "check-api-model-intent.mjs"),
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


def fixture_negative_controls(root, receipt):
    for mode in ("desktop-js", "mobile-js"):
        for failure in ("404", "tamper", "script", "focus-return", "cancel", "forward-focus", "host-failure"):
            log = make_log(mode + ":" + failure)
            if failure in ("focus-return", "cancel", "forward-focus", "host-failure"):
                fixture_cold_control(root, receipt, log, failure)
                continue
            if failure != "script":
                receipt["checks"].append({"id": log["id"] + ": failed explorer: ready transition retains visible unoccluded keyboard focus", "passed": True})
            if failure != "tamper":
                status = 404 if failure == "404" else 503
                url = "http://127.0.0.1:8081" + ("/openapi.json" if failure == "404" else "/scalar.js")
                log["http_failures"] = [{"url": url, "status": status, "intentional": True}]
                log["console"] = [{"type": "error", "text": "Synthetic intentional HTTP " + str(status)}]
            receipt["negative_controls"].append(log)
            receipt["checks"].append({"id": log["id"] + ": complete Plain HTML is the initial or failed-enhancement view", "passed": True})


def fixture_cold_control(root, receipt, log, failure):
    url = "http://127.0.0.1:8081/openapi.json"
    log["held_contract"] = {"status": 200, "digest": browser.digest(root / "openapi.json"),
                            "size": (root / "openapi.json").stat().st_size}
    log["requests"].append({"url": url, "method": "GET"})
    if failure not in ("forward-focus", "host-failure"):
        log["network_failures"] = [{"url": url, "error": "net::ERR_ABORTED", "intentional": True}]
    else:
        fixture_forward_checks(receipt, log, failure)
        receipt["negative_controls"].append(log)
        return
    names = ["cold contract exact actual held response", "cold contract: actual Tab enters explorer summary",
             "cold contract: cancellation keydown occurs during loading", "cold contract: cancelled explorer never resurrects"]
    names.append("cancel explorer: ready transition retains visible unoccluded keyboard focus" if failure == "cancel" else
                 "cold contract: moved Plain focus remains visible and owned")
    for name in names:
        record = {"id": log["id"] + ": " + name, "passed": True}
        if "moved Plain focus" in name:
            record["detail"] = {"target": True, "visible": True, "unoccluded": True,
                "rect": {"top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20},
                "viewport": {"width": 390, "height": 844} if log["id"].startswith("mobile") else {"width": 1280, "height": 900}}
        receipt["checks"].append(record)
    receipt["negative_controls"].append(log)


def fixture_forward_checks(receipt, log, failure):
    sample = {"target": True, "visible": True, "unoccluded": True,
        "rect": {"top": 100, "bottom": 120, "left": 10, "right": 210, "width": 200, "height": 20},
        "viewport": {"width": 390, "height": 844} if log["id"].startswith("mobile") else {"width": 1280, "height": 900}}
    details = {"cold contract exact actual held response": None, "cold forward: actual Tab enters explorer selector": None,
        "cold forward: actual Tab keydown occurs during loading": {"key": "Tab", "phase": "loading"},
        "cold forward: actual Tab selects host contract link": {"tag": "A", "href": "openapi.json", "host": True},
        "cold forward: ready retains visible unoccluded owned host focus": [copy.deepcopy(sample) for _ in range(4)]}
    if failure == "host-failure":
        details.pop("cold forward: ready retains visible unoccluded owned host focus")
        details["failed host explorer: ready transition retains visible unoccluded keyboard focus"] = None
        details["cold host failure: selects only Plain and disposes explorer"] = {"plain": True, "interactive": False, "mounted": False}
        log["http_failures"] = [{"url": "http://127.0.0.1:8081/openapi.json", "status": 404, "intentional": True}]
        log["console"] = [{"type": "error", "text": "Synthetic intentional HTTP 404"}]
    receipt["checks"] += [{"id": log["id"] + ": " + name, "passed": True, "detail": detail} for name, detail in details.items()]


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

    def test_forward_focus_requires_exact_bytes_request_owner_phase_and_geometry(self):
        mutations = [lambda log, checks: log["held_contract"].update(digest="sha256:" + "0" * 64),
            lambda log, checks: log["requests"].append(log["requests"][-1]),
            lambda log, checks: log["requests"][-1].update(url="http://127.0.0.1:8081/openapi.json?other"),
            lambda log, checks: log["requests"].append({"method": "GET", "url": "http://127.0.0.1:8081/openapi.json?extra"}),
            lambda log, checks: log["network_failures"].append({"url": "http://127.0.0.1:8081/openapi.json", "error": "net::ERR_ABORTED", "intentional": True}),
            lambda log, checks: checks["keydown"]["detail"].update(phase="ready"),
            lambda log, checks: checks["identity"]["detail"].update(href="other.json"),
            lambda log, checks: checks["identity"]["detail"].update(host=False),
            lambda log, checks: checks["identity"]["detail"].update(host=1),
            lambda log, checks: checks["geometry"]["detail"][0].update(target=False),
            lambda log, checks: checks["geometry"]["detail"][1]["rect"].update(bottom=90000),
            lambda log, checks: checks["geometry"]["detail"][2]["rect"].update(left=float("nan")),
            lambda log, checks: checks["geometry"]["detail"].pop(),
            lambda log, checks: checks["geometry"].update(id="removed mandatory forward geometry")]
        for mutate in mutations:
            with self.subTest(mutation=mutate):
                receipt = copy.deepcopy(self.receipt)
                log = next(log for log in receipt["negative_controls"] if log["id"] == "desktop-js:forward-focus")
                owned = {item["id"].split(": ", 1)[1]: item for item in receipt["checks"] if item["id"].startswith(log["id"] + ": ")}
                checks = {"keydown": owned["cold forward: actual Tab keydown occurs during loading"],
                    "identity": owned["cold forward: actual Tab selects host contract link"],
                    "geometry": owned["cold forward: ready retains visible unoccluded owned host focus"]}
                mutate(log, checks)
                self.save(receipt)
                with self.assertRaises(ValueError):
                    browser.verify(self.root)

    def test_host404_abort_requires_exact_correlated_missing_response(self):
        receipt = copy.deepcopy(self.receipt)
        log = next(log for log in receipt["negative_controls"] if log["id"] == "desktop-js:host-failure")
        abort = {"url": "http://127.0.0.1:8081/openapi.json", "error": "net::ERR_ABORTED", "intentional": True}
        log["network_failures"] = [abort]
        self.save(receipt)
        self.assertEqual(browser.verify(self.root)["status"], "passed")
        mutations = [lambda l: l["network_failures"].append(l["network_failures"][0]),
            lambda l: l["network_failures"][0].update(url="https://elsewhere/openapi.json"),
            lambda l: l["network_failures"][0].update(error="net::ERR_FAILED"),
            lambda l: l["network_failures"][0].update(intentional=False),
            lambda l: l["http_failures"].clear()]
        for mutate in mutations:
            changed = copy.deepcopy(receipt)
            mutate(next(l for l in changed["negative_controls"] if l["id"] == "desktop-js:host-failure"))
            self.save(changed)
            with self.assertRaises(ValueError):
                browser.verify(self.root)

    def test_host_failure_requires_exact_controlled_http_and_owned_plain_focus(self):
        mutations = [lambda log: log["http_failures"].clear(),
            lambda log: log["http_failures"][0].update(status=503),
            lambda log: log["http_failures"][0].update(url="http://127.0.0.1:8081/other.json"),
            lambda log: log["http_failures"].append(log["http_failures"][0]),
            lambda log: log["console"].append({"text": "Unexpected runtime error"})]
        for mutate in mutations:
            receipt = copy.deepcopy(self.receipt)
            mutate(next(log for log in receipt["negative_controls"] if log["id"] == "desktop-js:host-failure"))
            self.save(receipt)
            with self.assertRaises(ValueError):
                browser.verify(self.root)
        receipt = copy.deepcopy(self.receipt)
        check = next(c for c in receipt["checks"] if c["id"].startswith("desktop-js:host-failure: failed host explorer"))
        check["detail"][1]["target"] = False
        self.save(receipt)
        with self.assertRaises(ValueError):
            browser.verify(self.root)

    def test_host_failure_must_close_interactive_and_dispose_its_rendered_content(self):
        for detail in ({"plain": False, "interactive": False, "mounted": False},
                       {"plain": True, "interactive": True, "mounted": False},
                       {"plain": True, "interactive": False, "mounted": True},
                       {"plain": 1, "interactive": False, "mounted": False},
                       {"plain": True, "interactive": 0, "mounted": False},
                       {"plain": True, "interactive": False, "mounted": 0}, None):
            receipt = copy.deepcopy(self.receipt)
            check = next(c for c in receipt["checks"] if c["id"].startswith("desktop-js:host-failure: cold host failure:"))
            check["detail"] = detail
            self.save(receipt)
            with self.assertRaises(ValueError):
                browser.verify(self.root)

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

    def test_execution_duration_and_fixed_ceiling_cannot_be_counterfeited(self):
        invalid = [{}, {"deadline_ms": 1500001, "elapsed_ms": 1000}]
        invalid += [{"deadline_ms": 1500000, "elapsed_ms": value}
                    for value in (None, True, 0, -1, 1500001, float("nan"), float("inf"))]
        for execution in invalid:
            with self.subTest(execution=execution):
                receipt = copy.deepcopy(self.receipt)
                receipt["execution"] = execution
                with self.assertRaises(ValueError):
                    browser.verify_execution(receipt)

    def test_coverage_attempts_must_be_bounded_and_actual(self):
        detail = fixture_coverage_sample("desktop-js")
        for attempts in (None, [], detail["attempts"] * 4):
            with self.subTest(attempts=attempts):
                value = copy.deepcopy(detail)
                value["attempts"] = attempts
                receipt = {"checks": [{"id": "desktop-js: actual visible point unoccluded", "detail": value}]}
                with self.assertRaises(ValueError):
                    browser.verify_coverage_geometry(receipt)
        detail["attempts"].append(copy.deepcopy(detail["attempts"][0]))
        with self.assertRaises(ValueError):
            browser.verify_coverage_geometry({"checks": [{"id": "desktop-js: actual visible point unoccluded", "detail": detail}]})

    def test_coverage_geometry_cannot_claim_nonfinite_offscreen_or_occluded_pass(self):
        mutations = [lambda s: s["rect"].update(x=float("nan")),
                     lambda s: s["rect"].update(top=float("inf")),
                     lambda s: s["rect"].update(width=True),
                     lambda s: s.update(unobscured=False), lambda s: s.update(unobscured=1),
                     lambda s: s.update(hit="undefined#undefined"), lambda s: s["point"].update(y=9000),
                     lambda s: s.update(time=float("nan")), lambda s: s.update(scrollY=True),
                     lambda s: s.update(viewport={"width": 390, "height": 844}),
                     lambda s: (s["rect"].update(y=-40, top=-40, bottom=-20), s["point"].update(y=-10))]
        for mutate in mutations:
            detail = fixture_coverage_sample("desktop-js")
            mutate(detail["attempts"][0])
            detail.update(detail["attempts"][0])
            receipt = {"checks": [{"id": "desktop-js: actual visible point unoccluded", "detail": detail}]}
            with self.subTest(mutate=mutate), self.assertRaises(ValueError):
                browser.verify_coverage_geometry(receipt)

    def test_coverage_retries_retain_failed_samples_without_repairing_keyboard_evidence(self):
        detail = fixture_coverage_sample("desktop-js")
        failed = copy.deepcopy(detail["attempts"][0])
        failed.update(unobscured=False, time=900, hit="DIV#overlay")
        detail["attempts"].insert(0, failed)
        receipt = {"checks": [{"id": "desktop-js: actual visible point unoccluded", "detail": detail}]}
        browser.verify_coverage_geometry(receipt)
        detail["attempts"][1]["time"] = 899
        with self.assertRaises(ValueError):
            browser.verify_coverage_geometry(receipt)

    def test_coverage_summary_numeric_boolean_equality_cannot_replace_typed_samples(self):
        for key, value in (("unobscured", 1), ("scrollY", False)):
            detail = fixture_coverage_sample("desktop-js")
            detail[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                browser.verify_coverage_geometry({"checks": [{"id": "desktop-js: actual visible point unoccluded", "detail": detail}]})


    def test_changed_subject_and_producer_bytes_refused(self):
        for name in ("docs.html", "scripts/install.sh", browser.DIRECTORY + "/check-api-landmarks.mjs"):
            with self.subTest(name=name):
                path = self.root / name
                original = path.read_bytes()
                path.write_bytes(original + b" altered")
                with self.assertRaises(ValueError):
                    browser.verify(self.root)
                path.write_bytes(original)

    def test_entry_focus_geometry_cannot_be_missing_hidden_or_unowned(self):
        for field in ("target", "visible", "unoccluded", "headingVisible", "headingUnoccluded"):
            with self.subTest(field=field):
                receipt = copy.deepcopy(self.receipt)
                check = next(item for item in receipt["checks"] if item["id"].endswith(": ready transition retains visible unoccluded keyboard focus"))
                check["detail"][2][field] = False
                self.save(receipt)
                with self.assertRaises(ValueError):
                    browser.verify(self.root)
        receipt = copy.deepcopy(self.receipt)
        check = next(item for item in receipt["checks"] if item["id"].endswith(": ready transition retains visible unoccluded keyboard focus"))
        check["detail"].pop()
        self.save(receipt)
        with self.assertRaises(ValueError):
            browser.verify(self.root)

    def test_entry_geometry_claims_cannot_override_actual_viewport_or_dimensions(self):
        mutations = [("bottom", 90000), ("right", 90000), ("top", -1),
                     ("width", 0), ("height", 300), ("left", float("nan")), ("bottom", float("inf"))]
        for target in ("rect", "heading"):
            for field, value in mutations:
                with self.subTest(target=target, field=field):
                    receipt = copy.deepcopy(self.receipt)
                    check = next(item for item in receipt["checks"] if item["id"].endswith(": ready transition retains visible unoccluded keyboard focus"))
                    check["detail"][2][target][field] = value
                    self.save(receipt)
                    with self.assertRaises(ValueError):
                        browser.verify(self.root)

    def test_missing_keyboard_entry_and_failure_focus_checks_refused(self):
        markers = ["API keyboard entry/view link:", "API keyboard entry/plain view link:",
                   "API keyboard entry/native summary:", "cold contract: cancellation keydown occurs during loading", "failed explorer:"]
        for marker in markers:
            with self.subTest(marker=marker):
                receipt = copy.deepcopy(self.receipt)
                removed = next(item for item in receipt["checks"] if marker in item["id"])
                receipt["checks"].remove(removed)
                self.save(receipt)
                with self.assertRaises(ValueError):
                    browser.verify(self.root)

    def test_loading_key_observations_must_match_actual_control(self):
        for value in ({"key": "Tab", "phase": "ready"}, {"key": "Enter", "phase": "loading"}, None):
            receipt = copy.deepcopy(self.receipt)
            check = next(item for item in receipt["checks"] if ":focus-return:" in item["id"] and item["id"].endswith("cancellation keydown occurs during loading"))
            check["detail"] = value
            self.save(receipt)
            with self.assertRaises(ValueError):
                browser.verify(self.root)

    def test_cold_abort_is_bound_to_exact_request_bytes_and_user_focus(self):
        mutations = [lambda log: log["network_failures"][0].update(url="https://elsewhere/openapi.json"),
                     lambda log: log["requests"].append(log["requests"][-1]),
                     lambda log: log["held_contract"].update(digest="sha256:" + "0" * 64),
                     lambda log: log["held_contract"].update(size=0),
                     lambda log: log["network_failures"].append(log["network_failures"][0])]
        for mutate in mutations:
            receipt = copy.deepcopy(self.receipt)
            mutate(next(log for log in receipt["negative_controls"] if log["id"] == "desktop-js:focus-return"))
            self.save(receipt)
            with self.assertRaises(ValueError):
                browser.verify(self.root)
        receipt = copy.deepcopy(self.receipt)
        check = next(item for item in receipt["checks"] if item["id"] == "desktop-js:focus-return: cold contract: moved Plain focus remains visible and owned")
        check["detail"]["rect"]["bottom"] = 90000
        self.save(receipt)
        with self.assertRaises(ValueError):
            browser.verify(self.root)

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
