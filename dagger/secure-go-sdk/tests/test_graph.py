import importlib.util
import json
from pathlib import Path
import unittest

path = Path(__file__).parents[1] / "src/secure_go_sdk/graph.py"
spec = importlib.util.spec_from_file_location("secure_graph", path)
graph = importlib.util.module_from_spec(spec)
spec.loader.exec_module(graph)


class GraphTests(unittest.TestCase):
    def records(self):
        return [
            {"Path": path, "Version": version}
            for path, version in graph.EXPECTED.items()
        ] + [{"Path": "github.com/dagger/otel-go", "Version": "v1.43.0", "Replace": {"Path": "./third_party/otel-go"}}]

    def text(self, records):
        return "\n".join(json.dumps(record) for record in records)

    def test_fixed_selected_graph_is_admitted(self):
        graph.validate_graph(self.text(self.records()))

    def test_generated_vulnerable_replacement_is_rejected(self):
        for path in graph.LOG_MODULES:
            records = self.records()
            record = next(record for record in records if record["Path"] == path)
            record["Replace"] = {"Path": path, "Version": "v0.16.0"}
            with self.subTest(path=path), self.assertRaises(ValueError):
                graph.validate_graph(self.text(records))

    def test_missing_module_and_duplicate_module_are_rejected(self):
        records = self.records()
        with self.assertRaises(ValueError):
            graph.validate_graph(self.text(records[1:]))
        with self.assertRaises(ValueError):
            graph.validate_graph(self.text(records + records[:1]))

    def test_unreviewed_local_module_replacement_is_rejected(self):
        records = self.records()
        records[0]["Replace"] = {"Path": "./unreviewed"}
        with self.assertRaises(ValueError):
            graph.validate_graph(self.text(records))

    def test_missing_reviewed_telemetry_fork_is_rejected(self):
        records = self.records()
        records[-1].pop("Replace")
        with self.assertRaises(ValueError):
            graph.validate_graph(self.text(records))


if __name__ == "__main__":
    unittest.main()
