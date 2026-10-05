package main

import (
	"fmt"
	"strings"

	"dagger/xoscal/internal/dagger"
)

// Version-specific CLI contracts and official release checksums are recorded
// in ANALYSIS-POLICY.md. Building the scanner is outside the release hot path.
const trivyVersion = "v0.72.0"

func pinnedTrivyInstall() string {
	return fmt.Sprintf("set -eu\n"+
		"case \"$(uname -m)\" in\n"+
		"  x86_64) arch=64bit; expected=bbb64b9695866ce4a7a8f5c9592002c5961cab378577fa3f8a040df362b9b2ea ;;\n"+
		"  aarch64|arm64) arch=ARM64; expected=2ca2c023109c2db6b2b77366b6717291452d4531167377d95c79547f0c8e3467 ;;\n"+
		"  *) echo unsupported Trivy architecture >&2; exit 1 ;;\n"+
		"esac\n"+
		"curl -fsSL https://github.com/aquasecurity/trivy/releases/download/%s/trivy_%s_Linux-${arch}.tar.gz -o /tmp/trivy.tar.gz\n"+
		"printf '%%s  %%s\\n' \"$expected\" /tmp/trivy.tar.gz | sha256sum -c -\n"+
		"tar -xzf /tmp/trivy.tar.gz -C /usr/local/bin trivy\n"+
		"trivy --version | head -n 1 | grep -Fx 'Version: %s'\n", trivyVersion, strings.TrimPrefix(trivyVersion, "v"), strings.TrimPrefix(trivyVersion, "v"))
}

func (m *Xoscal) analysisGate(source *dagger.Directory, evidence *dagger.Directory, kind string) *dagger.Container {
	return dag.Container().From(goBuilderImage).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "python3"}).
		WithFile("/policy/analysis-gate.py", source.File("scripts/analysis-gate.py")).
		WithFile("/policy/frontend_inventory.py", dag.CurrentModule().Source().File("frontend_inventory.py")).
		WithFile("/policy/enrich_sbom.py", dag.CurrentModule().Source().File("enrich_sbom.py")).
		WithFile("/policy/publisher_metadata.py", dag.CurrentModule().Source().File("publisher_metadata.py")).
		WithFile("/policy/analysis-exceptions.json", source.File("docs/analysis-exceptions.json")).
		WithDirectory("/evidence", evidence).
		WithExec([]string{"python3", "/policy/analysis-gate.py", kind, "--evidence", "/evidence", "--exceptions", "/policy/analysis-exceptions.json", "--output", "/evidence/analysis-gate.json"})
}

// ImageAnalysisReports scans a locally staged image archive before promotion.
// All severities and unfixed findings remain in the raw report. The release
// gate retains the existing CRITICAL/fixed threshold; CI also blocks HIGH.
func (m *Xoscal) ImageAnalysisReports(archive *dagger.File) *dagger.Directory {
	return m.trivyBase().
		WithFile("/input/image.tar", archive).
		WithExec([]string{"sh", "-c", "mkdir -p /evidence\nsha256sum /input/image.tar > /evidence/image.sha256\nset +e\ntrivy image --input /input/image.tar --scanners vuln --format json --list-all-pkgs --exit-code 0 --output /evidence/trivy-results.json > /evidence/trivy.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/trivy.status"}).
		Directory("/evidence")
}

func (m *Xoscal) trivyBase() *dagger.Container {
	return dag.Container().From(goBuilderImage).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "python3"}).
		WithMountedCache("/root/.cache/trivy", dag.CacheVolume("trivy-db")).
		WithEnvVariable("GOMAXPROCS", "2").
		WithExec([]string{"sh", "-c", pinnedTrivyInstall()})
}

