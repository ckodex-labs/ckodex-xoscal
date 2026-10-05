# Checked Go runtime SDK

This local Python SDK preserves the builtin Dagger `v0.21.7` Go API generation
and static type-definition contract, then admits and compiles a fixed Go runtime
graph. The root `dagger.json` selects it. Python bootstrapping avoids compiling
the wrapper itself with the affected builtin Go dependency graph.

The upstream [Go SDK manifest](https://github.com/dagger/dagger/blob/v0.21.7/sdk/go/go.mod)
and [generator](https://github.com/dagger/dagger/blob/v0.21.7/cmd/codegen/generator/go/generate_module.go)
force four OTel log modules to `v0.16.0`. This SDK removes those replacements
*after* builtin generation, runs tidy and checksum verification, and validates
the actual `go list -m -json all` selected graph. Admission requires the reviewed
local `otel-go` compatibility fork, stable OTel `v1.45.0`, experimental log
modules `v0.21.0`, and gRPC `v1.83.2`. Missing dependencies, duplicate graph
entries, old replacement versions, and unreviewed replacement paths fail closed.

`module_types` delegates only the builtin static definition hook. In the pinned
[engine definition loader](https://github.com/dagger/dagger/blob/v0.21.7/core/schema/modulesource.go),
`runModuleDefInSDK` copies descriptions, object/interface/enum definitions onto
the original module; it retains this SDK's source and runtime hook. The builtin
[Go ModuleTypes implementation](https://github.com/dagger/dagger/blob/v0.21.7/core/sdk/go_sdk.go)
uses codegen for definitions, without executing the builtin user module runtime.
The engine and its prebuilt generator remain the pinned toolchain trust boundary.

Before returning a runtime, the SDK runs `govulncheck v1.7.0`, retains raw JSON,
and converts that same report through the text handler's finding exit gate.
JSON mode alone returns success for vulnerability findings and is never used
as admission. A pinned scanner report fixture containing a called vulnerability
must fail conversion with the documented finding exit status; the control input
and output are retained. It then runs module tests and vet, followed by the fork's checksum,
race, vet and vulnerability checks with the same JSON conversion gate. It compiles
the binary only after these gates pass. The compiler uses the immutable
`golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195`
image, `GOTOOLCHAIN=local`, and limited Go parallelism. No downgrade, vulnerability
exception, or suppressed scanner failure is used.

Retain evidence with the pinned CLI from the repository root:

```sh
dagger -m dagger/secure-go-sdk call runtime-evidence --mod-source . export --path /tmp/xoscal-runtime-evidence
```

This invokes the same audited compiler helper as `module_runtime` and exports
the actual graph, selected manifests, compiler version, raw scanner JSON,
test/vet outputs, and runtime SHA-256. `admission.json` exists only after all
commands succeed. This evidence concerns the module compiler, not hosted release
signing or the entire application's release pipeline.

Run admission negative tests without Dagger:

```sh
python3 -m unittest discover -s dagger/secure-go-sdk/tests -v
```

Maintain `graph.py` and the fork together when upgrading dependencies. Removing
this SDK before the upstream generator stops forcing vulnerable modules restores
the original problem. Roll back only to a verified runtime toolchain; do not
reintroduce `v0.16.0` to recover compilation. Do not export generated context over
active parallel edits; coordinate `dagger develop` with other repository work.
