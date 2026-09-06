package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

func (m *Xoscal) Security(source *dagger.Directory) *dagger.File {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"govulncheck", "./..."}).
		WithExec([]string{"gosec", "-concurrency=2", "-fmt", "sarif", "-out", "gosec-results.sarif", "-exclude-dir=proto", "-no-fail", "./server/..."}).
		File("/src/gosec-results.sarif")
}

// Image builds a distroless container image and returns it.
func (m *Xoscal) Image(source *dagger.Directory) *dagger.Container {
	bin := m.Build(source)
	return dag.Container().
		From(distrolessBase).
		WithWorkdir("/data").
		WithFile("/xoscal-server", bin).
		WithExposedPort(50051).
		WithExposedPort(9090).
		WithEntrypoint([]string{"/xoscal-server"}).
		WithDefaultArgs([]string{"-config", "/etc/xoscal/config.yaml"})
}

// Sbom generates a CycloneDX SBOM for the built binary using syft.
func (m *Xoscal) Sbom(source *dagger.Directory) *dagger.File {
	bin := m.Build(source)
	return m.toolBase().
		WithExec([]string{"sh", "-c", pinnedSyftInstall()}).
		WithFile("/xoscal-server", bin).
		WithExec([]string{"syft", "packages", "file:/xoscal-server", "-o", "cyclonedx-json", "-q", "--file", "/tmp/sbom.json"}).
		File("/tmp/sbom.json")
}

// SbomValidation validates the CycloneDX SBOM with the pinned sbom-tools
// release and emits an OSCAL 1.1.2 assessment-results document. The release
// binary is downloaded for the target architecture and checked against the
// published SHA-256 digest; compiling the Rust tool is not part of the site
// verification critical path.
func (m *Xoscal) SbomValidation(source *dagger.Directory) *dagger.File {
	sbom := m.Sbom(source)
	return dag.Container().
		From(ubuntuBase).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "coreutils", "tar"}).
		WithExec([]string{"sh", "-c", fmt.Sprintf("set -eu\n"+
			"case \"$(uname -m)\" in\n"+
			"  x86_64) asset_arch=x86_64; expected=%s ;;\n"+
			"  aarch64) asset_arch=aarch64; expected=%s ;;\n"+
			"  *) echo \"unsupported sbom-tools architecture: $(uname -m)\" >&2; exit 1 ;;\n"+
			"esac\n"+
			"archive=\"/tmp/sbom-tools-${asset_arch}.tar.gz\"\n"+
			"curl -fsSL \"https://github.com/sbom-tool/sbom-tools/releases/download/%s/sbom-tools-linux-${asset_arch}.tar.gz\" -o \"$archive\"\n"+
			"printf '%%s  %%s\\n' \"$expected\" \"$archive\" | sha256sum -c -\n"+
			"mkdir -p /opt/sbom-tools\n"+
			"tar -xzf \"$archive\" -C /opt/sbom-tools\n"+
			"test -x /opt/sbom-tools/sbom-tools\n", sbomToolsLinuxAMD64SHA, sbomToolsLinuxARM64SHA, sbomToolsVersion)}).
		WithFile("/tmp/sbom.cyclonedx.json", sbom).
		// sbom-tools validate exits non-zero when it finds violations — that's
		// the expected case (we WANT the findings). The OSCAL assessment-results
		// file is still written. Wrap so we don't fail the build; just verify
		// the output file exists and is non-empty.
		WithExec([]string{"sh", "-c", "/opt/sbom-tools/sbom-tools validate /tmp/sbom.cyclonedx.json --standard ntia -o oscal-json -O /tmp/sbom-assessment-results.oscal.json; test -s /tmp/sbom-assessment-results.oscal.json"}).
		File("/tmp/sbom-assessment-results.oscal.json")
}

// Dist packages the binary and SBOM into a directory.
