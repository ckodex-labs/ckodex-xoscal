#!/usr/bin/env python3
"""Installer control-flow tests. The gh fixture does not provide hosted cryptographic proof."""
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tarfile
import tempfile
import unittest

INSTALLER = Path(__file__).resolve().parent / "install.sh"
SOURCE = "a" * 40
TAG = "v1.2.3"
PAYLOAD = b"fixture binary bytes: never executed\n"
REAL_GH = shutil.which("gh")


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="xoscal-install-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.tools = self.root / "tools"
        self.tools.mkdir()
        # Check all trusted identity flags and digest binding, while explicitly
        # simulating success/failure. This is not a signature verifier.
        gh = self.tools / "gh"
        gh.write_text("""#!/usr/bin/env python3
import hashlib, json, os, sys
from pathlib import Path
a = sys.argv[1:]
expected = {
 '--hostname':'github.com', '--repo':'ckodex-labs/ckodex-xoscal',
 '--signer-digest':'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
 '--source-digest':'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
 '--source-ref':'refs/tags/v1.2.3',
 '--cert-identity':'https://github.com/ckodex-labs/ckodex-xoscal/.github/workflows/release.yml@refs/tags/v1.2.3',
 '--cert-oidc-issuer':'https://token.actions.githubusercontent.com',
 '--predicate-type':'https://slsa.dev/provenance/v1'}
try:
 assert a[:2] == ['attestation', 'verify']
 assert sum(flag in a for flag in ('--cert-identity', '--cert-identity-regex', '--signer-repo', '--signer-workflow')) == 1
 Path(os.environ['XOSCAL_TEST_GH_ARGS']).write_text(json.dumps(a))
 assert '--deny-self-hosted-runners' in a
 for key, value in expected.items(): assert a[a.index(key)+1] == value
 proof = json.loads(Path(a[a.index('--bundle')+1]).read_text())
 assert proof['identity'] == expected['--cert-identity']
 assert proof['digest'] == hashlib.sha256(Path(a[2]).read_bytes()).hexdigest()
except Exception:
 sys.exit(1)
""")
        gh.chmod(0o755)
        self.command_log = self.root / "gh-arguments.json"
        self.env = dict(os.environ, PATH=str(self.tools) + os.pathsep + os.environ["PATH"],
                        XOSCAL_TEST_GH_ARGS=str(self.command_log))
        self.os_name = {"Darwin": "darwin", "Linux": "linux"}[platform.system()]
        self.arch = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}[platform.machine().lower()]
        suffix = "x86_64" if self.arch == "amd64" else "arm64"
        self.name = "ckodex-xoscal_1.2.3_" + self.os_name.title() + "_" + suffix + ".tar.gz"
        self.archive = self.root / self.name
        self.manifest = self.root / "release-manifest.json"
        self.bundle = self.root / "release-manifest.sigstore.json"
        self.prefix = self.root / "installed"
        self.make_archive()
        self.write_manifest()

    def make_archive(self, members=None):
        with tarfile.open(self.archive, "w:gz") as tar:
            for name, payload, kind in members or [("xoscal-ctl", PAYLOAD, tarfile.REGTYPE), ("README.md", b"readme", tarfile.REGTYPE)]:
                info = tarfile.TarInfo(name)
                info.type = kind
                info.size = len(payload) if kind == tarfile.REGTYPE else 0
                info.linkname = "xoscal-ctl" if kind == tarfile.SYMTYPE else ""
                tar.addfile(info, io.BytesIO(payload) if kind == tarfile.REGTYPE else None)

    def write_manifest(self, change=None):
        data = {"schema_version": "xoscal-release-v1", "scope": "final", "release_tag": TAG, "source_revision": SOURCE,
                "artifacts": [{"path": "release/" + self.name, "name": self.name, "kind": "cli-archive",
                               "platform": {"os": self.os_name, "arch": self.arch},
                               "size": self.archive.stat().st_size,
                               "digest": "sha256:" + hashlib.sha256(self.archive.read_bytes()).hexdigest()}]}
        if change:
            change(data)
        self.manifest.write_text(json.dumps(data))
        self.bundle.write_text(json.dumps({"identity": "https://github.com/ckodex-labs/ckodex-xoscal/.github/workflows/release.yml@refs/tags/" + TAG,
                                           "digest": hashlib.sha256(self.manifest.read_bytes()).hexdigest()}))

    def run_installer(self, *extra, success=True):
        command = ["sh", str(INSTALLER), "--release", TAG, "--source", SOURCE,
                   "--manifest", str(self.manifest), "--bundle", str(self.bundle),
                   "--archive", str(self.archive), "--prefix", str(self.prefix), *extra]
        result = subprocess.run(command, env=self.env, text=True, capture_output=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        return result

    def assert_not_installed(self):
        self.assertFalse((self.prefix / "bin/xoscal-ctl").exists())

    def test_help_does_not_require_proof(self):
        result = subprocess.run(["sh", str(INSTALLER), "--help"], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0)
        self.assertIn("--source", result.stdout)

    def test_verified_fixture_dry_run(self):
        self.run_installer("--dry-run")
        self.assert_not_installed()

    def test_verified_fixture_install_and_no_clobber(self):
        self.run_installer()
        target = self.prefix / "bin/xoscal-ctl"
        self.assertEqual(target.read_bytes(), PAYLOAD)
        self.assertEqual(target.stat().st_mode & 0o777, 0o755)
        self.run_installer(success=False)
        self.assertEqual(target.read_bytes(), PAYLOAD)
        self.run_installer("--force")

    @unittest.skipUnless(REAL_GH, "installed gh CLI required for actual flag compatibility check")
    def test_installer_flags_accepted_by_actual_gh_and_invalid_proof_refused(self):
        # Capture the real installer command through its control-flow fixture,
        # then feed those flags to the actual CLI with deliberately invalid
        # trust/bundle data. This exercises Cobra validation, never signing.
        self.run_installer("--dry-run")
        command = json.loads(self.command_log.read_text())
        command[2] = str(self.manifest)
        command[command.index("--bundle") + 1] = str(self.bundle)
        invalid_root = self.root / "invalid-trusted-root.json"
        invalid_root.write_text("{}\n")
        command += ["--custom-trusted-root", str(invalid_root)]
        environment = dict(os.environ, GH_TOKEN="", GITHUB_TOKEN="", GH_PROMPT_DISABLED="1",
                           GH_CONFIG_DIR=str(self.root / "empty-gh-config"))
        result = subprocess.run([REAL_GH, *command], env=environment, text=True,
                                capture_output=True, timeout=30)
        self.assertNotEqual(result.returncode, 0, "invalid proof must never verify")
        error = (result.stdout + result.stderr).lower()
        self.assertNotIn("mutually exclusive", error)
        self.assertNotIn("if any flags in the group", error)
        self.assertNotIn("unknown flag", error)
        self.assertTrue(any(word in error for word in ("sigstore", "trusted root", "bundle")), error)
        self.assert_not_installed()
        conflicting = command + ["--signer-workflow", "ckodex-labs/ckodex-xoscal/.github/workflows/release.yml"]
        rejected = subprocess.run([REAL_GH, *conflicting], env=environment, text=True,
                                  capture_output=True, timeout=30)
        self.assertNotEqual(rejected.returncode, 0)
        self.assertIn("if any flags in the group", rejected.stderr.lower())

    def test_payload_or_missing_scope_manifest_refused(self):
        for scope in ("payload", None):
            with self.subTest(scope=scope):
                self.write_manifest(lambda data: data.update(scope=scope))
                self.assertIn("final admitted", self.run_installer(success=False).stderr)
                self.assert_not_installed()

    def test_tampered_archive_rejected(self):
        raw = bytearray(self.archive.read_bytes())
        raw[-1] ^= 1
        self.archive.write_bytes(raw)
        self.assertIn("digest/size mismatch", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_tampered_manifest_rejected_before_archive(self):
        self.manifest.write_bytes(self.manifest.read_bytes() + b" ")
        self.assertIn("provenance verification failed", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_wrong_identity_proof_rejected(self):
        proof = json.loads(self.bundle.read_text())
        proof["identity"] = "https://github.com/attacker/repo/.github/workflows/release.yml@refs/tags/" + TAG
        self.bundle.write_text(json.dumps(proof))
        self.assertIn("provenance verification failed", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_missing_proof_rejected(self):
        self.bundle.unlink()
        self.assertIn("signed provenance bundle", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_wrong_manifest_source_rejected(self):
        self.write_manifest(lambda data: data.update(source_revision="b" * 40))
        self.assertIn("requested release and source", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_wrong_manifest_tag_rejected(self):
        self.write_manifest(lambda data: data.update(release_tag="v1.2.4"))
        self.run_installer(success=False)
        self.assert_not_installed()

    def test_mutable_latest_rejected(self):
        self.assertIn("explicit", self.run_installer("--release", "latest", success=False).stderr)
        self.assert_not_installed()

    def test_duplicate_artifact_rejected(self):
        self.write_manifest(lambda data: data["artifacts"].append(dict(data["artifacts"][0])))
        self.assertIn("duplicate artifact", self.run_installer(success=False).stderr)
        self.assert_not_installed()

    def test_wrong_platform_rejected(self):
        self.write_manifest(lambda data: data["artifacts"][0].update(platform={"os": "windows", "arch": "amd64"}))
        self.run_installer(success=False)
        self.assert_not_installed()

    def test_unsafe_manifest_path_rejected(self):
        self.write_manifest(lambda data: data["artifacts"][0].update(path="../" + self.name))
        self.run_installer(success=False)
        self.assert_not_installed()

    def test_unsafe_archive_rejected(self):
        for name in ("../escape", "/absolute", "dir/../../escape"):
            with self.subTest(name=name):
                self.make_archive([("xoscal-ctl", PAYLOAD, tarfile.REGTYPE), (name, b"unsafe", tarfile.REGTYPE)])
                self.write_manifest()
                self.run_installer(success=False)
                self.assert_not_installed()

    def test_symlink_archive_rejected(self):
        self.make_archive([("xoscal-ctl", PAYLOAD, tarfile.REGTYPE), ("link", b"", tarfile.SYMTYPE)])
        self.write_manifest()
        self.run_installer(success=False)
        self.assert_not_installed()

    def test_missing_binary_rejected(self):
        self.make_archive([("README.md", b"readme", tarfile.REGTYPE)])
        self.write_manifest()
        self.run_installer(success=False)
        self.assert_not_installed()


if __name__ == "__main__":
    print("Local fixture checks only; hosted GitHub/Sigstore verification is not exercised.", flush=True)
    unittest.main(verbosity=2)
