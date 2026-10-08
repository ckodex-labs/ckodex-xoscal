"""Independent reentry and explicit-route receipt refusal controls."""
import copy
import json
from pathlib import Path
import tempfile
import unittest
import presentation_browser_verify as browser
from test_presentation_browser_verify import make_fixture, write_json

class RouteReceiptTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.receipt = make_fixture(self.root)

    def save(self, receipt):
        write_json(self.root / browser.DIRECTORY / browser.FILES[0], receipt)

    def test_reentry_and_new_route_controls_are_mandatory(self):
        for fragment in ("intro reentry", "model reentry", "initial model deep link", "new route:", "ready model navigation", "model route:"):
            changed = copy.deepcopy(self.receipt)
            changed["checks"] = [c for c in changed["checks"] if fragment not in c["id"]]
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "mandatory"):
                browser.verify(self.root)

    def test_new_route_response_and_geometry_refuse_false_evidence(self):
        for field, value in (("target", False), ("hash", "#description/introduction"),
                             ("viewport", {"width": 1, "height": 1})):
            changed = copy.deepcopy(self.receipt)
            control = next(c for c in changed["checks"] if c["id"].endswith("new route: requested model owns visible focus after readiness"))
            control["detail"][0][field] = value
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "new route"):
                browser.verify(self.root)
        changed = copy.deepcopy(self.receipt)
        next(c for c in changed["checks"] if c["id"] == "desktop-js: cold contract exact actual held response")["detail"]["digest"] = "sha256:" + "0" * 64
        self.save(changed)
        with self.assertRaisesRegex(ValueError, "held response"):
            browser.verify(self.root)

    def test_reentry_fragment_loading_and_deep_link_evidence_is_exact(self):
        suffixes = ("intro reentry: actual prior Scalar fragment", "model reentry: native close retains prior fragment",
                    "initial model deep link preserves requested route", "new route: actual hash navigation occurs during loading", "model reentry: pre-close setup preserves fragment and summary focus")
        for suffix in suffixes:
            changed = copy.deepcopy(self.receipt)
            next(c for c in changed["checks"] if c["id"].endswith(suffix))["detail"] = {}
            self.save(changed)
            with self.assertRaises(ValueError):
                browser.verify(self.root)

    def test_pre_close_focus_requires_actual_boolean(self):
        for value in (False, 1, "true"):
            changed = copy.deepcopy(self.receipt)
            next(c for c in changed["checks"] if c["id"].endswith("model reentry: pre-close setup preserves fragment and summary focus"))["detail"]["summaryFocused"] = value
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "pre-close"):
                browser.verify(self.root)

    def test_model_focus_geometry_requires_strict_booleans_and_finite_rectangles(self):
        for mutation in (lambda s: s.update(ready=1), lambda s: s.update(ready=False), lambda s: s.update(target=1), lambda s: s.update(visible=False),
                         lambda s: s["rect"].update(top=float("nan")), lambda s: s["rect"].update(bottom=10000)):
            changed = copy.deepcopy(self.receipt)
            control = next(c for c in changed["checks"] if c["id"].endswith("initial model deep link: requested model owns visible focus after readiness"))
            mutation(control["detail"][0])
            self.save(changed)
            with self.assertRaises(ValueError):
                browser.verify(self.root)

    def test_model_route_yields_only_with_actual_boolean_observations(self):
        for value in (False, 1, "true"):
            changed = copy.deepcopy(self.receipt)
            next(c for c in changed["checks"] if c["id"].endswith("model route: actual Tab cancels previous model ownership"))["detail"][0]["owned"] = value
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "moved focus"):
                browser.verify(self.root)

    def test_model_route_must_not_overwrite_newer_host_fragment(self):
        for field, value in (("hash", "#models/old"), ("owned", False), ("owned", 1), ("owned", "true")):
            changed = copy.deepcopy(self.receipt)
            next(c for c in changed["checks"] if c["id"].endswith("model route: newer host fragment cancels previous route"))["detail"][2][field] = value
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "newer host"):
                browser.verify(self.root)

    def test_model_focus_movement_controls_are_mandatory_and_exact(self):
        for suffix in ("model-focus-away: cold contract exact actual held response", "pre-ready host focus is observed", "ready preserves moved host focus"):
            changed = copy.deepcopy(self.receipt)
            changed["checks"] = [c for c in changed["checks"] if not c["id"].endswith(suffix)]
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "mandatory"):
                browser.verify(self.root)
        for field, value in (("phase", "ready"), ("modelHadFocus", False), ("modelHadFocus", 1), ("hostFocus", "true"), ("host", False), ("host", 1), ("href", "wrong"), ("tag", "BUTTON")):
            changed = copy.deepcopy(self.receipt)
            next(c for c in changed["checks"] if c["id"].endswith("pre-ready host focus is observed"))["detail"][field] = value
            self.save(changed)
            with self.assertRaisesRegex(ValueError, "model focus movement"):
                browser.verify(self.root)

    def test_model_focus_movement_samples_refuse_hidden_or_stolen_focus(self):
        for mutation in (lambda s: s.update(target=False), lambda s: s.update(ready=1), lambda s: s.update(unoccluded="true"),
                         lambda s: s["rect"].update(top=float("nan")), lambda s: s["rect"].update(bottom=10000)):
            changed = copy.deepcopy(self.receipt)
            mutation(next(c for c in changed["checks"] if c["id"].endswith("ready preserves moved host focus"))["detail"][2])
            self.save(changed)
            with self.assertRaises(ValueError):
                browser.verify(self.root)
        changed = copy.deepcopy(self.receipt)
        next(c for c in changed["checks"] if c["id"].endswith("model-focus-away: cold contract exact actual held response"))["detail"]["digest"] = "sha256:" + "0" * 64
        self.save(changed)
        with self.assertRaisesRegex(ValueError, "held response"):
            browser.verify(self.root)
