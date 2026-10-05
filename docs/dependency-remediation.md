# Go dependency remediation

Scope: independent dependency change in the root application and nested Dagger
module, assessed on 2026-10-05. Host and actual pinned-engine compiler checks
establish the selected graphs and compatibility; hosted release identity/signing
is a separate boundary.

## Advisory and actual application path

[GO-2026-6505](https://pkg.go.dev/vuln/GO-2026-6505) and the
[upstream advisory](https://github.com/open-telemetry/opentelemetry-go/security/advisories/GHSA-8wmf-6v46-5gfg)
cover SDK and trace exporters through `v1.44.0`, fixed in `v1.45.0`.
The conditional exposure requires verbose OpenTelemetry internal diagnostic logging.
The Go database entry is marked unreviewed; the upstream details were checked directly.

No application endpoint leak was demonstrated. The server initializes a stdout
exporter in `server/internal/observability/tracing.go`, tracing defaults to false
in `server/internal/config/config.go`, and repository Go sources do not install
an OpenTelemetry diagnostic logger with `otel.SetLogger`. This is removal of an
affected dependency, without a claim that collector credentials were exposed.

## Selected versions and compatibility

The root stable OTel core, SDK, stdout exporter, trace and metric modules select
`v1.45.0`. The Dagger stable SDK, metrics and OTLP trace/metric exporter family
selects `v1.45.0`; its OTel log API/SDK/gRPC and HTTP exporters select `v0.21.0`,
and gRPC selects `v1.83.2`.

The additional nested advisories found during verification were:

| Advisory | Affected selected module before remediation | Fixed version |
| --- | --- | --- |
| [GO-2026-6615](https://pkg.go.dev/vuln/GO-2026-6615) | SDK log `v0.16.0`, batch processor busy-spin | `v0.21.0` |
| [GO-2026-6508](https://pkg.go.dev/vuln/GO-2026-6508) | OTLP log gRPC `v0.16.0`, TLS environment configuration | `v0.21.0` |
| [GO-2026-4985](https://pkg.go.dev/vuln/GO-2026-4985) | OTLP log HTTP `v0.16.0`, oversized response exhaustion | `v0.19.0` |
| [GO-2026-6348](https://pkg.go.dev/vuln/GO-2026-6348) | gRPC `v1.83.0`, HTTP/2 memory exhaustion | `v1.83.1` |

Published `github.com/dagger/otel-go v1.43.0` cannot compile with log `v0.21.0`:
it uses removed `log.Value` and `log.KeyValue` symbols. The explicitly maintained
local module under `dagger/third_party/otel-go` migrates only these log value and
attribute representations, preserves upstream source/license provenance, and
includes consumer tests for OTLP scalar/byte/nested/typed-array roundtrips and
real SDK stdio payload/attribute/EOF preservation. See its `PATCHES.md`.
The old vulnerable `v0.16.0` replacements are removed. No advisory exception or
scanner threshold was added.

## Local verification

- Root: `go mod tidy`, `go mod verify`, `go test ./...`, `go test -race ./...`,
  `go build ./...`, `go vet ./...` passed with host Go `1.27.1`.
- Root pinned toolchain: `GOTOOLCHAIN=go1.25.13 go test ./...`, `go build ./...`,
  `go vet ./...` passed after the concurrent API generation completed.
- Root advisory gate: host `govulncheck v1.8.0 ./...` and pinned
  `GOTOOLCHAIN=go1.25.13 go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`
  reported no vulnerabilities.
- Nested Dagger and local telemetry module: tidy, checksum verification, tests,
  race, build, vet and `govulncheck ./...` passed with the patched graphs.
  Dagger itself has no Go unit tests; its runtime compilation is the relevant check.
- Synthetic verbosity-4 diagnostic trigger: both OTLP HTTP and gRPC providers
  disclosed the synthetic collector endpoint with SDK/exporters `v1.44.0` and
  did not disclose it with `v1.45.0`. Real provider creation/shutdown succeeded
  in both runs. The probe uses a fake `.invalid` endpoint and emits no spans.
- An independent read-only candidate review found no remaining trace-module
  downgrade or source-backed compatibility regression. `git diff --check` passed.

The initial nested tidy attempt, before ignored generated runtime sources existed,
removed unused module requirements. That output was discarded; the baseline was
restored and updates reapplied before generation and a successful final tidy.
Do not tidy this nested module before generating its runtime.

## Dagger regeneration boundary

The pinned engine remains `0.21.7`. The initial controlled invocation exposed a
builtin SDK blocker: static `dagger functions` succeeded, but `dagger call version`
failed to compile because generation silently substituted log `v0.16.0`.
`dagger develop --eager-runtime` returned zero while reintroducing those vulnerable
replacements. That earlier output was discarded and was never security evidence.

The exact upstream rule is
[`sdk/go/go.mod`](https://github.com/dagger/dagger/blob/v0.21.7/sdk/go/go.mod)
with four unversioned log replacements, and
[`generate_module.go`](https://github.com/dagger/dagger/blob/v0.21.7/cmd/codegen/generator/go/generate_module.go)
`syncModReplaceAndTidy`, which unconditionally runs `go mod edit -replace` for
each. These commands also remove attempted version-specific override pins.
Upstream `v0.21.9` has the same rule; a patch upgrade alone does not resolve it.

The root now selects `dagger/secure-go-sdk`, a local Python SDK that delegates
builtin Go generation/static definitions, removes the forced replacements after
generation, verifies checksums, and admits the actual compiler's selected graph.
It compiles the user runtime only after vulnerability, test, vet and fork race
gates pass. Pinned engine source confirms static-definition delegation copies
only definitions onto the original module and retains this custom runtime hook;
it does not execute the vulnerable builtin user runtime. The engine and its
prebuilt code generator remain the existing pinned-toolchain trust boundary.
See [the SDK contract](../dagger/secure-go-sdk/README.md).

The nested selected module requires Go `1.26.2`; its actual compiler is immutable
Go `1.27.1` on Linux/arm64, not the root application's separate Go `1.25.13` check.
`GOTOOLCHAIN=local` prevents an implicit runtime compiler upgrade.

Actual pinned-engine checks passed:

- `dagger functions` loaded the complete Xoscal function interface.
- `dagger call version` executed the checked runtime and returned `dev`.
- `dagger -m dagger/secure-go-sdk call runtime-evidence --mod-source . export`
  admitted the actual `go list -m -json all` graph, verified checksums, compiled
  the generated module, and ran its tests/vet plus fork race/tests/vet.
- Runtime and fork `govulncheck v1.7.0` reports contain zero finding messages;
  admission converts each exact retained JSON through the text finding gate and
  reports `No vulnerabilities found.` JSON mode alone returns success with
  findings and is explicitly insufficient. Independent review caught this issue
  before final admission verification; the enforcing conversion is now mandatory.
- All five graph negative tests passed, including each forced log `v0.16.0`
  replacement, missing/duplicate dependencies and an unreviewed local replacement.
  The pinned scanner's official called-vulnerability fixture was rejected inside
  the actual compiler container with exit status `3` (the `go run` wrapper returns
  `1`). This permanent regression control retains its input/output and must pass
  before admitting a runtime.

Final runtime evidence is retained at `/tmp/xoscal-runtime-evidence-final` on
MNC-MBP, with `admission.json`, actual graph/manifests/compiler output, scanner
JSON and enforcing text, test/vet outputs and runtime digest. The tested runtime
SHA-256 is `49eebf0239ea63e8e28d7aee4e3f025f5d450bde5f2f39e0024326fc4c22b3ce`.
Rerun the export after source changes; that digest identifies the tested snapshot,
not a future runtime or release payload. The full site/release pipeline remains
the integrating task's verification responsibility, and these compiler checks
do not establish hosted signing or release promotion proof.

The same audited helper guards every actual module runtime invocation and the
evidence export. Code generation validates the fixed graph before returning
generated context. Coordinate `dagger develop` with parallel edits, then rerun
runtime admission. Rollback must retain a verified fixed compiler/SDK graph;
restoring the vulnerable log replacements is not a supported rollback.

The synthetic probe and its before/after module inputs are retained by the
Codex Security managed artifact store. Repository documentation records the
commands and limits without converting local checks into hosted proof.
