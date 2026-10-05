#!/usr/bin/env python3
"""Prepare package metadata and deterministic SDK archives; never report build success."""
import argparse
import hashlib
import json
import re
import shutil
import stat
import zipfile
from pathlib import Path

LANGUAGES = ("go", "python", "java", "csharp", "ts", "swift")
MODULE = "github.com/mchorfa/xoscal/proto/oscal"
PRODUCER = "ckodex-labs"
REPOSITORY = "https://github.com/ckodex-labs/ckodex-xoscal"
EPOCH = 315532800


def write(root, name, content):
    path = root / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def version(value):
    value = "0.0.0-dev" if value == "dev" else value.removeprefix("v")
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", value):
        raise ValueError("SDK version must be SemVer without build metadata")
    return value


def python_version(value):
    """Preserve a SemVer prerelease identity in a valid PEP 440 version."""
    value = version(value)
    base, separator, prerelease = value.partition("-")
    if not separator:
        return base
    if prerelease == "dev":
        return base + ".dev0"
    match = re.fullmatch(r"(alpha|beta|rc)\.?([0-9]+)", prerelease)
    if match:
        return base + {"alpha": "a", "beta": "b", "rc": "rc"}[match[1]] + match[2]
    return base + ".dev0+" + prerelease.replace("-", ".")


def producer_metadata(language, value):
    names = {"go": MODULE, "python": "xoscal-sdk", "java": "io.ckodex:xoscal-sdk",
             "csharp": "Xoscal.SDK", "ts": "@ckodex/xoscal-sdk", "swift": "XoscalSDK"}
    return {"schema_version": 1, "role": "artifact-producer",
            "entity": {"name": PRODUCER, "url": [REPOSITORY]}, "language": language,
            "release_version": value,
            "package": {"name": names[language],
                        "version": python_version(value) if language == "python" else value}}


