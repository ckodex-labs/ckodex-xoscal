package main

import (
	"fmt"
	"runtime"

	"dagger/xoscal/internal/dagger"
)

// Immutable Linux SDK toolchains for the two supported runner architectures.
// The language packages themselves are portable; these are build tools only.
var sdkImages = map[string][2]string{
	"python": {"python:3.12-slim-bookworm@sha256:9901e0a8d75037d8242ed43155cbcb2d1f61be1356383d8054afb59fd50e39c4", "python:3.12-slim-bookworm@sha256:349275ed26e7aea20752e1fd63c40c30a0281d324e51fb41612c2433cb6482d4"},
	"ts":     {"node:22-bookworm-slim@sha256:25330af3531fb5e23318554a0aa911125b6e91b1b777edf7655501d207c067a2", "node:22-bookworm-slim@sha256:a0ddbc73510e98f5e824fd64266ffe1c2c343ba9cf260d95ca2985ad632a3f3e"},
	"java":   {"maven:3.9-eclipse-temurin-21@sha256:80034909d9a9f0b89d3d2c583f07f3b7650309bc4e6b886ad5b860b9e1bfa439", "maven:3.9-eclipse-temurin-21@sha256:71a6832f657180ea206901ffc548c0e73a257cb09b4912619177586d0268d59c"},
	"csharp": {"mcr.microsoft.com/dotnet/sdk:8.0-bookworm-slim@sha256:7ff19b091200c3a3a1bf1eefa6abae9efd7995d8e3298e66aa0d6905872bf642", "mcr.microsoft.com/dotnet/sdk:8.0-bookworm-slim@sha256:499ef1502b0215a158a8051a0324dfcc62f69a2f7d0a8bde03b2fb0f5d15edec"},
	"swift":  {"swift:6.1-bookworm@sha256:48a6d9c230cd654b262f805978d1d56b2aadf5b3c0d2fc088643296692cbca62", "swift:6.1-bookworm@sha256:c6c5f7130ba5f11177b5c2ca48c4fd085e8a4bc52b14346fa66cc1a0aecbf24c"},
}

func sdkToolchain(language string) *dagger.Container {
	if language == "go" {
		return dag.Container().From(goBuilderImage).
			WithEnvVariable("GOFLAGS", "-p=2").WithEnvVariable("GOMAXPROCS", "2")
	}
	index := 0
	if runtime.GOARCH == "arm64" {
		index = 1
	}
	return dag.Container(dagger.ContainerOpts{Platform: dagger.Platform("linux/" + runtime.GOARCH)}).From(sdkImages[language][index]).
		WithEnvVariable("SOURCE_DATE_EPOCH", "315532800").
		WithEnvVariable("DOTNET_CLI_TELEMETRY_OPTOUT", "1").
		WithEnvVariable("DOTNET_SKIP_FIRST_TIME_EXPERIENCE", "1")
}

func (m *Xoscal) sdkGenerated(source *dagger.Directory) *dagger.Directory {
	return dag.Container().From(goBuilderImage).
		WithExec([]string{"sh", "-ec", "apt-get update && apt-get install -y --no-install-recommends ca-certificates curl python3"}).
		WithExec([]string{"sh", "-ec", pinnedBufInstall()}).
		WithDirectory("/src/proto/oscal", source.Directory("proto/oscal"), dagger.ContainerWithDirectoryOpts{Include: []string{"**/*.proto", "buf.yaml", "buf.lock"}}).
		WithFile("/src/scripts/sdk-buf.gen.yaml", source.File("scripts/sdk-buf.gen.yaml")).
		WithWorkdir("/src/proto/oscal").
		WithExec([]string{"buf", "generate", "--template", "/src/scripts/sdk-buf.gen.yaml"}).
		Directory("/generated")
}

func sdkPrepared(source, generated *dagger.Directory, language, version string) *dagger.Directory {
	prepared := sdkToolchain("python").
		WithDirectory("/generated", generated).
		WithFile("/sdk-package.py", source.File("scripts/sdk-package.py")).
		WithExec([]string{"python3", "/sdk-package.py", "prepare", "/generated", "/sdk", language, version}).
		WithFile("/sdk/SDK-CONTRACT.md", source.File("docs/SDK-CONTRACT.md")).
		WithFile("/sdk/LICENSE", source.File("LICENSE"))
	if language == "java" {
		prepared = prepared.WithFile("/sdk/src/main/resources/META-INF/LICENSE", source.File("LICENSE"))
	}
	return prepared.Directory("/sdk")
}

// SdkPackage produces one installable package and verifies an independent
// consumer of its exact bytes. This entry point also isolates language failures.
func (m *Xoscal) SdkPackage(source *dagger.Directory, language string) (*dagger.Directory, error) {
	switch language {
	case "go", "python", "java", "csharp", "ts", "swift":
		return m.sdkPackage(source, m.sdkGenerated(source), language), nil
	default:
		return nil, fmt.Errorf("unsupported SDK language %q", language)
	}
}

