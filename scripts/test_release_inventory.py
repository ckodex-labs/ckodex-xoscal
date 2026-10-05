import copy
import tempfile
import unittest
from pathlib import Path

from release_inventory import create, required_payload, validate, write


class InventoryContract(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.root.joinpath("sdk").mkdir()
        self.root.joinpath("sdk/go.zip").write_bytes(b"candidate bytes")
        self.manifest = create(self.root, "v1.2.3", "a" * 40)

    def test_actual_bytes(self):
        self.assertEqual(len(validate(self.root, self.manifest)), 1)

    def test_tamper(self):
        self.root.joinpath("sdk/go.zip").write_bytes(b"tampered")
        with self.assertRaisesRegex(ValueError, "integrity"):
            validate(self.root, self.manifest)

    def test_missing_and_uncovered(self):
        self.root.joinpath("sdk/python.zip").write_bytes(b"uncovered")
        with self.assertRaisesRegex(ValueError, "uncovered"):
            validate(self.root, self.manifest)
        self.root.joinpath("sdk/go.zip").unlink()
        with self.assertRaises(ValueError):
            validate(self.root, self.manifest)

    def test_duplicate_and_traversal(self):
        for path in ("../escape", "/absolute", "sdk/../escape", "sdk\\escape"):
            value = copy.deepcopy(self.manifest)
            value["artifacts"][0]["path"] = path
            with self.assertRaises(ValueError):
                validate(self.root, value)
        self.manifest["artifacts"] *= 2
        with self.assertRaisesRegex(ValueError, "duplicate"):
            validate(self.root, self.manifest)

    def test_no_self_reference_and_only_detached_envelope_exempt(self):
        write(self.root / "payload-manifest.json", self.manifest)
        final = create(self.root, "v1.2.3", "a" * 40, final=True)
        write(self.root / "release-manifest.json", final)
        write(self.root / "proofs/release-manifest.sigstore.json", {})
        validate(self.root, final, final=True)
        write(self.root / "proofs/invented-proof.json", {})
        with self.assertRaisesRegex(ValueError, "uncovered"):
            validate(self.root, final, final=True)

    def test_frontend_runtime_is_payload_covered_with_admission_references(self):
        self.root.joinpath("scalar.js").write_bytes(b"fixture browser bytes")
        self.root.joinpath("scalar.js.map").write_bytes(b"fixture source map")
        for filename in ("analysis-gate.json", "frontend-inventory.json", "sbom.cyclonedx.json"):
            path = self.root / "evidence/frontend" / filename
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"fixture evidence")
        proof = self.root / "evidence/frontend-sbom/analysis-gate.json"
        proof.parent.mkdir(parents=True, exist_ok=True);proof.write_bytes(b"fixture SBOM evidence")
        manifest = create(self.root, "v1.2.3", "a" * 40)
        records = {item["path"]: item for item in validate(self.root, manifest)}
        for path in ("scalar.js", "scalar.js.map"):
            self.assertEqual(records[path]["kind"], "frontend-runtime")
            self.assertEqual(records[path]["producer"], "FrontendAssets")
            self.assertEqual(len(records[path]["evidence_refs"]), 4)
        self.assertEqual(records["evidence/frontend/analysis-gate.json"]["producer"], "FrontendAssets")

    def test_frontend_served_byte_mutation_invalidates_payload_manifest(self):
        self.root.joinpath("scalar.js").write_bytes(b"original served bytes")
        manifest = create(self.root, "v1.2.3", "a" * 40)
        self.root.joinpath("scalar.js").write_bytes(b"mutated served bytes")
        with self.assertRaisesRegex(ValueError, "integrity"):
            validate(self.root, manifest)

    def test_missing_mandatory_outputs(self):
        with self.assertRaisesRegex(ValueError, "missing required"):
            required_payload(self.root, self.manifest)

    def test_metadata_tampering_rejected(self):
        for key, value in (("kind", "cli-archive"), ("producer", "invented"),
                           ("name", "renamed.zip"), ("platform", {"os": "darwin", "arch": "amd64"}),
                           ("release_asset_name", "release-site.tar.gz"), ("evidence_refs", ["invented"])):
            document = copy.deepcopy(self.manifest)
            document["artifacts"][0][key] = value
            with self.assertRaises(ValueError):
                validate(self.root, document)

    def test_attachment_collisions_rejected_before_signing(self):
        directory = self.root / "sdk/packages/go"
        directory.mkdir(parents=True)
        (directory / "go.zip").write_bytes(b"collision")
        with self.assertRaisesRegex(ValueError, "attachment"):
            create(self.root, "v1.2.3", "a" * 40)

    def test_symlinks_rejected(self):
        self.root.joinpath("sdk/link").symlink_to("go.zip")
        with self.assertRaisesRegex(ValueError, "symlinks"):
            validate(self.root, self.manifest)


if __name__ == "__main__":
    unittest.main()