def prepare(generated, destination, language, value):
    value = version(value)
    src = generated / language
    if not src.is_dir():
        raise ValueError(f"missing generated language: {language}")
    expected = {"go": ".pb.go", "python": "_pb2.py", "java": ".java",
                "csharp": ".cs", "ts": ".js", "swift": ".pb.swift"}[language]
    files = sorted(p for p in src.rglob("*") if p.is_file() and p.name.endswith(expected))
    if not files:
        raise ValueError(f"empty generated language: {language}")
    if any(p.is_symlink() for p in src.rglob("*")):
        raise ValueError("generated sources contain a symlink")
    root = destination
    root.mkdir(parents=True, exist_ok=True)
    write(root, "SDK-VERSION", value + "\n")
    write(root, "SDK-PRODUCER.json", json.dumps(producer_metadata(language, value),
                                               sort_keys=True, indent=2) + "\n")
    layouts = {"go": "", "python": "xoscal_sdk", "java": "src/main/java",
               "csharp": "", "ts": "esm", "swift": "Sources/XoscalSDK"}
    target = root / layouts[language]
    # Only generator outputs are copied. No stale files, pyc, caches or binaries.
    for p in sorted(src.rglob("*")):
        if not p.is_file() or not (p.name.endswith(expected) or
                                  (language == "python" and p.name.endswith("_pb2_grpc.py")) or
                                  (language == "ts" and p.name.endswith(".d.ts"))):
            continue
        dest = target / p.relative_to(src)
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(p, dest)
    write(root, "SDK-CONTRACT.md", f"""# xOSCAL {language} SDK {value}

Generated from the release's pinned proto sources by Dagger SdkPackages.
See the release SDK-CONTRACT.md for install commands and supported transports.
The outer release manifest covers this archive; SHA sidecars alone are not signatures.
""")
    if language == "go":
        write(root, "go.mod", f"""module {MODULE}

go 1.25.0

require (
 google.golang.org/grpc v1.83.2
 google.golang.org/protobuf v1.36.12
 google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa
 go.opentelemetry.io/otel v1.45.0
 go.opentelemetry.io/otel/metric v1.45.0
 go.opentelemetry.io/otel/trace v1.45.0
 go.opentelemetry.io/otel/sdk v1.45.0
 go.opentelemetry.io/otel/sdk/metric v1.45.0
)
""")
    elif language == "python":
        namespaces = {p.parts[0] for p in (p.relative_to(src) for p in files)}
        for p in target.rglob("*.py"):
            text = p.read_text()
            text = re.sub(r"^(from|import) ([A-Za-z_][A-Za-z0-9_]*)(?=[. ])",
                          lambda m: m[1] + " " + ("xoscal_sdk." if m[2] in namespaces else "") + m[2],
                          text, flags=re.MULTILINE)
            module = ".".join(p.relative_to(target).with_suffix("").parts)
            text = text.replace("'" + module + "'", "'xoscal_sdk." + module + "'")
            p.write_text(text)
        for d in [target] + [p for p in target.rglob("*") if p.is_dir()]:
            write(d, "__init__.py", "")
        # Python cannot spell a generic SemVer prerelease verbatim. Keep its
        # identity in a PEP 440 local version rather than dropping it.
        pyversion = python_version(value)
        write(root, "pyproject.toml", f"""[build-system]
requires = ["setuptools==80.10.2", "wheel==0.46.3"]
build-backend = "setuptools.build_meta"
[project]
name = "xoscal-sdk"
version = "{pyversion}"
requires-python = ">=3.10"
description = "xOSCAL protobuf messages and gRPC clients"
authors = [{{name = "{PRODUCER}"}}]
license = "Apache-2.0"
dependencies = ["protobuf==6.33.5", "grpcio==1.83.1", "googleapis-common-protos==1.73.0", "typing-extensions==4.16.0"]
[project.urls]
Repository = "{REPOSITORY}"
[tool.setuptools.packages.find]
include = ["xoscal_sdk*"]
""")
        write(root, "requirements.txt", "protobuf==6.33.5\ngrpcio==1.83.1\ngoogleapis-common-protos==1.73.0\ntyping-extensions==4.16.0\n")
    elif language == "java":
        write(root, "pom.xml", f"""<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
 <modelVersion>4.0.0</modelVersion><groupId>io.ckodex</groupId><artifactId>xoscal-sdk</artifactId><version>{value}</version>
 <url>{REPOSITORY}</url><organization><name>{PRODUCER}</name><url>https://github.com/ckodex-labs</url></organization>
 <developers><developer><id>ckodex-labs</id><name>{PRODUCER}</name><url>https://github.com/ckodex-labs</url></developer></developers>
 <scm><url>{REPOSITORY}</url></scm>
 <licenses><license><name>Apache License 2.0</name><url>https://www.apache.org/licenses/LICENSE-2.0</url><distribution>repo</distribution></license></licenses>
 <properties><maven.compiler.release>17</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding><project.build.outputTimestamp>1980-01-01T00:00:02Z</project.build.outputTimestamp></properties>
 <dependencies>
  <dependency><groupId>com.google.protobuf</groupId><artifactId>protobuf-java</artifactId><version>4.36.1</version></dependency>
  <dependency><groupId>com.google.api.grpc</groupId><artifactId>proto-google-common-protos</artifactId><version>2.68.0</version></dependency>
  <dependency><groupId>io.grpc</groupId><artifactId>grpc-protobuf</artifactId><version>1.84.0</version></dependency>
  <dependency><groupId>io.grpc</groupId><artifactId>grpc-stub</artifactId><version>1.84.0</version></dependency>
  <dependency><groupId>javax.annotation</groupId><artifactId>javax.annotation-api</artifactId><version>1.3.2</version></dependency>
 </dependencies>
 <build><plugins>
  <plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-compiler-plugin</artifactId><version>3.14.1</version></plugin>
  <plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-jar-plugin</artifactId><version>3.4.2</version></plugin>
 </plugins></build>
</project>
""")
    elif language == "csharp":
        write(root, "XoscalSDK.csproj", f"""<Project Sdk="Microsoft.NET.Sdk">
 <PropertyGroup><TargetFramework>net8.0</TargetFramework><PackageId>Xoscal.SDK</PackageId><Version>{value}</Version><Authors>{PRODUCER}</Authors><PackageProjectUrl>{REPOSITORY}</PackageProjectUrl><RepositoryUrl>{REPOSITORY}</RepositoryUrl><RepositoryType>git</RepositoryType><Description>xOSCAL protobuf models and four generated gRPC clients</Description><PackageLicenseExpression>Apache-2.0</PackageLicenseExpression><RestorePackagesWithLockFile>true</RestorePackagesWithLockFile><Deterministic>true</Deterministic><ContinuousIntegrationBuild>true</ContinuousIntegrationBuild></PropertyGroup>
 <ItemGroup><PackageReference Include="Google.Protobuf" Version="3.36.1" /><PackageReference Include="Google.Api.CommonProtos" Version="2.17.0" /><PackageReference Include="Grpc.Core.Api" Version="2.76.0" /><PackageReference Include="Grpc.Net.Client" Version="2.76.0" /></ItemGroup>
</Project>
""")
    elif language == "ts":
        package = {"name": "@ckodex/xoscal-sdk", "version": value, "type": "module",
                   "license": "Apache-2.0",
                   "author": {"name": PRODUCER, "url": "https://github.com/ckodex-labs"},
                   "repository": {"type": "git", "url": REPOSITORY},
                   "description": "xOSCAL typed protobuf messages and service descriptors",
                   "files": ["esm", "cjs", "SDK-CONTRACT.md", "SDK-PRODUCER.json"],
                   "exports": {"./*": {"import": {"types": "./esm/*.d.ts", "default": "./esm/*.js"},
                                      "require": {"types": "./cjs/*.d.ts", "default": "./cjs/*.js"}}},
                   "dependencies": {"@bufbuild/protobuf": "2.14.1"},
                   "devDependencies": {"typescript": "5.9.3"},
                   "scripts": {"build": "tsc -p tsconfig.cjs.json"}}
        write(root, "package.json", json.dumps(package, indent=2) + "\n")
        write(root, "tsconfig.cjs.json", json.dumps({"compilerOptions": {"allowJs": True, "checkJs": False,
             "target": "ES2020", "module": "CommonJS", "moduleResolution": "Node", "outDir": "cjs",
             "rootDir": "esm", "skipLibCheck": False}, "include": ["esm/**/*.js"]}) + "\n")
    elif language == "swift":
        write(root, "Package.swift", """// swift-tools-version: 6.0
import PackageDescription
let package = Package(
 name: "XoscalSDK",
 platforms: [.macOS(.v13), .iOS(.v16)],
 products: [.library(name: "XoscalSDK", targets: ["XoscalSDK"])],
 dependencies: [.package(url: "https://github.com/apple/swift-protobuf.git", exact: "1.38.1")],
 targets: [.target(name: "XoscalSDK", dependencies: [.product(name: "SwiftProtobuf", package: "swift-protobuf")])]
)
""")


