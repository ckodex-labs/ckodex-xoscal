#!/usr/bin/env python3
"""Recheck consumer subject digests against final SDK output before promotion."""
import argparse
import hashlib
import importlib.util
import json
import stat
import tarfile
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

spec = importlib.util.spec_from_file_location("sdk_package", Path(__file__).with_name("sdk-package.py"))
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)

LANGUAGES = ("go", "python", "java", "csharp", "ts", "swift")
NATIVE = {"go": [], "python": ["*.whl", "*.tar.gz"], "java": ["*.jar", "*.pom"],
          "csharp": ["*.nupkg"], "ts": ["*.tgz"], "swift": []}
METADATA = {"go": "go.mod", "python": "pyproject.toml", "java": "pom.xml",
            "csharp": "XoscalSDK.csproj", "ts": "package.json", "swift": "Package.swift"}


def digest(path):
    if not path.is_file() or path.is_symlink():
        raise ValueError(f"missing or unsafe SDK output: {path}")
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify(root, languages=LANGUAGES):
    release_version = (root / "sdk-version.txt").read_text().strip()
    if package.version(release_version) != release_version:
        raise ValueError("noncanonical SDK release version")
    subjects = []
    for language in languages:
        archive = root / (language + ".zip")
        archive_digest = digest(archive)
        if (root / (language + ".zip.sha256")).read_text().strip() != "sha256:" + archive_digest:
            raise ValueError(f"SHA sidecar mismatch: {language}")
        paths = {language + ".zip"}
        native = []
        for pattern in NATIVE[language]:
            matches = list((root / "packages" / language).glob(pattern))
            if len(matches) != 1:
                raise ValueError(f"expected exactly one native package {language}/{pattern}")
            native.extend(matches)
            paths.add(matches[0].relative_to(root).as_posix())
        report = json.loads((root / "smoke" / (language + ".json")).read_text())
        if type(report.get("schema_version")) is not int or report.get("hosted_proof") is not False or (report.get("schema_version"), report.get("language"), report.get("status"), report.get("scope")) != (1, language, "passed", "package-install-and-consumer"):
            raise ValueError(f"missing consumer verdict: {language}")
        actual = report.get("subjects", [])
        if not isinstance(actual, list) or len(actual) != len(paths) or {s.get("path") for s in actual} != paths:
            raise ValueError(f"consumer subject coverage differs: {language}")
        for subject in actual:
            if digest(root / subject["path"]) != subject.get("sha256"):
                raise ValueError(f"consumer subject tampered: {subject['path']}")
            subjects.append(subject)
        with zipfile.ZipFile(archive) as z:
            names = z.namelist()
            if len(names) != len(set(names)):
                raise ValueError(f"duplicate ZIP member: {language}")
            for info in z.infolist():
                name = Path(info.filename)
                if name.is_absolute() or ".." in name.parts or "\\" in info.filename or stat.S_ISLNK(info.external_attr >> 16):
                    raise ValueError(f"unsafe ZIP member: {language}")
            if METADATA[language] not in names or "LICENSE" not in names:
                raise ValueError(f"missing package metadata/license: {language}")
            if z.read("SDK-VERSION").decode().strip() != release_version:
                raise ValueError(f"mixed SDK source version: {language}")
            producer = json.loads(z.read("SDK-PRODUCER.json"))
            if producer != package.producer_metadata(language, release_version):
                raise ValueError(f"missing declared SDK producer metadata: {language}")
            if language == "swift" and "Package.resolved" not in names:
                raise ValueError("Swift source package lacks its resolved dependency lock")
            if language == "csharp" and "packages.lock.json" not in names:
                raise ValueError("NuGet source package lacks its resolved dependency lock")
            if language == "ts" and "package-lock.json" not in names:
                raise ValueError("Node source package lacks its resolved dependency lock")
            for path in native:
                name = path.relative_to(root).as_posix()
                if name not in names or z.read(name) != path.read_bytes():
                    raise ValueError(f"ZIP/native package bytes differ: {name}")
        if language == "python":
            wheel = next(p for p in native if p.suffix == ".whl")
            with zipfile.ZipFile(wheel) as z:
                names = [n for n in z.namelist() if n.endswith(".dist-info/METADATA")]
                if len(names) != 1 or "Version: " + package.python_version(release_version) not in z.read(names[0]).decode().splitlines():
                    raise ValueError("Python wheel version differs from release")
            sdist = next(p for p in native if p.name.endswith(".tar.gz"))
            with tarfile.open(sdist, "r:gz") as archive:
                names = [n for n in archive.getnames() if n.endswith("/PKG-INFO") and n.count("/") == 1]
                if len(names) != 1 or "Version: " + package.python_version(release_version) not in archive.extractfile(names[0]).read().decode().splitlines():
                    raise ValueError("Python sdist version differs from release")
        elif language == "java":
            pom = next(p for p in native if p.suffix == ".pom")
            declared = ET.fromstring(pom.read_bytes()).find("{http://maven.apache.org/POM/4.0.0}version")
            if declared is None or declared.text != release_version:
                raise ValueError("Maven POM version differs from release")
            jar = next(p for p in native if p.suffix == ".jar")
            with zipfile.ZipFile(jar) as z:
                properties = z.read("META-INF/maven/io.ckodex/xoscal-sdk/pom.properties").decode().splitlines()
                if "version=" + release_version not in properties:
                    raise ValueError("Java JAR version differs from release")
        elif language == "csharp":
            with zipfile.ZipFile(native[0]) as z:
                names = [n for n in z.namelist() if n.endswith(".nuspec")]
                if len(names) != 1:
                    raise ValueError("NuGet package has no unique nuspec")
                metadata = ET.fromstring(z.read(names[0]))
                declared = next((v.text for v in metadata.iter() if v.tag.split("}")[-1] == "version"), None)
                if declared != release_version:
                    raise ValueError("NuGet version differs from release")
        elif language == "ts":
            with tarfile.open(native[0], "r:gz") as archive:
                declared = json.load(archive.extractfile("package/package.json"))["version"]
                if declared != release_version:
                    raise ValueError("npm version differs from release")
        elif language == "go":
            modules = (root / "smoke/go-modules.txt").read_text().splitlines()
            for suffix in ["", "/metric", "/trace", "/sdk", "/sdk/metric"]:
                if "go.opentelemetry.io/otel" + suffix + " v1.45.0" not in modules:
                    raise ValueError("consumer Go module graph lacks fixed OTel family")
    return {"schema_version": 1, "status": "passed", "scope": "final-sdk-consumer-subject-integrity",
            "version": release_version, "hosted_proof": False, "subjects": sorted(subjects, key=lambda s: s["path"])}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--language", choices=LANGUAGES, action="append")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args()
    result = verify(args.directory, args.language or LANGUAGES)
    content = json.dumps(result, sort_keys=True, indent=2) + "\n"
    if args.report:
        args.report.write_text(content)
    else:
        print(content, end="")


if __name__ == "__main__":
    main()
