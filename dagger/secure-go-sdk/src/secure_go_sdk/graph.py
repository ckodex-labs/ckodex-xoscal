"""Admission for the exact generated Go runtime's selected module graph."""

import json

LOG_MODULES = (
    "go.opentelemetry.io/otel/log",
    "go.opentelemetry.io/otel/sdk/log",
    "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc",
    "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp",
)
EXPECTED = {module: "v0.21.0" for module in LOG_MODULES}
EXPECTED.update(
    {
        "go.opentelemetry.io/otel": "v1.45.0",
        "go.opentelemetry.io/otel/sdk": "v1.45.0",
        "go.opentelemetry.io/otel/trace": "v1.45.0",
        "go.opentelemetry.io/otel/metric": "v1.45.0",
        "go.opentelemetry.io/otel/sdk/metric": "v1.45.0",
        "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc": "v1.45.0",
        "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp": "v1.45.0",
        "go.opentelemetry.io/otel/exporters/otlp/otlptrace": "v1.45.0",
        "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc": "v1.45.0",
        "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp": "v1.45.0",
        "google.golang.org/grpc": "v1.83.2",
    }
)


def validate_graph(text: str) -> None:
    decoder = json.JSONDecoder()
    modules = {}
    remaining = text.lstrip()
    while remaining:
        module, end = decoder.raw_decode(remaining)
        path = module["Path"]
        if path in modules:
            raise ValueError(f"duplicate module: {path}")
        modules[path] = module
        remaining = remaining[end:].lstrip()
    for path, version in EXPECTED.items():
        if path not in modules:
            raise ValueError(f"runtime dependency missing: {path}")
        module = modules[path]
        replacement = module.get("Replace")
        actual = replacement or module
        if actual.get("Path") != path or actual.get("Version") != version:
            raise ValueError(f"unexpected runtime dependency: {path}: {actual}")
    fork = modules.get("github.com/dagger/otel-go", {}).get("Replace", {})
    if fork.get("Path") != "./third_party/otel-go" or fork.get("Version"):
        raise ValueError("runtime must use the reviewed local telemetry compatibility fork")
