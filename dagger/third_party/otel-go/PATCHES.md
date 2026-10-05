# Local OpenTelemetry log API compatibility

Upstream: https://github.com/dagger/otel-go, tag `v1.43.0`, commit
`a9e4d7ae7eb276fe94a2aebc27d08673d50f66b3` (Go proxy origin metadata).
The upstream Apache-2.0 `LICENSE` and source files are preserved. This local
module is an explicit fork; it does not claim to be an unmodified upstream release.

OpenTelemetry log `v0.21.0` fixes the SDK batch processor busy-spin and the log
gRPC exporter TLS certificate advisory. The earlier log HTTP response exhaustion
fix is included. Its log values now use `attribute.Value` and `attribute.KeyValue`.
Published Dagger telemetry `v1.43.0` uses the removed log value types, preventing a
manifest-only upgrade. The old `v0.16.0` replacements cannot satisfy the security gate.

Changes from upstream are limited to `logging.go`, `transform.go`, this document,
`compatibility_test.go`, and the aligned module manifests:

- Log writer/stdio values and attributes use the supported attribute API.
- OTLP log conversion uses attribute type constants and explicit key conversion.
- Byte bodies and the typed attribute slices supported by the new API retain their
  wire representation rather than falling through to an invalid value.
- Dependencies resolve stable OTel `v1.45.0`, log family `v0.21.0`, and gRPC `v1.83.2`.
- Tests cover scalar, byte, nested map/array and typed slice OTLP roundtrips, plus
  stdout/stderr payload, custom attribute and EOF preservation with the real log SDK.

Run `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod verify`, and
`govulncheck ./...` in this directory as well as the owning Dagger module.
Replace this fork with a released compatible upstream version after that version
passes these checks. Do not restore vulnerable replacements to make generation compile.