// ArtifactAnalysisReports analyzes safe extracted ZIP/tar package bytes. Empty
// or unsupported package inventory is rejected by ArtifactAnalysis admission.
func (m *Xoscal) ArtifactAnalysisReports(archive *dagger.File) *dagger.Directory {
	return m.trivyBase().
		WithFile("/input/artifact", archive).
		WithFile("/tools/extract_archive.py", dag.CurrentModule().Source().File("extract_archive.py")).
		WithExec([]string{"sh", "-c", "mkdir -p /evidence\nsha256sum /input/artifact > /evidence/artifact.sha256\nset +e\npython3 /tools/extract_archive.py /input/artifact /input/unpacked > /evidence/extraction.log 2>&1\nextract_status=$?\nprintf '%s\\n' \"$extract_status\" > /evidence/extraction.status\nif [ \"$extract_status\" -eq 0 ]; then\n  trivy fs /input/unpacked --scanners vuln --format json --list-all-pkgs --exit-code 0 --output /evidence/trivy-results.json > /evidence/trivy.log 2>&1\n  scan_status=$?\nelse\n  scan_status=125\nfi\nprintf '%s\\n' \"$scan_status\" > /evidence/trivy.status"}).Directory("/evidence")
}

// ArtifactAnalysis applies the existing release CRITICAL/fixed threshold to
// the exact staged SDK/package or platform binary archive.
func (m *Xoscal) ArtifactAnalysis(source *dagger.Directory, archive *dagger.File) *dagger.Directory {
	return m.analysisGate(source, m.ArtifactAnalysisReports(archive), "artifact-release").Directory("/evidence")
}

// ImageAnalysis enforces the release vulnerability policy for exact staged bytes.
func (m *Xoscal) ImageAnalysis(source *dagger.Directory, archive *dagger.File) *dagger.Directory {
	return m.analysisGate(source, m.ImageAnalysisReports(archive), "image-release").Directory("/evidence")
}

// ImageAnalysisCi enforces the existing HIGH/CRITICAL CI threshold.
func (m *Xoscal) ImageAnalysisCi(source *dagger.Directory, archive *dagger.File) *dagger.Directory {
	return m.analysisGate(source, m.ImageAnalysisReports(archive), "image-ci").Directory("/evidence")
}

// ArtifactSbom generates an SBOM directly from supplied artifact bytes, instead
// of rebuilding a nominally equivalent binary. Archives are safely extracted
// so source dependency files and compiled package metadata are inventoried.
func (m *Xoscal) ArtifactSbom(artifact *dagger.File) *dagger.File {
	return m.ArtifactSbomReports(artifact).File("sbom.cyclonedx.json")
}

// ArtifactSbomReports binds the exact original archive/binary digest to the
// extracted dependency inventory without reconstructing the original payload.
func (m *Xoscal) ArtifactSbomReports(artifact *dagger.File) *dagger.Directory {
	return m.syftBase().
		WithFile("/input/artifact", artifact).
		WithFile("/tools/extract_archive.py", dag.CurrentModule().Source().File("extract_archive.py")).
		WithExec([]string{"python3", "/tools/extract_archive.py", "/input/artifact", "/input/unpacked", "--allow-file"}).
		WithExec([]string{"sh", "-c", "mkdir -p /evidence; sha256sum /input/artifact | cut -d ' ' -f 1 > /evidence/subject.sha256; syft version > /evidence/sbom-producer.txt"}).
		WithExec([]string{"syft", "scan", "dir:/input/unpacked", "-o", "cyclonedx-json", "-q", "--file", "/evidence/sbom.cyclonedx.json"}).
		WithExec([]string{"sh", "-c", "for pair in SDK-PRODUCER.json:sdk-producer.json go.sum:source-go.sum Package.resolved:source-Package.resolved requirements.txt:source-requirements.txt; do src=${pair%:*}; dst=${pair#*:}; if [ -f /input/unpacked/$src ]; then cp /input/unpacked/$src /evidence/$dst; fi; done"}).Directory("/evidence")
}

// ArtifactSbomAnalysis requires compliance of the exact generated package SBOM
// and returns original payload digest, SBOM, assessment and policy receipt.
func (m *Xoscal) ArtifactSbomAnalysis(source *dagger.Directory, artifact *dagger.File) *dagger.Directory {
	return m.boundSbomAnalysis(source, m.ArtifactSbomReports(artifact))
}

