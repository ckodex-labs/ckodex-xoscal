# SDK package contract

`dagger call sdk-packages --source=.` owns fresh generation, packaging and clean
consumer checks. `sdk-package --language=<go|python|java|csharp|ts|swift>` isolates
one language. All outputs remain unsigned candidates until the release manifest
and hosted provenance verifier admit their exact bytes.

| Download | Supported contents | Consumer gate |
| --- | --- | --- |
| `go.zip` | Go module `github.com/mchorfa/xoscal/proto/oscal`, protobuf messages and all four gRPC clients | unpack ZIP; external module with local replacement; compile and protobuf roundtrip |
| `python.zip` | namespaced `xoscal_sdk` source, wheel and sdist | separate fresh venvs install wheel and sdist; import all four gRPC stubs; message and pickle roundtrips; pip dependency checks |
| `java.zip` | Maven sources, JAR and POM, protobuf messages and all four gRPC stubs; Java 17+ | fresh Maven consumer installs actual JAR/POM; resolves runtime dependencies; compile and roundtrip |
| `csharp.zip` | .NET 8 NuGet package, protobuf messages and all four generated gRPC clients | fresh .NET 8 consumer restores actual local nupkg, roundtrips a message and exercises all four clients' HTTP/2 gRPC wire routes through a local HTTP handler |
| `ts.zip` | npm tarball, ESM and CommonJS modules, TypeScript declarations and all four service descriptors | fresh Node consumer installs actual tarball; ESM/CJS imports and roundtrip; strict TypeScript compile |
| `swift.zip` | Swift Package Manager manifest and **public protobuf models**, Swift 6+ | external Linux Swift consumer builds actual unpacked ZIP and roundtrips a message |

Each native package is also present in `packages/<language>/`. ZIP members use
lexical order, fixed 1980 timestamps, regular-file modes, and no symlinks. Generated
Compiled outputs are covered by their actual digest. Dagger does not infer reproducibility across
different toolchains from a matching source ZIP. Source-date/reproducible compiler
settings are applied where supported. Consumer evidence is `smoke/<language>.json`;
it is written only after a successful command and is local evidence, not signing.
`sdk-verify.py` binds those consumer subjects to the final native package and ZIP
bytes and checks that the ZIP embeds the exact published native package. The
aggregate producer emits `sdk-contract-check.json` only after that gate passes.
Each ZIP includes `SDK-PRODUCER.json` with the declared repository producer and
canonical repository URL. Native package author/developer metadata declares the
same owner. These fields support SBOM attribution; they do not claim registry
publication or replace hosted provenance verification.
`python3 scripts/sdk-verify-test.py <exported-sdk-directory>` exercises tampered
archive/native bytes, sidecar rewriting, incomplete verdict and missing report
against actual produced artifacts. Go package metadata deliberately retains the
fixed OpenTelemetry 1.45 family even for model-only consumers of gRPC.
The verifier also checks package versions against the common `sdk-version.txt`;
the Go consumer exports its actual module graph as `smoke/go-modules.txt` and
requires all five core/SDK OTel modules at 1.45.0. Stable Python versions preserve
the SemVer number; `rc`, `alpha` and `beta` use PEP 440 spellings. Other prerelease
names preserve their identity as a development/local version, for example
`v0.0.0-local` becomes Python `0.0.0.dev0+local`. These local candidate versions
do not claim a published release.

## Installation

First verify the pinned release manifest and hosted provenance as described in
RELEASE-CONTRACT.md. SHA sidecars detect corruption but are not signatures.

- Go: unpack `go.zip`; in a consumer module use
  `go mod edit -replace=github.com/mchorfa/xoscal/proto/oscal=/absolute/unpacked/path`
  and `go get github.com/mchorfa/xoscal/proto/oscal@v0.0.0`; import
  `github.com/mchorfa/xoscal/proto/oscal/services/v1`.
- Python: `python -m pip install packages/python/xoscal_sdk-<version>-py3-none-any.whl`;
  import `xoscal_sdk.services.v1.oscal_service_pb2_grpc`.
- Maven: `mvn org.apache.maven.plugins:maven-install-plugin:3.1.4:install-file
  -Dfile=packages/java/xoscal-sdk-<version>.jar
  -DpomFile=packages/java/xoscal-sdk.pom`; depend on `io.ckodex:xoscal-sdk:<version>`.
- .NET: add the absolute `packages/csharp/` directory as a NuGet source alongside
  nuget.org; add `Xoscal.SDK` at the pinned release version. Import `Oscal.Services.V1`.
- Node: `npm install packages/ts/ckodex-xoscal-sdk-<version>.tgz`; import or require
  `@ckodex/xoscal-sdk/services/v1/oscal_service_pb`.
- Swift: unpack `swift.zip`; add `.package(path: "/absolute/unpacked/path")`
  and product `XoscalSDK` to your SPM target. Import `XoscalSDK`.

## Concrete compatibility decisions

The pinned generators produce protobuf models, not Python Pydantic validation
models or Java Jackson bindings. Those claims are removed; this release does not
silently add another serialization contract. OSCAL structural validation stays
in the service and schema validation pipeline.

C# SDK generation adds pinned `buf.build/grpc/csharp:v1.83.1`, producing current
client stubs alongside the models. The NuGet package uses `Grpc.Core.Api` and
`Grpc.Net.Client` 2.76.0. Its fresh consumer verifies HTTP/2 gRPC framing, request
and response marshalling, and a unary route for each of the four services through
a local HTTP handler. This is package/transport adapter evidence; it does not
claim a live server or hosted TLS connection was exercised. No stale checked-in
`*Grpc.cs` files enter the package.

The Connect Swift runtime is an Apple-platform transport implementation. A Linux
Dagger engine cannot establish macOS/iOS Connect client usability. Therefore the
release supplies a tested portable SPM protobuf **model** package, without Connect
clients, and makes no Apple runtime or live transport claim. Adding that promise
requires generation with public client visibility and a real macOS/iOS consumer
gate before promotion. JSON gateway and gRPC client transport integration is a
separate service-level gate; package smoke tests do not claim network coverage.

No package registry publication is performed here. Consumers use verified local
release packages. New registry publication or signing identities need separate
authorization.
