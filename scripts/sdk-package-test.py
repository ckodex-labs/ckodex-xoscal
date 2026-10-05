#!/usr/bin/env python3
"""Archive contract tests: reproducible bytes, stale-file exclusion, fail closed."""
import importlib.util
import tempfile
import unittest
import zipfile
from pathlib import Path

spec = importlib.util.spec_from_file_location("sdk_package", Path(__file__).with_name("sdk-package.py"))
sdk = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sdk)


class SDKArchives(unittest.TestCase):
    def test_release_prerelease_identity_for_python(self):
        self.assertEqual(sdk.python_version("v0.0.0-local"), "0.0.0.dev0+local")
        self.assertEqual(sdk.python_version("v1.2.3-rc.4"), "1.2.3rc4")
        self.assertEqual(sdk.python_version("v1.2.3"), "1.2.3")

    def test_archive_bytes_ignore_mtime_and_discovery_order(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "source"
            source.mkdir()
            (source / "z.txt").write_text("z")
            (source / "a.txt").write_text("a")
            sdk.archive(source, root / "a.zip")
            import os
            os.utime(source / "z.txt", (1700000000, 1700000000))
            sdk.archive(source, root / "b.zip")
            self.assertEqual((root / "a.zip").read_bytes(), (root / "b.zip").read_bytes())
            with zipfile.ZipFile(root / "a.zip") as z:
                self.assertEqual(z.namelist(), ["a.txt", "z.txt"])
                self.assertTrue(all(i.date_time == (1980, 1, 1, 0, 0, 0) for i in z.infolist()))

    def test_reject_traversal_and_symlink_members(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ["../escape", "/escape", "dir\\escape"]:
                with zipfile.ZipFile(root / "bad.zip", "w") as z:
                    z.writestr(name, "bad")
                with self.assertRaises(ValueError):
                    sdk.extract(root / "bad.zip", root / "destination")
            info = zipfile.ZipInfo("link")
            info.create_system = 3
            info.external_attr = 0o120777 << 16
            with zipfile.ZipFile(root / "bad.zip", "w") as z:
                z.writestr(info, "target")
            with self.assertRaises(ValueError):
                sdk.extract(root / "bad.zip", root / "destination")

    def test_empty_generation_and_invalid_version_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "generated/python").mkdir(parents=True)
            with self.assertRaises(ValueError):
                sdk.prepare(root / "generated", root / "out", "python", "dev")
            for value in ["latest", "../0.1.0", "1.2.3;echo", "1.2.3\n"]:
                with self.assertRaises(ValueError):
                    sdk.version(value)

    def test_sources_exclude_cached_and_stale_clients(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "generated/python/common/v1"
            source.mkdir(parents=True)
            (source / "common_pb2.py").write_text("from common.v1 import other_pb2\n")
            (source / "common_pb2_grpc.py").write_text("from common.v1 import common_pb2\n")
            (source / "cached.pyc").write_bytes(b"cache")
            sdk.prepare(root / "generated", root / "out", "python", "dev")
            self.assertFalse(list((root / "out").rglob("*.pyc")))
            generated = (root / "out/xoscal_sdk/common/v1/common_pb2.py").read_text()
            self.assertIn("from xoscal_sdk.common.v1", generated)


if __name__ == "__main__":
    unittest.main()