func (m *Xoscal) boundSbomAnalysis(source *dagger.Directory, produced *dagger.Directory) *dagger.Directory {
	produced = m.enrichSbomReports(source, produced)
	evidence := m.SbomValidationReports(source, produced.File("sbom.cyclonedx.json")).
		WithFile("sbom.raw.cyclonedx.json", produced.File("sbom.raw.cyclonedx.json")).
		WithFile("sbom-enrichment.json", produced.File("sbom-enrichment.json")).
		WithFile("sbom-distributor.json", produced.File("sbom-distributor.json")).
		WithDirectory("publishers", produced.Directory("publishers")).
		WithFile("subject.sha256", produced.File("subject.sha256")).
		WithFile("sbom-producer.txt", produced.File("sbom-producer.txt"))
	return m.analysisGate(source, evidence, "sbom").Directory("/evidence")
}

// ImageSbom inventories the exact staged image archive.
func (m *Xoscal) ImageSbom(archive *dagger.File) *dagger.File {
	return m.ImageSbomReports(archive).File("sbom.cyclonedx.json")
}

// ImageSbomReports binds image SBOM inventory to the exact original image tar.
func (m *Xoscal) ImageSbomReports(archive *dagger.File) *dagger.Directory {
	return m.syftBase().
		WithFile("/input/image.tar", archive).
		WithExec([]string{"sh", "-c", "mkdir -p /evidence; sha256sum /input/image.tar | cut -d ' ' -f 1 > /evidence/subject.sha256; syft version > /evidence/sbom-producer.txt"}).
		WithExec([]string{"syft", "scan", "docker-archive:/input/image.tar", "-o", "cyclonedx-json", "-q", "--file", "/evidence/sbom.cyclonedx.json"}).Directory("/evidence")
}

// ImageSbomAnalysis returns only compliant image inventory with original digest.
func (m *Xoscal) ImageSbomAnalysis(source *dagger.Directory, archive *dagger.File) *dagger.Directory {
	return m.boundSbomAnalysis(source, m.ImageSbomReports(archive))
}

func (m *Xoscal) syftBase() *dagger.Container {
	return dag.Container().From(goBuilderImage).
		WithEnvVariable("GOMAXPROCS", "2").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "ca-certificates", "curl", "python3"}).
		WithExec([]string{"sh", "-c", pinnedSyftInstall()})
}

// enrichSbomReports records the distributor only for observed packaged bytes.
// Original scanner evidence and exact-subject receipt survive admission.
func (m *Xoscal) enrichSbomReports(source *dagger.Directory, produced *dagger.Directory) *dagger.Directory {
	return dag.Container().From(goBuilderImage).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "python3"}).
		WithDirectory("/evidence", produced).
		WithFile("/evidence/sbom.raw.cyclonedx.json", produced.File("sbom.cyclonedx.json")).
		WithFile("/evidence/sbom-distributor.json", source.File("docs/SBOM-DISTRIBUTOR.json")).
		WithFile("/tools/enrich_sbom.py", dag.CurrentModule().Source().File("enrich_sbom.py")).
		WithFile("/tools/publisher_metadata.py", dag.CurrentModule().Source().File("publisher_metadata.py")).
		WithExec([]string{"python3", "/tools/publisher_metadata.py", "--sbom", "/evidence/sbom.raw.cyclonedx.json", "--output", "/evidence/publishers", "--producer", "/evidence/sdk-producer.json", "--go-sum", "/evidence/source-go.sum", "--resolved", "/evidence/source-Package.resolved", "--requirements", "/evidence/source-requirements.txt"}).
		WithExec([]string{"python3", "/tools/enrich_sbom.py", "--raw", "/evidence/sbom.raw.cyclonedx.json", "--policy", "/evidence/sbom-distributor.json", "--subject", "/evidence/subject.sha256", "--output", "/evidence/sbom.cyclonedx.json", "--receipt", "/evidence/sbom-enrichment.json", "--publisher-dir", "/evidence/publishers"}).Directory("/evidence")
}

// ArtifactSbomProducedReports preserves the enriched producer evidence before admission.
func (m *Xoscal) ArtifactSbomProducedReports(source *dagger.Directory, artifact *dagger.File) *dagger.Directory {
	return m.enrichSbomReports(source, m.ArtifactSbomReports(artifact))
}

// ImageSbomProducedReports preserves real unknown metadata and raw assessment inputs.
func (m *Xoscal) ImageSbomProducedReports(source *dagger.Directory, archive *dagger.File) *dagger.Directory {
	return m.enrichSbomReports(source, m.ImageSbomReports(archive))
}
