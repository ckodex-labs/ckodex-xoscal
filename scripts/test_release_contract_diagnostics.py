"""Failed verifier diagnostics only; no successful hosted proof is fabricated."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("release_contract_diagnostics", Path(__file__).with_name("release-contract.py"))
CONTRACT = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CONTRACT)


class VerifierDiagnostics(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.manifest = self.root / "payload-manifest.json"
        self.manifest.write_text(json.dumps({"release_tag": "v1.2.3", "source_revision": "a" * 40}))
        self.bundle = self.root / "invalid.sigstore.json"
        self.bundle.write_text("{}\n")

    def test_rejection_preserves_stdout_stderr_exit_and_verification_flags(self):
        failure = subprocess.CalledProcessError(1, ["gh"], output="invalid bundle schema", stderr="no verification material found")
        with patch.object(CONTRACT.subprocess, "run", side_effect=failure) as run:
            with self.assertRaisesRegex(ValueError, "gh attestation verify failed") as raised:
                CONTRACT.verify_attestation(self.root, self.manifest, self.bundle, "v1.2.3", "a" * 40)
        message = str(raised.exception)
        self.assertIn("exit 1", message)
        self.assertIn("stdout:\ninvalid bundle schema", message)
        self.assertIn("stderr:\nno verification material found", message)
        command = ["gh", "attestation", "verify", str(self.manifest)] + CONTRACT.verification_flags(self.bundle, "v1.2.3", "a" * 40)
        run.assert_called_once_with(command, check=True, capture_output=True, text=True)
        self.assertFalse((self.root / "proofs/payload-verification.json").exists())

    def test_failure_diagnostics_redact_credentials_without_hiding_auth_error(self):
        secret = "synthetic-environment-token"
        failure = subprocess.CalledProcessError(4, ["gh"], output="Authorization: Bearer arbitrary-secret", stderr=f"authentication failed: {secret}; ghp_syntheticgithubtoken")
        with patch.dict(CONTRACT.os.environ, {"GH_TOKEN": secret}), patch.object(CONTRACT.subprocess, "run", side_effect=failure):
            with self.assertRaises(ValueError) as raised:
                CONTRACT.verify_attestation(self.root, self.manifest, self.bundle, "v1.2.3", "a" * 40)
        message = str(raised.exception)
        for token in (secret, "arbitrary-secret", "ghp_syntheticgithubtoken"):
            self.assertNotIn(token, message)
        self.assertIn("authentication failed", message)
        self.assertIn("exit 4", message)
        self.assertIn("[REDACTED]", message)

    def test_identity_mismatch_still_blocks_before_verifier(self):
        with patch.object(CONTRACT.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "independently expected release identity"):
                CONTRACT.verify_attestation(self.root, self.manifest, self.bundle, "v1.2.4", "a" * 40)
        run.assert_not_called()


if __name__ == "__main__":
    unittest.main()