def archive(source, output):
    """Stable ZIP bytes: lexical order, fixed epoch/mode, no directory entries."""
    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for path in sorted(source.rglob("*")):
            if path.is_symlink():
                raise ValueError("archive source contains a symlink")
            if not path.is_file():
                continue
            name = path.relative_to(source).as_posix()
            info = zipfile.ZipInfo(name, (1980, 1, 1, 0, 0, 0))
            info.create_system = 3
            info.external_attr = (stat.S_IFREG | 0o644) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
    output.with_suffix(output.suffix + ".sha256").write_text("sha256:" + hashlib.sha256(output.read_bytes()).hexdigest() + "\n")


def extract(source, destination):
    with zipfile.ZipFile(source) as z:
        for item in z.infolist():
            p = Path(item.filename)
            if p.is_absolute() or ".." in p.parts or "\\" in item.filename or stat.S_ISLNK(item.external_attr >> 16):
                raise ValueError("unsafe ZIP member")
        z.extractall(destination)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("prepare")
    p.add_argument("generated", type=Path)
    p.add_argument("destination", type=Path)
    p.add_argument("language", choices=LANGUAGES)
    p.add_argument("version")
    p = sub.add_parser("archive")
    p.add_argument("source", type=Path)
    p.add_argument("output", type=Path)
    p = sub.add_parser("extract")
    p.add_argument("source", type=Path)
    p.add_argument("destination", type=Path)
    args = parser.parse_args()
    if args.command == "prepare":
        prepare(args.generated, args.destination, args.language, args.version)
    elif args.command == "archive":
        archive(args.source, args.output)
    else:
        extract(args.source, args.destination)


if __name__ == "__main__":
    main()
