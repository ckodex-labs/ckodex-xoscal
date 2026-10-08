#!/usr/bin/env python3
"""Validate exact-byte browser evidence; hosted provenance supplies producer trust."""

import argparse
import hashlib
import json
import math
import re
from pathlib import Path
from urllib.parse import urlsplit
from html.parser import HTMLParser

DIRECTORY = "evidence/presentation-browser"
FILES = ("presentation-browser.json", "check-presentation-browser.mjs",
         "check-api-landmarks.mjs", "check-api-model-intent.mjs", "package.json", "package-lock.json",
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
                        ("model_intent_digest", "check-api-model-intent.mjs"),
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
                (expected[1] is None or any(response.get("url") == failure["url"] and response.get("status") == expected[1] and
                    response.get("intentional") is True for response in log.get("http_failures", []))),
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
           mode + ": all 491 native operation and model disclosures",
           mode + "/portal.html: sample downloads distinct from release downloads"}
    for index in range(491):
        prefix = mode + ": native contract " + str(index) + ": "
        ids |= {prefix + "keyboard expansion", prefix + "keyboard collapse",
                prefix + "actual visible point unoccluded"}
    for route in ROUTES | {"/"}:
        for phase in ("initial", "after native content"):
            prefix = mode + "/" + route + ": " + phase
            ids |= {prefix + ": sole canonical main, unique IDs and ARIA", prefix + ": viewport fits native content"}
    if mode.endswith("-js"):
        for kind in ("view link", "plain view link", "native summary"):
            ids.add(mode + ": API keyboard entry/" + kind +
                    ": ready transition retains visible unoccluded keyboard focus")
        for name in ("view link: actual Tab reaches explorer selector",
                     "plain view link: actual Shift+Tab reaches Plain selector",
                     "native summary: actual Tab reaches explorer summary after far Plain contract"):
            ids.add(mode + ": API keyboard entry/" + name)
        for kind in ("intro reentry", "model reentry"):
            ids |= {mode + ": API native reentry/" + kind + ": " + suffix for suffix in
                    ("actual prior Scalar fragment", "native close retains prior fragment", "pre-close setup preserves fragment and summary focus",
                     "actual Tab reaches explorer summary after far Plain contract", "ready transition selects only interactive reference",
                     "ready transition retains visible unoccluded keyboard focus")}
        ids |= {mode + ": API native reentry/" + name for name in
                ("initial model deep link preserves requested route", "initial model deep link: requested model owns visible focus after readiness",
                 "ready model navigation: requested model owns visible focus after readiness", "model route: actual Tab cancels previous model ownership",
                 "model route: newer host fragment cancels previous route")}
        ids |= {mode + ": API route during loading/" + suffix for suffix in
                ("new route: actual hash navigation occurs during loading",
                 "new route: requested model owns visible focus after readiness")}
        ids.add(mode + ": cold contract exact actual held response")
        ids.add(mode + ":model-focus-away: cold contract exact actual held response")
        ids |= {mode + ": API model focus movement/" + name for name in
                ("pre-ready host focus is observed", "ready preserves moved host focus")}
        for phase in ("online API lifecycle", "loaded offline API lifecycle"):
            for cycle in range(3):
                ids.add(mode + ": " + phase + "/cycle " + str(cycle) + ": Escape dismisses the API client")
    return ids


def verify_negative_controls(root, receipt):
    controls = receipt.get("negative_controls", [])
    expected = {mode + ":" + failure for mode in ("desktop-js", "mobile-js") for failure in ("404", "tamper", "script", "focus-return", "cancel", "forward-focus", "host-failure")}
    require(len(controls) == 14 and {item.get("id") for item in controls} == expected, "incomplete negative controls")
    ids = {item["id"] for item in receipt["checks"]}
    for log in controls:
        if log["id"].endswith((":forward-focus", ":host-failure")):
            verify_forward_control(root, receipt, log)
            continue
        if log["id"].endswith((":focus-return", ":cancel")):
            verify_cold_control(root, receipt, log)
            continue
        expected_response = (("/openapi.json", 404) if log["id"].endswith(":404") else
                             ("/scalar.js", 503) if log["id"].endswith(":script") else None)
        verify_requests(log, expected_response)
        require(log["id"] + ": complete Plain HTML is the initial or failed-enhancement view" in ids,
                "missing fail-closed fallback assertion")
        if not log["id"].endswith(":script"):
            require(log["id"] + ": failed explorer: ready transition retains visible unoccluded keyboard focus" in ids,
                    "missing failed-entry focus assertion")
        if log["id"].endswith(":tamper"):
            require(log.get("console") == log.get("http_failures") == [], "tamper control caused an unexpected runtime error")
        else:
            status = 404 if log["id"].endswith(":404") else 503
            failures = log.get("http_failures", [])
            require(len(failures) == 1 and failures[0].get("status") == status and failures[0].get("intentional") is True,
                    "missing intentional negative HTTP response")
            require(failures[0].get("url") == "http://127.0.0.1:8081" + expected_response[0],
                    "negative control HTTP failure differs from the expected resource")
            console = log.get("console", [])
            require(len(console) == 1 and all(re.search(r"404|503", item.get("text", "")) for item in console),
                    "unexpected negative-control console error")


