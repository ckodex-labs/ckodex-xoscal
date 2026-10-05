package main

import (
	"fmt"

	"dagger/xoscal/internal/dagger"
)

func (m *Xoscal) Security(source *dagger.Directory) *dagger.File {
	return m.SecurityAnalysis(source).File("gosec-results.sarif")
}

// SecurityAnalysis returns all admitted source reports and its policy verdict.
func (m *Xoscal) SecurityAnalysis(source *dagger.Directory) *dagger.Directory {
	return m.analysisGate(source, m.SecurityReports(source), "security").Directory("/evidence")
}

// SecurityReports collects raw results without admitting them for publication.
// Export this directory separately on gate failure so findings and tool errors
// remain reviewable. Each scanner status is evaluated by analysis-gate.py.
func (m *Xoscal) SecurityReports(source *dagger.Directory) *dagger.Directory {
	return m.toolBase().
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", "mkdir -p /evidence\nset +e\ngovulncheck -json ./... > /evidence/govulncheck.json 2> /evidence/govulncheck.log\nprintf '%s\\n' \"$?\" > /evidence/govulncheck.status\ngosec -track-suppressions -concurrency=2 -fmt json -out /evidence/gosec-results.json -exclude-dir=proto ./server/... > /evidence/gosec.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/gosec.status\ngosec -track-suppressions -concurrency=2 -fmt sarif -out /evidence/gosec-results.sarif -exclude-dir=proto ./server/... > /evidence/gosec-sarif.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/gosec-sarif.status"}).
		Directory("/evidence")
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
	return m.enrichSbomReports(source, m.ArtifactSbomReports(m.Build(source))).File("sbom.cyclonedx.json")
}

// BinarySbomAnalysis keeps original byte-bound production and assessment evidence.
func (m *Xoscal) BinarySbomAnalysis(source *dagger.Directory) *dagger.Directory {
	return m.ArtifactSbomAnalysis(source, m.Build(source))
}

// SbomValidation validates the CycloneDX SBOM with the pinned sbom-tools
// release and emits an OSCAL 1.1.2 assessment-results document. The release
// binary is downloaded for the target architecture and checked against the
// published SHA-256 digest; compiling the Rust tool is not part of the site
// verification critical path.
func (m *Xoscal) SbomValidation(source *dagger.Directory) *dagger.File {
	return m.BinarySbomAnalysis(source).File("sbom-assessment-results.oscal.json")
}

// SbomArtifactValidation admits only an assessed exact SBOM supplied by its
// producer (binary, archive, or image). A report's existence is never a pass.
func (m *Xoscal) SbomArtifactValidation(source *dagger.Directory, sbom *dagger.File) *dagger.File {
	return m.SbomArtifactAnalysis(source, sbom).File("sbom-assessment-results.oscal.json")
}

// SbomArtifactAnalysis returns the entire admitted evidence directory including
// the input SBOM, raw OSCAL report, exit status, log and admission summary.
func (m *Xoscal) SbomArtifactAnalysis(source *dagger.Directory, sbom *dagger.File) *dagger.Directory {
	return m.analysisGate(source, m.SbomValidationReports(source, sbom), "sbom").Directory("/evidence")
}

// SbomValidationReports preserves input bytes, raw findings and exit status.
func (m *Xoscal) SbomValidationReports(source *dagger.Directory, sbom *dagger.File) *dagger.Directory {
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
		WithFile("/evidence/sbom.cyclonedx.json", sbom).
		WithExec([]string{"sh", "-c", "set +e\n/opt/sbom-tools/sbom-tools validate /evidence/sbom.cyclonedx.json --standard ntia -o oscal-json -O /evidence/sbom-assessment-results.oscal.json > /evidence/sbom-validation.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/sbom-validation.status"}).
		Directory("/evidence")
}

// Dist packages the binary and SBOM into a directory.