// SdkPackages generates once, compiles supported package formats, consumes the
// actual outputs in fresh containers, and returns six compatibility ZIPs plus
// native packages and per-language local smoke evidence. Any failed gate aborts.
func (m *Xoscal) SdkPackages(source *dagger.Directory) *dagger.Directory {
	generated := m.sdkGenerated(source)
	result := dag.Directory()
	for _, language := range []string{"go", "python", "java", "csharp", "ts", "swift"} {
		result = result.WithDirectory(".", m.sdkPackage(source, generated, language))
	}
	return sdkToolchain("python").WithDirectory("/out", result).
		WithFile("/sdk-verify.py", source.File("scripts/sdk-verify.py")).
		WithFile("/sdk-package.py", source.File("scripts/sdk-package.py")).
		WithFile("/sdk-package-test.py", source.File("scripts/sdk-package-test.py")).
		WithFile("/sdk-verify-test.py", source.File("scripts/sdk-verify-test.py")).
		WithExec([]string{"python3", "/sdk-package-test.py"}).
		WithExec([]string{"python3", "/sdk-verify-test.py", "/out", "--language", "python"}).
		WithExec([]string{"python3", "/sdk-verify.py", "/out", "--report", "/out/sdk-contract-check.json"}).
		Directory("/out")
}

func (m *Xoscal) sdkPackage(source, generated *dagger.Directory, language string) *dagger.Directory {
	prepared := sdkPrepared(source, generated, language, m.Version)
	builder := sdkToolchain(language).
		WithDirectory("/sdk", prepared).
		WithWorkdir("/sdk").
		WithExec([]string{"mkdir", "-p", "/out/packages/" + language}).
		WithExec([]string{"cp", "SDK-VERSION", "/out/sdk-version.txt"})
	switch language {
	case "go":
		builder = builder.WithExec([]string{"go", "mod", "tidy"}).
			// Retain security pins even though a message-only consumer does not
			// import grpc's optional OpenTelemetry instrumentation packages.
			WithExec([]string{"go", "mod", "edit", "-require=go.opentelemetry.io/otel@v1.45.0", "-require=go.opentelemetry.io/otel/metric@v1.45.0", "-require=go.opentelemetry.io/otel/trace@v1.45.0", "-require=go.opentelemetry.io/otel/sdk@v1.45.0", "-require=go.opentelemetry.io/otel/sdk/metric@v1.45.0"}).
			WithExec([]string{"go", "mod", "download"}).
			WithExec([]string{"go", "test", "./..."})
	case "python":
		builder = builder.WithExec([]string{"python3", "-m", "pip", "install", "build==1.4.0"}).
			WithExec([]string{"python3", "-m", "build", "--outdir", "/out/packages/python"})
	case "java":
		builder = builder.WithEnvVariable("MAVEN_OPTS", "-Xmx1g").
			WithExec([]string{"mvn", "--batch-mode", "package"}).
			WithExec([]string{"sh", "-ec", "cp target/xoscal-sdk-*.jar /out/packages/java/; cp pom.xml /out/packages/java/xoscal-sdk.pom"})
	case "csharp":
		builder = builder.WithExec([]string{"dotnet", "pack", "--configuration", "Release", "--output", "/out/packages/csharp"})
	case "ts":
		builder = builder.WithExec([]string{"npm", "install", "--ignore-scripts", "--no-audit", "--no-fund"}).
			WithExec([]string{"npm", "run", "build"}).
			WithExec([]string{"sh", "-ec", "find esm -name '*.d.ts' | while read -r f; do d=cjs/${f#esm/}; mkdir -p \"$(dirname \"$d\")\"; cp \"$f\" \"$d\"; done; printf '{\"type\":\"commonjs\"}\n' > cjs/package.json"}).
			WithExec([]string{"npm", "pack", "--ignore-scripts", "--pack-destination", "/out/packages/ts"})
	case "swift":
		builder = builder.WithExec([]string{"swift", "build", "--jobs", "2"})
	}
	// Only distributable source, metadata and native packages enter the ZIP.
	clean := builder.WithExec([]string{"sh", "-ec", "rm -rf /sdk/.build /sdk/.swiftpm /sdk/node_modules /sdk/build /sdk/dist /sdk/target /sdk/bin /sdk/obj /sdk/*.egg-info; cp -a /out/packages /sdk/packages"}).Directory("/sdk")
	archive := sdkToolchain("python").WithDirectory("/sdk", clean).
		WithDirectory("/out", builder.Directory("/out")).
		WithFile("/sdk-package.py", source.File("scripts/sdk-package.py")).
		WithExec([]string{"python3", "/sdk-package.py", "archive", "/sdk", "/out/" + language + ".zip"})
	consumer := sdkConsumer(source, archive.Directory("/out"), language)
	result := archive.Directory("/out").WithFile("smoke/"+language+".json", consumer.File("/smoke.json"))
	if language == "go" {
		result = result.WithFile("smoke/go-modules.txt", consumer.File("/go-module-graph.txt"))
	}
	return result
}

