"""The tag workflow cannot request a main-only deployment environment."""

import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[1]


class PagesHandoff(unittest.TestCase):
    def test_tag_workflow_records_exact_identity_without_requesting_protected_environment(self):
        release = (ROOT / ".github/workflows/release.yml").read_text()
        self.assertNotIn("actions/deploy-pages@", release)
        self.assertNotIn("name: github-pages", release)
        self.assertIn("path: admitted", release)
        handoff = release.split("  pages-handoff:\n", 1)[1]
        self.assertIn("needs: promote", handoff)
        self.assertNotIn("id-token: write", handoff)
        script = textwrap.dedent(handoff.split("        run: |\n", 1)[1].split("        env:\n", 1)[0])
        with tempfile.TemporaryDirectory() as temporary:
            summary = Path(temporary) / "summary.md"
            env = dict(os.environ, GITHUB_STEP_SUMMARY=str(summary), RELEASE_TAG="v1.2.3",
                       SOURCE_REVISION="a" * 40)
            result = subprocess.run(["sh", "-eu", "-c", script], env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            text = summary.read_text()
            self.assertIn("deployment is pending", text)
            self.assertIn("--ref main -f release_tag=v1.2.3 -f source_revision=" + "a" * 40, text)
            self.assertIn("both verification and deployment to succeed", text)

    def test_main_pages_workflow_reverifies_explicit_release_before_deployment(self):
        pages = (ROOT / ".github/workflows/pages.yml").read_text()
        self.assertIn("source_revision:", pages)
        self.assertIn("verify-published-release", pages)
        self.assertLess(pages.index("verify-published-release"), pages.index("actions/deploy-pages@"))
        self.assertIn("needs: verify", pages)
        self.assertIn("name: github-pages", pages)


if __name__ == "__main__":
    unittest.main()
