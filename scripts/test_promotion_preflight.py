import importlib.util
from pathlib import Path
from subprocess import CompletedProcess
import unittest

spec = importlib.util.spec_from_file_location('preflight', Path(__file__).with_name('promotion-preflight.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class PreflightTests(unittest.TestCase):
    def run_gate(self, release, image):
        responses = iter([CompletedProcess([], *release), CompletedProcess([], *image)])
        return module.preflight('v1.2.3', 'a' * 40, run=lambda *a, **k: next(responses))

    def test_existing_and_uncertain_release_block(self):
        for response in ((0, 'HTTP/2 200 OK\n', ''), (1, 'HTTP/2 403 Forbidden\n', ''), (1, '', 'timeout')):
            with self.assertRaisesRegex(ValueError, 'release'):
                self.run_gate(response, (1, '', 'manifest unknown'))

    def test_existing_and_uncertain_registry_block(self):
        for response in ((0, '{}', ''), (1, '', 'unauthorized'), (1, '', 'timeout')):
            with self.assertRaisesRegex(ValueError, 'registry'):
                self.run_gate((1, 'HTTP/2 404 Not Found\n', ''), response)

    def test_only_confirmed_absence_passes(self):
        self.run_gate((1, 'HTTP/2 404 Not Found\n', ''), (1, '', 'manifest unknown'))