func sdkConsumer(source, output *dagger.Directory, language string) *dagger.Container {
	consumer := sdkToolchain(language).WithDirectory("/artifacts", output).
		WithDirectory("/checks", source.Directory("scripts/sdk-consumers")).
		WithWorkdir("/consumer")
	switch language {
	case "go":
		consumer = consumer.WithExec([]string{"sh", "-ec", "apt-get update && apt-get install -y --no-install-recommends unzip; unzip -q /artifacts/go.zip -d /module; cp /checks/go.mod /checks/go-main.go .; mv go-main.go main.go; go mod tidy; go run .; go list -m all > /go-module-graph.txt; for suffix in '' /metric /trace /sdk /sdk/metric; do grep -Fx \"go.opentelemetry.io/otel${suffix} v1.45.0\" /go-module-graph.txt; done"})
	case "python":
		consumer = consumer.WithExec([]string{"sh", "-ec", "python3 -m venv /venv; /venv/bin/pip install /artifacts/packages/python/*.whl; /venv/bin/python /checks/python-smoke.py; /venv/bin/pip check; python3 -m venv /sdist-venv; /sdist-venv/bin/pip install /artifacts/packages/python/*.tar.gz; /sdist-venv/bin/python /checks/python-smoke.py; /sdist-venv/bin/pip check"})
	case "java":
		consumer = consumer.WithEnvVariable("MAVEN_OPTS", "-Xmx1g").WithExec([]string{"sh", "-ec", "mvn --batch-mode org.apache.maven.plugins:maven-install-plugin:3.1.4:install-file -Dfile=$(find /artifacts/packages/java -name '*.jar') -DpomFile=/artifacts/packages/java/xoscal-sdk.pom; sed \"s/@VERSION@/$(cat /artifacts/sdk-version.txt)/g\" /checks/java-pom.xml > pom.xml; mkdir -p src/main/java; cp /checks/JavaSmoke.java src/main/java/; mvn --batch-mode package org.apache.maven.plugins:maven-dependency-plugin:3.7.0:build-classpath -Dmdep.outputFile=classpath; java -cp target/classes:$(cat classpath) JavaSmoke"})
	case "csharp":
		consumer = consumer.WithExec([]string{"sh", "-ec", "sed \"s/@VERSION@/$(cat /artifacts/sdk-version.txt)/g\" /checks/CsharpSmoke.csproj > Smoke.csproj; cp /checks/CsharpSmoke.cs Program.cs; dotnet restore --source /artifacts/packages/csharp --source https://api.nuget.org/v3/index.json; dotnet run --no-restore --configuration Release"})
	case "ts":
		consumer = consumer.WithExec([]string{"sh", "-ec", "npm init -y; npm install --ignore-scripts --no-audit --no-fund /artifacts/packages/ts/*.tgz typescript@5.9.3; cp /checks/node-* .; node node-esm.mjs; node node-cjs.cjs; cp node-types.ts node-types.mts; ./node_modules/.bin/tsc --strict --noEmit --module nodenext --moduleResolution nodenext node-types.ts node-types.mts"})
	case "swift":
		consumer = consumer.WithExec([]string{"sh", "-ec", "apt-get update && apt-get install -y --no-install-recommends unzip; unzip -q /artifacts/swift.zip -d /module; cp /checks/SwiftPackage.swift Package.swift; mkdir -p Sources/Smoke; cp /checks/swift-main.swift Sources/Smoke/main.swift; swift run --jobs 2 Smoke"})
	}
	// Written only after consuming succeeds. No network service or hosted signing
	// claim is established by this deliberately local install/import/build gate.
	return consumer.WithExec([]string{"sh", "-ec", "printf '%s' '{\"schema_version\":1,\"language\":\"" + language + "\",\"status\":\"passed\",\"scope\":\"package-install-and-consumer\",\"hosted_proof\":false,\"subjects\":[' > /smoke.json; sep=''; for f in /artifacts/" + language + ".zip /artifacts/packages/" + language + "/*; do [ -f \"$f\" ] || continue; digest=$(sha256sum \"$f\"); digest=${digest%% *}; printf '%s{\"path\":\"%s\",\"sha256\":\"%s\"}' \"$sep\" \"${f#/artifacts/}\" \"$digest\" >> /smoke.json; sep=','; done; printf ']}\n' >> /smoke.json"})
}