def verify_forward_control(root, receipt, log):
    failed = log["id"].endswith(":host-failure")
    verify_requests(log, ("/openapi.json", 404) if failed else None)
    failures = log.get("network_failures", [])
    require(len(failures) <= int(failed) and all(failure.get("url") == "http://127.0.0.1:8081/openapi.json" for failure in failures),
            "only one exact controlled host404 abort may occur")
    if failed:
        require(log.get("http_failures") == [{"url": "http://127.0.0.1:8081/openapi.json", "status": 404, "intentional": True}],
                "host failure must have its exact controlled missing response")
        require(len(log.get("console", [])) == 1 and re.search(r"404", log["console"][0].get("text", "")),
                "host failure console must contain only its controlled missing response")
    else:
        require(log.get("console") == log.get("http_failures") == [], "cold forward must have no runtime or HTTP failures")
    url = "http://127.0.0.1:8081/openapi.json"
    require(log.get("held_contract") == {"status": 200, "digest": digest(root / "openapi.json"),
                                         "size": (root / "openapi.json").stat().st_size},
            "cold forward did not hold exact actual contract bytes")
    require(len([request for request in log["requests"] if request.get("method") == "GET" and
                request.get("url") == url]) == 1 and
            len([request for request in log["requests"] if urlsplit(request.get("url", "")).path == "/openapi.json"]) == 1,
            "cold forward must correlate one exact contract GET")
    checks = {item["id"]: item for item in receipt["checks"]}
    prefix = log["id"] + ": "
    required = {prefix + name for name in ("cold contract exact actual held response",
        "cold forward: actual Tab enters explorer selector", "cold forward: actual Tab keydown occurs during loading",
        "cold forward: actual Tab selects host contract link")}
    required.add(prefix + ("failed host explorer: ready transition retains visible unoccluded keyboard focus" if failed else
                          "cold forward: ready retains visible unoccluded owned host focus"))
    require(required <= checks.keys(), "missing cold forward observations")
    require(checks[prefix + "cold forward: actual Tab keydown occurs during loading"].get("detail") ==
            {"key": "Tab", "phase": "loading"}, "missing actual forward loading keydown")
    identity = checks[prefix + "cold forward: actual Tab selects host contract link"].get("detail")
    require(isinstance(identity, dict) and identity.get("host") is True and
            identity == {"tag": "A", "href": "openapi.json", "host": True}, "forward focus differs from actual host contract link")
    if failed:
        state_id = prefix + "cold host failure: selects only Plain and disposes explorer"
        state = checks.get(state_id, {}).get("detail")
        require(isinstance(state, dict) and state.get("plain") is True and state.get("interactive") is False and
                state.get("mounted") is False and set(state) == {"plain", "interactive", "mounted"},
                "failed host explorer did not select only the complete Plain reference")
        return  # The common entry replay validates all four owned Plain samples.
    samples = checks[prefix + "cold forward: ready retains visible unoccluded owned host focus"].get("detail")
    require(isinstance(samples, list) and len(samples) == 4, "missing forward geometry samples")
    viewport = {"width": 390, "height": 844} if log["id"].startswith("mobile") else {"width": 1280, "height": 900}
    for sample in samples:
        require(isinstance(sample, dict) and all(sample.get(key) is True for key in ("target", "visible", "unoccluded")),
                "forward focus is hidden, unowned or occluded")
        require(sample.get("viewport") == viewport, "forward viewport differs")
        verify_entry_rect(sample.get("rect", {}), viewport)


