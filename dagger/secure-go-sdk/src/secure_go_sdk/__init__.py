"""Builtin Go generation followed by admission of the patched runtime graph."""

import dagger
import json
from dagger import dag, function, object_type

from .graph import LOG_MODULES, validate_graph

GO_IMAGE = "golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195"


@object_type
class SecureGoSdk:
    @function
    async def module_types(
        self, mod_source: dagger.ModuleSource, introspection_json: dagger.File,
        output_file_path: str,
    ) -> dagger.Container:
        # Go 0.21.7 has a separate static type-definition hook; its dispatcher
        # deliberately cannot answer the reflective empty-name runtime call.
        # The engine copies only these definitions onto the original module,
        # retaining this SDK's checked runtime rather than the delegated one.
        module_id = await mod_source.with_sdk("go").as_module().id()
        return (
            dag.container().from_(GO_IMAGE)
            .with_new_file(output_file_path, json.dumps(str(module_id)))
            .with_entrypoint(["true"])
        )

    async def generated(self, mod_source: dagger.ModuleSource) -> dagger.Container:
        # Delegate only this source to the builtin SDK. Calling this SDK's own
        # generated_context_directory would recurse into its custom hook.
        builtin = mod_source.with_sdk("go")
        source_path = await builtin.source_subpath()
        root_path = await mod_source.source_root_subpath()
        config_path = f"{root_path}/dagger.json" if root_path else "dagger.json"
        generated = mod_source.context_directory().with_directory(
            ".", builtin.generated_context_directory()
        ).with_file(
            config_path, mod_source.context_directory().file(config_path)
        ).with_directory(
            f"{source_path}/third_party",
            mod_source.context_directory().directory(f"{source_path}/third_party"),
        )
        command = ["go", "mod", "edit"]
        command.extend(f"-dropreplace={module}" for module in LOG_MODULES)
        ctr = (
            dag.container()
            .from_(GO_IMAGE)
            .with_directory("/src", generated)
            .with_workdir(f"/src/{source_path}")
            .with_env_variable("GOTOOLCHAIN", "local")
            .with_env_variable("GOFLAGS", "-p=2")
            .with_env_variable("GOMAXPROCS", "2")
            .with_mounted_cache("/go/pkg/mod", dag.cache_volume("secure-go-sdk-mod-1.45-0.21"))
            .with_mounted_cache("/root/.cache/go-build", dag.cache_volume("secure-go-sdk-build-1.27.1"))
            .with_exec(command)
            .with_exec(["go", "mod", "tidy"])
            .with_exec(["go", "mod", "verify"])
            .with_exec(["sh", "-c", "mkdir -p /runtime-evidence; go list -m -json all > /runtime-evidence/go-modules.json"])
        )
        validate_graph(await ctr.file("/runtime-evidence/go-modules.json").contents())
        return ctr

    @function
    async def codegen(
        self, mod_source: dagger.ModuleSource, introspection_json: dagger.File
    ) -> dagger.GeneratedCode:
        ctr = await self.generated(mod_source)
        return (
            dag.generated_code(ctr.directory("/src"))
            .with_vcs_generated_paths(["dagger.gen.go", "internal/dagger/**"])
            .with_vcs_ignored_paths(["dagger.gen.go", "internal/dagger"])
        )

    @function
    async def module_runtime(
        self, mod_source: dagger.ModuleSource, introspection_json: dagger.File
    ) -> dagger.Container:
        ctr = await self.audited(mod_source)
        return (
            ctr.with_workdir("/scratch")
            .without_mount("/go/pkg/mod")
            .without_mount("/root/.cache/go-build")
            .with_entrypoint(["/runtime"])
        )

    async def audited(self, mod_source: dagger.ModuleSource) -> dagger.Container:
        ctr = await self.generated(mod_source)
        ctr = (
            ctr.with_exec(["sh", "-ec", "go version > /runtime-evidence/compiler.txt; cp go.mod go.sum /runtime-evidence/"])
            # JSON output alone returns zero for vulnerability findings. Admit
            # the same report through text conversion's finding exit gate.
            .with_exec(["sh", "-ec", "go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -json ./... > /runtime-evidence/govulncheck.json; go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -mode=convert < /runtime-evidence/govulncheck.json > /runtime-evidence/govulncheck.txt"])
            .with_exec(["sh", "-ec", '''
scanner_fixture="$(go env GOMODCACHE)/golang.org/x/vuln@v1.7.0/internal/scan/testdata/source.json"
cp "$scanner_fixture" /runtime-evidence/scanner-negative-input.json
if go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -mode=convert < "$scanner_fixture" > /runtime-evidence/scanner-negative.txt 2>&1; then
    printf 'scanner admitted the called-vulnerability control\n' >&2
    exit 1
fi
grep -q '^exit status 3$' /runtime-evidence/scanner-negative.txt
grep -q 'GO-0000-0001' /runtime-evidence/scanner-negative.txt
'''])
            .with_exec(["sh", "-ec", "go test ./... > /runtime-evidence/tests.txt 2>&1"])
            .with_exec(["sh", "-ec", "go vet ./... > /runtime-evidence/vet.txt 2>&1"])
            .with_exec(["sh", "-ec", "cd third_party/otel-go; go mod verify > /runtime-evidence/fork-checksums.txt; go test -race ./... > /runtime-evidence/fork-tests.txt 2>&1; go vet ./... > /runtime-evidence/fork-vet.txt 2>&1; go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -json ./... > /runtime-evidence/fork-govulncheck.json; go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 -mode=convert < /runtime-evidence/fork-govulncheck.json > /runtime-evidence/fork-govulncheck.txt"])
            .with_exec(["sh", "-ec", "go build -ldflags '-s -w' -o /runtime . > /runtime-evidence/build.txt 2>&1; sha256sum /runtime > /runtime-evidence/runtime.sha256"])
            .with_new_file("/runtime-evidence/admission.json", json.dumps({
                "schema": 1, "engineVersion": "v0.21.7", "compilerImage": GO_IMAGE,
                "graph": "go-modules.json", "runtimeDigest": "runtime.sha256",
                "stages": ["graph", "checksums", "govulncheck", "scanner-negative-control", "tests", "vet", "fork-race-tests", "fork-govulncheck", "build"],
                "status": "passed",
            }, indent=2) + "\n")
        )
        # Force all gates before admitting a container or evidence directory.
        return await ctr.sync()

    @function
    async def runtime_evidence(self, mod_source: dagger.ModuleSource) -> dagger.Directory:
        """Retain checks of the same compiler and module graph used by runtime."""
        return (await self.audited(mod_source)).directory("/runtime-evidence")

    @function
    def required_paths(self) -> list[str]:
        return ["**/*.go", "**/go.mod", "**/go.sum", "third_party/**"]
