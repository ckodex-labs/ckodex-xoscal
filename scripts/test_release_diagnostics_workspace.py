"""Execute the real diagnostic shell with explicitly synthetic tool fixtures."""
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


def fixture_dagger(tools):
    # The fixture checks cleanliness; it makes no analysis or signing claim.
    tool = tools / "dagger"
    tool.write_text("#!/usr/bin/env python3\n" + textwrap.dedent('''\
        import pathlib, subprocess, sys
        status = subprocess.check_output(['git', 'status', '--porcelain', '--untracked-files=all'], text=True)
        if status:
            print('dirty source fixture:', status, file=sys.stderr)
            sys.exit(13)
        args = sys.argv[1:]
        subject = next(a.split('=',1)[1] for a in args if a.startswith('--subject='))
        if subject == 'source':
            print('deliberate independent export failure fixture', file=sys.stderr)
            sys.exit(7)
        target = pathlib.Path(next(a.split('=',1)[1] for a in args if a.startswith('--path=')))
        target.mkdir(parents=True)
        (target / 'raw-fixture.txt').write_text('synthetic exporter fixture only\\n')
        '''))
    tool.chmod(0o755)


class DiagnosticWorkspace(unittest.TestCase):
    def test_preserves_original_bytes_and_keeps_source_clean_despite_one_export_failure(self):
        workflow = Path(__file__).resolve().parents[1] / ".github/workflows/release.yml"
        text = workflow.read_text().split("      - name: Preserve raw findings when candidate gate fails\n", 1)[1]
        run = text.split("        run: |\n", 1)[1].split("        env:\n", 1)[0]
        script = textwrap.dedent(run)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            checkout, workspace, tools = root / "source", root / "outputs", root / "tools"
            checkout.mkdir(); tools.mkdir()
            env = dict(os.environ, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL="/dev/null")
            subprocess.run(["git", "init", "-q", str(checkout)], check=True, env=env)
            (checkout / "tracked-source.txt").write_text("source fixture\n")
            subprocess.run(["git", "-C", str(checkout), "add", "."], check=True, env=env)
            subprocess.run(["git", "-C", str(checkout), "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                            "-c", "core.hooksPath=/dev/null", "commit", "-qm", "fixture"], check=True, env=env)
            original = workspace / "candidate/evidence/raw.json"
            original.parent.mkdir(parents=True)
            original.write_bytes(b'{"fixture":"original producer bytes; not real analysis"}\n')
            bundle = root / "payload-fixture.json"
            bundle.write_bytes(b'{"fixture":"not a signature"}\n')
            fixture_dagger(tools)
            env.update(PATH=str(tools) + os.pathsep + env["PATH"], RELEASE_WORKSPACE=str(workspace),
                       PAYLOAD_BUNDLE=str(bundle), RELEASE_TAG="v1.2.3", SOURCE_REVISION="a" * 40)
            result = subprocess.run(["sh", "-eu", "-c", script], cwd=checkout, env=env, capture_output=True, text=True)
            self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
            rows = (workspace / "failed-analysis/exports.tsv").read_text().splitlines()
            self.assertEqual(len(rows), 45)
            self.assertEqual(rows[1], "source\tvulnerabilities\t7")
            self.assertTrue(all(row.endswith("\t0") for row in rows[2:]))
            self.assertEqual((workspace / "failed-analysis/produced-candidate/evidence/raw.json").read_bytes(), original.read_bytes())
            self.assertEqual((workspace / "failed-analysis/payload.sigstore.json").read_bytes(), bundle.read_bytes())
            self.assertEqual(len(list((workspace / "failed-analysis/logs").glob("*.log"))), 44)
            status = subprocess.check_output(["git", "-C", str(checkout), "status", "--porcelain", "--untracked-files=all"], env=env, text=True)
            self.assertEqual(status, "")


if __name__ == "__main__":
    unittest.main()