def verify_cold_control(root, receipt, log):
    verify_requests(log, ("/openapi.json", None))
    require(log.get("console") == log.get("http_failures") == [] and len(log.get("network_failures", [])) == 1,
            "cold cancellation must have only its exact contract abort")
    url = "http://127.0.0.1:8081/openapi.json"
    require(log["network_failures"][0].get("url") == url, "cold cancellation abort URL differs")
    require(log.get("held_contract") == {"status": 200, "digest": digest(root / "openapi.json"),
                                         "size": (root / "openapi.json").stat().st_size},
            "cold cancellation did not hold exact actual contract bytes")
    require(len([request for request in log["requests"] if request.get("method") == "GET" and
                request.get("url") == url]) == 1,
            "cold cancellation must correlate one exact contract GET")
    prefix = log["id"] + ": "
    required = {prefix + name for name in ("cold contract exact actual held response",
                "cold contract: actual Tab enters explorer summary",
                "cold contract: cancellation keydown occurs during loading",
                "cold contract: cancelled explorer never resurrects")}
    required.add(prefix + ("cancel explorer: ready transition retains visible unoccluded keyboard focus" if
                 log["id"].endswith(":cancel") else "cold contract: moved Plain focus remains visible and owned"))
    require(required <= {item["id"] for item in receipt["checks"]}, "missing cold cancellation observations")
    if log["id"].endswith(":focus-return"):
        sample = next(item["detail"] for item in receipt["checks"] if item["id"] == prefix +
                      "cold contract: moved Plain focus remains visible and owned")
        require(all(sample.get(key) is True for key in ("target", "visible", "unoccluded")), "moved focus is hidden or unowned")
        require(sample.get("viewport") == ({"width": 390, "height": 844} if log["id"].startswith("mobile") else
                {"width": 1280, "height": 900}), "moved focus viewport differs")
        verify_entry_rect(sample.get("rect", {}), sample["viewport"])


def verify_entry_geometry(receipt):
    for check in receipt["checks"]:
        if check["id"].endswith(": cold contract: cancellation keydown occurs during loading"):
            key = "Tab" if ":focus-return:" in check["id"] else "Enter"
            require(check.get("detail") == {"key": key, "phase": "loading"}, "missing actual loading keydown observation")
        if not check["id"].endswith(": ready transition retains visible unoccluded keyboard focus"):
            continue
        samples = check.get("detail")
        require(isinstance(samples, list) and len(samples) == 4, "missing entry geometry samples")
        for sample in samples:
            require(isinstance(sample, dict) and all(sample.get(key) is True for key in
                    ("target", "visible", "unoccluded", "headingVisible", "headingUnoccluded")),
                    "entry focus or heading is hidden, unowned or occluded")
            viewport = sample.get("viewport", {})
            require(viewport == ({"width": 390, "height": 844} if check["id"].startswith("mobile") else
                                 {"width": 1280, "height": 900}), "entry viewport differs from actual mode")
            for name in ("rect", "heading"):
                verify_entry_rect(sample.get(name, {}), viewport)


def verify_route_navigation(root, receipt):
    controls = {c["id"]: c for c in receipt["checks"]}
    for mode in ("desktop-js", "mobile-js"):
        held = controls[mode + ": cold contract exact actual held response"]
        require(held.get("detail") == {"status": 200, "digest": digest(root / "openapi.json"),
                "size": (root / "openapi.json").stat().st_size}, "new route held response differs from exact contract")
        model = "#models/oscalservicesv1CreateComponentDefinitionRequest"
        for kind, fragment in (("intro reentry", "#description/introduction"), ("model reentry", model)):
            for suffix in ("actual prior Scalar fragment", "native close retains prior fragment"):
                require(controls[mode + ": API native reentry/" + kind + ": " + suffix].get("detail") ==
                        {"fragment": fragment}, "reentry fragment differs from observed route")
            setup = controls[mode + ": API native reentry/" + kind + ": pre-close setup preserves fragment and summary focus"].get("detail", {})
            require(setup == {"fragment": fragment, "summaryFocused": True} and setup["summaryFocused"] is True, "pre-close setup did not preserve route and focus")
        require(controls[mode + ": API native reentry/initial model deep link preserves requested route"].get("detail") ==
                {"fragment": model}, "initial model route differs")
        require(controls[mode + ": API route during loading/new route: actual hash navigation occurs during loading"].get("detail") ==
                {"phase": "loading", "requested": model}, "missing actual new route loading observation")
        away = controls[mode + ": API native reentry/model route: newer host fragment cancels previous route"].get("detail")
        require(isinstance(away, list) and len(away) == 4 and all(isinstance(s, dict) and
                s == {"hash": "#unknown-api-model-route", "owned": True} and s["owned"] is True
                for s in away), "model route overwrote newer host fragment")
        yielded = controls[mode + ": API native reentry/model route: actual Tab cancels previous model ownership"].get("detail")
        require(isinstance(yielded, list) and len(yielded) == 4 and all(isinstance(s, dict) and
                s.get("owned") is True and s.get("moved") is True for s in yielded), "model route ignored moved focus")
        for prefix in (": API route during loading/new route", ": API native reentry/initial model deep link",
                       ": API native reentry/ready model navigation"):
            control = controls[mode + prefix + ": requested model owns visible focus after readiness"]
            verify_model_route_samples(mode, control.get("detail"), model)


