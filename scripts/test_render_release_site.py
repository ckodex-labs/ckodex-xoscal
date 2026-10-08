"""Exercise real renderer fixture bytes. No test supplies successful hosted proof."""
import copy
import importlib.util
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

from release_inventory import create, digest, write

SCRIPTS = Path(__file__).resolve().parent
REPO = SCRIPTS.parent
spec = importlib.util.spec_from_file_location("site_smoke", SCRIPTS / "site-smoke.py")
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)
presentation_spec = importlib.util.spec_from_file_location("release_presentation", SCRIPTS / "check-release-presentation.py")
presentation = importlib.util.module_from_spec(presentation_spec)
presentation_spec.loader.exec_module(presentation)


class RenderReleaseSite(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "site"
        shutil.copytree(REPO / "site", self.root)
        (self.root / "sbom-assessment-results.json").unlink(missing_ok=True)
        # These are real file/ZIP fixtures for the static contract; language
        # installability and the pinned production Scalar bundle have their own gates.
        frontend_assets = {
            "scalar.js": "/* static contract fixture */\n//# sourceMappingURL=scalar.js.map\n",
            "scalar.js.map": '{"version":3,"file":"scalar.js","sources":[],"names":[],"mappings":""}\n',
            "scalar.css": "/* static contract fixture */\n.fixture {}\n/*# sourceMappingURL=scalar.css.map */\n",
            "scalar.css.map": '{"version":3,"file":"scalar.css","sources":[],"names":[],"mappings":""}\n',
            "THIRD-PARTY-NOTICES.txt": "Synthetic static contract fixture; no production publisher claims.\n",
        }
        for name, content in frontend_assets.items():
            (self.root / name).write_text(content)
        (self.root / "scripts").mkdir(exist_ok=True)
        shutil.copyfile(SCRIPTS / "install.sh", self.root / "scripts/install.sh")
        (self.root / "sdk").mkdir()
        for language in smoke.LANGUAGES:
            path = self.root / f"sdk/{language}.zip"
            with zipfile.ZipFile(path, "w") as archive:
                archive.writestr("README.md", f"{language} static smoke fixture\n")
            Path(str(path) + ".sha256").write_text(digest(path) + "\n")
        for name in [f"fixture-{i:02}" for i in range(34)] + ["cccs-medium-cloud-pbmm"]:
            directory = self.root / "frameworks" / name
            directory.mkdir(parents=True)
            path = directory / "catalog.json"
            write(path, {"catalog": {"uuid": "fixture", "metadata": {"title": name}}})
            Path(str(path) + ".sha256").write_text(digest(path) + "\n")
        profile = self.root / "frameworks/cccs-medium-cloud-pbmm/profile.json"
        write(profile, {"profile": {"uuid": "fixture", "imports": []}})
        Path(str(profile) + ".sha256").write_text(digest(profile) + "\n")
        self.manifest = self.root / "payload-manifest.json"
        self.reset_manifest()

    def reset_manifest(self):
        write(self.manifest, create(self.root, "v1.2.3", "a" * 40))

    def render(self, *arguments, env=None):
        result = subprocess.run([sys.executable, str(SCRIPTS / "render-release-site.py"),
                                 "--root", str(self.root), "--manifest", str(self.manifest),
                                 *arguments], capture_output=True, text=True, env=env)
        return result

    def pages(self):
        return {path.name: smoke.Page(path.read_text()) for path in self.root.glob("*.html")}

    def assert_refused_without_mutation(self, *arguments, env=None):
        before = {name: (self.root / name).read_bytes() if (self.root / name).exists() else None for name in
                  ("downloads.html", "transparency.html", "sbom-validation.html", "downloads.json", "provenance.json")}
        result = self.render(*arguments, env=env)
        self.assertNotEqual(result.returncode, 0, result.stdout)
        for name, content in before.items():
            if content is None:
                self.assertFalse((self.root / name).exists())
            else:
                self.assertEqual((self.root / name).read_bytes(), content)
        return result

    def test_removes_legacy_release_inventory_after_admission(self):
        legacy = self.root / "release-assets.json"
        legacy.write_text('[{"url":"https://example.invalid/other-release.zip"}]')
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(legacy.exists())
        index = json.loads((self.root / "downloads.json").read_text())
        self.assertEqual(index["verification"], "incomplete")
        self.assertFalse(any(item["eligible"] for item in index["artifacts"]))

    def test_canonical_pages_keep_every_prebuilt_metadata_record_without_duplicate_download_actions(self):
        self.assertEqual(self.render().returncode, 0)
        presentation.check(self.root)
        pages = self.pages()
        self.assertEqual(pages["transparency.html"].downloads, [])
        self.assertIn("Downloads", "".join(pages["downloads.html"].main_text))
        self.assertIn("Supply-chain Transparency", "".join(pages["transparency.html"].main_text))

    def test_inventory_disclosure_rejects_lost_or_duplicate_rows_metadata_and_actions(self):
        self.assertEqual(self.render().returncode, 0)
        path = self.root / "transparency.html"
        original = path.read_text()
        row = re.search(r'<tr id="inventory-artifact-[^>]*>.*?</tr>', original, re.S).group()
        item = json.loads(self.manifest.read_text())["artifacts"][0]
        mutations = {
            "missing row": original.replace(row, "", 1),
            "duplicate row": original.replace(row, row + row, 1),
            "missing digest": original.replace(item["digest"], "omitted", 1),
            "duplicate title": original.replace("<h1>Supply-chain Transparency</h1>", "<h1>Downloads</h1>"),
            "duplicate action": original.replace("</main>", '<a download href="sdk/go.zip">Duplicate download</a></main>'),
            "outside main": original.replace(row, "", 1).replace("</main>", "</main>" + row),
        }
        for name, text in mutations.items():
            with self.subTest(mutation=name):
                path.write_text(text)
                with self.assertRaises(ValueError):
                    presentation.check(self.root)
        path.write_text(original)

    def test_raw_sbom_remains_prebuilt_inside_native_disclosure(self):
        document = {"assessment-results": {"results": [{"findings": [], "observations": []}]}}
        write(self.root / "sbom-assessment-results.json", document)
        self.reset_manifest()
        self.assertEqual(self.render().returncode, 0)
        text = (self.root / "sbom-validation.html").read_text()
        self.assertRegex(text, r'<details><summary>Inspect the complete raw assessment</summary><pre')
        self.assertIn("assessment-results", text)
        self.assertIn('href="sbom-assessment-results.json"', text)

    def test_unsigned_generated_no_js_inventory_and_http_surface(self):
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        index = json.loads((self.root / "downloads.json").read_text())
        self.assertEqual(index["verification"], "incomplete")
        self.assertTrue(all(item["eligible"] is False and item["evidence_refs"] == [] for item in index["artifacts"]))
        page = self.pages()["downloads.html"]
        self.assertEqual(page.downloads, [])
        text = "".join(page.main_text)
        self.assertIn("Unsigned local candidate", text)
        self.assertIn("frameworks/cccs-medium-cloud-pbmm/profile.json", text)
        for language in smoke.LANGUAGES:
            self.assertIn(f"sdk/{language}.zip", text)
            self.assertIn(f"sdk/{language}.zip.sha256", text)
        self.assertIn("SHA-256 sidecars", text)
        self.assertIn("frameworks/cccs-medium-cloud-pbmm/profile.json.sha256", text)
        self.assertIn("frameworks/cccs-medium-cloud-pbmm/catalog.json.sha256", text)
        self.assertIn("Download unavailable", text)
        smoke.check_openapi(self.root)
        smoke.check_generated(self.root, self.pages())
        result = subprocess.run([sys.executable, str(SCRIPTS / "site-smoke.py"),
                                 "--root", str(self.root), "--generated"], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("96 API operations", result.stdout)

    def test_generated_http_surface_refuses_missing_scalar_source_map(self):
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        (self.root / "scalar.js.map").unlink()
        result = subprocess.run([sys.executable, str(SCRIPTS / "site-smoke.py"),
                                 "--root", str(self.root), "--generated"], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("site-smoke: FAIL", result.stdout)
        self.assertIn("HTTP Error 404", result.stdout)
        self.assertRegex(result.stderr, r'"GET /scalar\.js\.map HTTP/1\.[01]" 404')

    def empty_log_fixture(self, status=b"0\n"):
        log = self.root / "evidence/frontend/build.log"
        log.parent.mkdir(parents=True, exist_ok=True)
        log.write_bytes(b"")
        log.with_suffix(".status").write_bytes(status)
        self.reset_manifest()
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        page = self.root / "transparency.html"
        page.write_text(page.read_text().replace("</main>",
                        '<p>Static HTTP fixture: <a href="evidence/frontend/build.log">empty tool log</a></p></main>'))
        return log

    def run_http_smoke(self):
        return subprocess.run([sys.executable, str(SCRIPTS / "site-smoke.py"),
                               "--root", str(self.root), "--generated"], capture_output=True, text=True)

    def test_http_accepts_exact_inventoried_empty_success_log(self):
        self.empty_log_fixture()
        result = self.run_http_smoke()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertRegex(result.stderr, r'"GET /evidence/frontend/build\.log HTTP/1\.[01]" 200')

    def test_http_refuses_empty_log_with_failed_missing_or_changed_status(self):
        for mutation in ("failed", "missing", "changed"):
            with self.subTest(mutation=mutation):
                log = self.empty_log_fixture(status=b"1\n" if mutation == "failed" else b"0\n")
                if mutation == "missing":
                    log.with_suffix(".status").unlink()
                elif mutation == "changed":
                    log.with_suffix(".status").write_bytes(b"00\n")
                result = self.run_http_smoke()
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_http_refuses_empty_log_not_declared_in_inventory(self):
        self.empty_log_fixture()
        document = json.loads(self.manifest.read_text())
        document["artifacts"] = [a for a in document["artifacts"] if a["path"] != "evidence/frontend/build.log"]
        write(self.manifest, document)
        result = self.run_http_smoke()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("HTTP 200 or empty body", result.stdout)

    def test_http_still_refuses_empty_runtime_asset_even_when_inventoried(self):
        self.root.joinpath("scalar.js").write_bytes(b"")
        self.reset_manifest()
        self.assertEqual(self.render().returncode, 0)
        result = self.run_http_smoke()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("scalar.js: HTTP 200 or empty body", result.stdout)

    def test_generated_smoke_rechecks_complete_final_tree(self):
        self.assertEqual(self.render().returncode, 0)
        marker = self.root / "evidence/presentation.ok"
        marker.parent.mkdir(parents=True, exist_ok=True)
        marker.write_bytes(b"presentation-analysis-ok\n")
        write(self.root / "release-manifest.json", create(self.root, "v1.2.3", "a" * 40, final=True))
        result = self.run_http_smoke()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_internal_analysis_checksum_is_inventoried_without_download_card(self):
        internal = self.root / "evidence/internal/subject"
        internal.parent.mkdir(parents=True)
        internal.write_bytes(b"static analysis subject fixture\n")
        checksum = Path(str(internal) + ".sha256")
        checksum.write_text(digest(internal) + "\n")
        self.reset_manifest()
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        item = next(item for item in json.loads((self.root / "downloads.json").read_text())["artifacts"]
                    if item["path"] == "evidence/internal/subject.sha256")
        self.assertEqual(item["kind"], "checksum")
        self.assertNotIn("release_asset_name", item)
        self.assertIs(item["eligible"], False)
        page = self.pages()["downloads.html"]
        self.assertNotIn("evidence/internal/subject.sha256", "".join(page.main_text))
        self.assertIn("sdk/go.zip.sha256", "".join(page.main_text))
        self.assertEqual(page.downloads, [])
        smoke.check_generated(self.root, self.pages())

    def test_unsigned_actual_evidence_refs_retained_without_eligibility(self):
        # Static reference fixtures only; these JSON files do not assert actual
        # successful analysis, consumption or hosted signature verification.
        for path in ("sdk/smoke/go.json", "evidence/sdk-go/analysis-gate.json",
                     "evidence/sdk-go-sbom/analysis-gate.json"):
            write(self.root / path, {"fixture": True, "hosted_proof": False})
        (self.root / "sdk/sdk-version.txt").write_text("1.2.3\n")
        self.reset_manifest()
        original = next(item for item in json.loads(self.manifest.read_text())["artifacts"]
                        if item["path"] == "sdk/go.zip")
        self.assertTrue(original["evidence_refs"], "SDK producer must connect its existing evidence")
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        index = json.loads((self.root / "downloads.json").read_text())
        item = next(item for item in index["artifacts"] if item["path"] == "sdk/go.zip")
        self.assertEqual(item["evidence_refs"], original["evidence_refs"])
        self.assertIs(item["eligible"], False)
        for name in ("downloads.html", "transparency.html"):
            page = self.pages()[name]
            self.assertEqual(page.downloads, [])
            self.assertIn("OSCAL touchpoint: " + original["oscal_touchpoint"], "".join(page.main_text))
            self.assertTrue(set(original["evidence_refs"]).issubset(set(page.links)))
            self.assertNotIn("proofs/payload.sigstore.json", page.links)
            self.assertNotIn("proofs/payload-verification.json", page.links)
        smoke.check_generated(self.root, self.pages())
        for references in ([], ["https://attacker.invalid/proof"], ["sdk/ts.zip"],
                           original["evidence_refs"] + ["payload-manifest.json", "proofs/payload.sigstore.json"]):
            with self.subTest(references=references):
                item["evidence_refs"] = references
                write(self.root / "downloads.json", index)
                with self.assertRaisesRegex(ValueError, "evidence references differ"):
                    smoke.check_generated(self.root, self.pages())
        # Claiming verified state still cannot replace subject analysis with
        # proof-only links. No successful hosted verifier is substituted.
        originals = {entry["path"]: entry for entry in json.loads(self.manifest.read_text())["artifacts"]}
        index["verification"] = "verified"
        for entry in index["artifacts"]:
            entry["eligible"] = True
            entry["evidence_refs"] = originals[entry["path"]]["evidence_refs"] + ["payload-manifest.json", "proofs/payload.sigstore.json"]
        item["evidence_refs"] = ["payload-manifest.json", "proofs/payload.sigstore.json"]
        write(self.root / "downloads.json", index)
        with self.assertRaisesRegex(ValueError, "evidence references differ"):
            smoke.check_generated(self.root, self.pages())
        item["evidence_refs"] = original["evidence_refs"] + ["payload-manifest.json", "proofs/payload.sigstore.json"]
        write(self.root / "downloads.json", index)
        with self.assertRaises(FileNotFoundError):
            smoke.check_generated(self.root, self.pages())

    def test_noncanonical_manifest_evidence_refs_refused_before_rendering(self):
        document = json.loads(self.manifest.read_text())
        item = next(item for item in document["artifacts"] if item["path"] == "sdk/go.zip")
        item["evidence_refs"] = ["sdk/ts.zip"]
        write(self.manifest, document)
        self.assert_refused_without_mutation()

    def test_tampered_producer_and_name_refused_before_rendering(self):
        for field, injection in (("name", '\"><img src=x onerror=alert(1)>'),
                                 ("producer", '<script>alert("producer")</script>')):
            self.reset_manifest()
            document = json.loads(self.manifest.read_text())
            item = next(item for item in document["artifacts"] if item["path"] == "sdk/go.zip")
            item[field] = injection
            write(self.manifest, document)
            self.assert_refused_without_mutation()

    def test_missing_manifest_refused(self):
        self.manifest.unlink()
        self.assert_refused_without_mutation()

    def test_manifest_wrong_digest_refused(self):
        document = json.loads(self.manifest.read_text())
        document["artifacts"][0]["digest"] = "sha256:" + "0" * 64
        write(self.manifest, document)
        self.assert_refused_without_mutation()

    def test_tampered_payload_refused(self):
        (self.root / "sdk/go.zip").write_bytes(b"tampered exact bytes")
        self.assert_refused_without_mutation()

    def test_uncovered_payload_refused(self):
        (self.root / "sdk/uncovered.txt").write_text("uncovered")
        self.assert_refused_without_mutation()

    def test_missing_verifier_or_bundle_cannot_enable_links(self):
        env = dict(os.environ, PATH="")
        self.assert_refused_without_mutation("--bundle", str(self.root / "missing.sigstore.json"),
                                             "--tag", "v1.2.3", "--revision", "a" * 40, env=env)

    def test_verifier_rejection_cannot_enable_links(self):
        # A failure-only executable makes this deterministic/offline. It never
        # returns a claimed attestation; no hosted crypto success is simulated.
        verifier = Path(self.temp.name) / "bin"
        verifier.mkdir()
        command = verifier / "gh"
        command.write_text("#!/bin/sh\nexit 42\n")
        command.chmod(0o700)
        bundle = self.root / "proofs/payload.sigstore.json"
        write(bundle, {"invalid": "not a signature"})
        env = dict(os.environ, PATH=str(verifier))
        result = self.assert_refused_without_mutation("--bundle", str(bundle), "--tag", "v1.2.3",
                                                      "--revision", "a" * 40, env=env)
        self.assertIn("42", result.stderr)

    def test_independent_release_identity_mismatch_refused(self):
        bundle = self.root / "proofs/payload.sigstore.json"
        write(bundle, {})
        result = self.assert_refused_without_mutation("--bundle", str(bundle), "--tag", "v9.9.9", "--revision", "a" * 40)
        self.assertIn("independently expected release identity", result.stderr)

    def test_bundle_without_independent_identity_refused(self):
        bundle = self.root / "proofs/payload.sigstore.json"
        write(bundle, {})
        self.assert_refused_without_mutation("--bundle", str(bundle))

    def test_claimed_json_proof_is_not_a_signature(self):
        write(self.root / "proofs/payload-verification.json", {"verified": True, "results": ["fabricated"]})
        document = json.loads(self.manifest.read_text())
        for item in document["artifacts"]:
            item["eligible"] = True
        write(self.manifest, document)
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.pages()["downloads.html"].downloads, [])
        self.assertFalse(any(item["eligible"] for item in json.loads((self.root / "downloads.json").read_text())["artifacts"]))

    def test_sbom_missing_and_actual_counts_are_honest(self):
        self.assertEqual(self.render().returncode, 0)
        self.assertIn("Incomplete: no SBOM assessment supplied", (self.root / "sbom-validation.html").read_text())
        write(self.root / "sbom-assessment-results.json", {"assessment-results": {"results": [{
            "findings": [{"target": {"status": {"state": "satisfied"}}},
                         {"target": {"status": {"state": "not-satisfied"}}}], "observations": [{}]}]}})
        self.reset_manifest()
        result = self.render()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Observed findings: 2; satisfied: 1; other states: 1; observations: 1.",
                      (self.root / "sbom-validation.html").read_text())

    def test_generated_smoke_rejects_forged_verification_boolean(self):
        self.assertEqual(self.render().returncode, 0)
        index = json.loads((self.root / "downloads.json").read_text())
        index["verification"] = "verified"
        for item in index["artifacts"]:
            item["eligible"] = True
            item["evidence_refs"] = ["payload-manifest.json", "proofs/payload.sigstore.json"]
        write(self.root / "downloads.json", index)
        with self.assertRaises(FileNotFoundError):
            smoke.check_generated(self.root, self.pages())

    def test_generated_smoke_rejects_bad_sidecar_even_when_inventoried(self):
        (self.root / "sdk/go.zip.sha256").write_text("sha256:" + "0" * 64 + "\n")
        self.reset_manifest()
        self.assertEqual(self.render().returncode, 0)
        with self.assertRaisesRegex(ValueError, "sidecar mismatch"):
            smoke.check_generated(self.root, self.pages())

    def test_openapi_empty_missing_operation_and_bad_reference_refused(self):
        original = json.loads((self.root / "openapi.json").read_text())
        for mutation in ("empty", "missing", "reference"):
            document = copy.deepcopy(original)
            if mutation == "empty":
                document["paths"] = {}
            elif mutation == "missing":
                del document["paths"]["/v1/search/semantic"]["get"]
            else:
                document["components"]["schemas"]["broken"] = {"$ref": "#/components/schemas/missing"}
            write(self.root / "openapi.json", document)
            with self.assertRaises((ValueError, KeyError)):
                smoke.check_openapi(self.root)


if __name__ == "__main__":
    unittest.main()