def verify_model_focus_movement(root, receipt):
    controls = {c["id"]: c for c in receipt["checks"]}
    for mode in ("desktop-js", "mobile-js"):
        held = controls[mode + ":model-focus-away: cold contract exact actual held response"].get("detail")
        require(held == {"status": 200, "digest": digest(root / "openapi.json"),
                "size": (root / "openapi.json").stat().st_size}, "model focus movement held response differs")
        setup = controls[mode + ": API model focus movement/pre-ready host focus is observed"].get("detail", {})
        require(isinstance(setup, dict) and setup.get("phase") == "loading" and
                setup.get("requested") == "#models/oscalservicesv1CreateComponentDefinitionRequest" and
                setup.get("modelHadFocus") is True and setup.get("hostFocus") is True and setup.get("host") is True and
                setup.get("tag") == "A" and setup.get("href") == "openapi.json",
                "model focus movement was not observed during loading")
        samples = controls[mode + ": API model focus movement/ready preserves moved host focus"].get("detail")
        require(isinstance(samples, list) and len(samples) == 4, "missing model focus movement samples")
        viewport = {"width": 390, "height": 844} if mode.startswith("mobile") else {"width": 1280, "height": 900}
        for sample in samples:
            require(isinstance(sample, dict) and all(sample.get(key) is True for key in
                    ("ready", "target", "visible", "unoccluded")) and sample.get("viewport") == viewport and
                    isinstance(sample.get("hash"), str), "model focus movement did not retain visible owned host focus")
            verify_entry_rect(sample.get("rect", {}), viewport)


def verify_model_route_samples(mode, samples, model):
    require(isinstance(samples, list) and len(samples) == 4, "missing new route focus samples")
    viewport = {"width": 390, "height": 844} if mode.startswith("mobile") else {"width": 1280, "height": 900}
    for sample in samples:
        require(isinstance(sample, dict) and all(sample.get(key) is True for key in
                ("ready", "target", "visible", "unoccluded")) and sample.get("hash") == model and sample.get("viewport") == viewport,
                "new route focus is hidden, unowned or misrouted")
        verify_entry_rect(sample.get("rect", {}), viewport)


def verify_entry_rect(rect, viewport):
    require(all(type(rect.get(key)) in (int, float) and math.isfinite(rect[key]) for key in
                ("top", "bottom", "left", "right", "width", "height")), "missing finite actual entry geometry")
    require(0 <= rect["top"] < rect["bottom"] <= viewport["height"] and
            0 <= rect["left"] < rect["right"] <= viewport["width"] and
            rect["width"] > 0 and rect["height"] > 0 and
            abs(rect["right"] - rect["left"] - rect["width"]) < .01 and
            abs(rect["bottom"] - rect["top"] - rect["height"]) < .01,
            "entry geometry exceeds viewport or has inconsistent dimensions")


def verify_execution(receipt):
    execution = receipt.get("execution", {})
    require(isinstance(execution, dict), "missing browser execution observations")
    elapsed = execution.get("elapsed_ms")
    require(type(execution.get("deadline_ms")) is int and execution["deadline_ms"] == 1500000,
            "browser execution ceiling differs")
    require(type(elapsed) in (int, float) and math.isfinite(elapsed) and 0 < elapsed <= 1500000,
            "missing finite bounded browser duration")


def verify_coverage_geometry(receipt):
    for check in receipt["checks"]:
        if not check["id"].endswith(": actual visible point unoccluded"):
            continue
        detail = check.get("detail", {})
        require(isinstance(detail, dict), "missing coverage geometry observations")
        attempts = detail.get("attempts")
        require(isinstance(attempts, list) and 1 <= len(attempts) <= 3, "missing bounded coverage samples")
        viewport = {"width": 390, "height": 844} if check["id"].startswith("mobile") else {"width": 1280, "height": 900}
        verify_coverage_sample(detail, viewport)
        for index, sample in enumerate(attempts):
            verify_coverage_sample(sample, viewport)
            require(sample.get("unobscured") is (index == len(attempts) - 1), "invalid coverage attempt outcome")
            require(index == 0 or sample["time"] >= attempts[index - 1]["time"], "coverage observation time reversed")
        require({key: value for key, value in detail.items() if key != "attempts"} == attempts[-1],
                "coverage summary differs from its final actual sample")


def verify_coverage_sample(sample, viewport):
    require(isinstance(sample, dict) and sample.get("viewport") == viewport, "coverage viewport differs")
    rect, point = sample.get("rect", {}), sample.get("point", {})
    require(isinstance(rect, dict) and isinstance(point, dict), "missing coverage rectangle or point")
    require(all(type(rect.get(key)) in (int, float) and math.isfinite(rect[key]) for key in
                ("x", "y", "top", "bottom", "left", "right", "width", "height")), "nonfinite coverage geometry")
    require(rect["width"] >= 0 and rect["height"] >= 0 and
            abs(rect["right"] - rect["left"] - rect["width"]) < .01 and
            abs(rect["bottom"] - rect["top"] - rect["height"]) < .01 and
            rect["x"] == rect["left"] and rect["y"] == rect["top"], "inconsistent coverage dimensions")
    left, right = max(rect["left"], 0), min(rect["right"], viewport["width"])
    top, bottom = max(rect["top"], 0), min(rect["bottom"], viewport["height"])
    require(all(type(point.get(key)) in (int, float) and math.isfinite(point[key]) for key in ("x", "y")) and
            abs(point["x"] - (left + right) / 2) < .01 and abs(point["y"] - (top + bottom) / 2) < .01,
            "coverage clipped point differs")
    require(type(sample.get("unobscured")) is bool and
            (not sample["unobscured"] or (right > left and bottom > top and sample.get("hit") not in (None, "", "undefined#undefined"))),
            "offscreen or occluded coverage cannot pass")
    require(all(type(sample.get(key)) in (int, float) and math.isfinite(sample[key]) and sample[key] >= 0
                for key in ("time", "scrollY", "documentHeight")) and sample["documentHeight"] > 0 and
            all(isinstance(sample.get(key), str) for key in ("focus", "hash", "hit")),
            "missing actual coverage context")


def verify_checks(root, receipt):
    require(receipt.get("schema_version") == 1 and receipt.get("status") == "passed", "missing successful raw receipt")
    checks = receipt.get("checks")
    require(isinstance(checks, list) and len(checks) >= 100, "missing actual checks")
    ids = [item.get("id") for item in checks if isinstance(item, dict)]
    require(len(ids) == len(checks) == len(set(ids)) and all(isinstance(i, str) and i for i in ids),
            "missing or duplicate check identity")
    require(all(item.get("passed") is True for item in checks), "failed, missing or nonboolean check")
    require(receipt.get("failures") == [], "raw failures cannot admit")
    verify_execution(receipt)
    inventory = receipt.get("inventory_observations", {})
    verify_requests(inventory)
    require(inventory.get("console") == inventory.get("http_failures") == [],
            "unexpected native inventory console or HTTP failure")
    require("inventory: read-only same-origin resources" in ids and receipt.get("proxy_denials") == [],
            "missing or failed inventory and proxy boundary observations")
    modes = receipt.get("modes", [])
    require(len(modes) == 4 and {mode.get("id") for mode in modes} == MODES, "incomplete browser modes")
    for mode in modes:
        require(mode.get("javaScriptEnabled") is mode["id"].endswith("-js"), "mode JavaScript setting differs")
        verify_requests(mode)
        require(mode.get("console") == mode.get("http_failures") == [], "unexpected console or HTTP failures")
        require(mandatory_checks(mode["id"]) <= set(ids), "missing mandatory browser assertions")
        verify_coverage(root, receipt, mode)
    verify_negative_controls(root, receipt)
    verify_entry_geometry(receipt)
    verify_route_navigation(root, receipt)
    verify_model_focus_movement(root, receipt)
    verify_coverage_geometry(receipt)


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
