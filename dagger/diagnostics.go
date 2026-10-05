package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"dagger/xoscal/internal/dagger"
)

// CandidateSubjectReports exports unadmitted failure evidence for one exact
// production subject. Reports never grant release eligibility. Independent
// calls preserve other subjects when a producer or diagnostic tool fails.
func (m *Xoscal) CandidateSubjectReports(source *dagger.Directory, releaseTag string, sourceRevision string, subject string, reportMode string) (*dagger.Directory, error) {
	kind, parameter, err := diagnosticSubject(subject)
	if err != nil {
		return nil, err
	}
	if !diagnosticMode(kind, reportMode) {
		return nil, fmt.Errorf("unsupported diagnostic mode %q for subject %q", reportMode, subject)
	}
	identity := m.ReleaseIdentity(source, releaseTag, sourceRevision)
	tracked := identity.Directory("/tmp/release-source")
	var evidence *dagger.Directory
	switch kind {
	case "source":
		evidence = m.SecurityReports(tracked)
	case "frameworks":
		frameworks := m.OscalFrameworks(tracked)
		evidence = m.OscalConstraintReports(tracked, frameworks).Directory("/tmp/constraint-evidence")
	case "frontend":
		produced := m.FrontendProductionReports()
		switch reportMode {
		case "vulnerabilities":
			evidence = m.FrontendAnalysisReports()
		case "sbom-raw":
			evidence = produced
		case "sbom-assessment":
			// Preserve the publisher tar, map and inventory binding alongside the
			// exact SBOM assessment; no synthetic whole-package graph is added.
			evidence = m.SbomValidationReports(tracked, produced.File("sbom.cyclonedx.json")).WithDirectory("production", produced)
		}
	default:
		var artifact *dagger.File
		switch kind {
		case "sdk":
			// Same per-language graph used by SdkBundles, without forcing every
			// other language to succeed before its reports can be exported.
			artifact = m.sdkPackage(tracked, m.sdkGenerated(tracked), parameter).File(parameter + ".zip")
		case "archive":
			name := fmt.Sprintf("ckodex-xoscal_%s_%s.tar.gz", strings.TrimPrefix(releaseTag, "v"), parameter)
			artifact = m.Release(tracked).File(name)
		case "image":
			artifact = m.imagePlatform(tracked, parameter).AsTarball()
		case "binary":
			artifact = m.Build(tracked)
		}
		switch reportMode {
		case "vulnerabilities":
			switch kind {
			case "image":
				evidence = m.ImageAnalysisReports(artifact)
			case "binary":
				evidence = m.binaryDiagnosticReports(artifact)
			default:
				evidence = m.ArtifactAnalysisReports(artifact)
			}
		case "sbom-raw":
			if kind == "image" {
				evidence = m.ImageSbomReports(artifact)
			} else {
				evidence = m.ArtifactSbomReports(artifact)
			}
		case "sbom-assessment":
			var produced *dagger.Directory
			if kind == "image" {
				produced = m.ImageSbomProducedReports(tracked, artifact)
			} else {
				produced = m.ArtifactSbomProducedReports(tracked, artifact)
			}
			// Retain ALL original metadata, raw inventory and publisher records,
			// including the exact payload hash, even when assessment.status != 0.
			evidence = m.SbomValidationReports(tracked, produced.File("sbom.cyclonedx.json")).
				WithDirectory("production", produced).
				WithFile("subject.sha256", produced.File("subject.sha256"))
		}
	}
	context, err := json.MarshalIndent(map[string]interface{}{
		"schema_version": 1, "role": "unadmitted failure diagnostics", "admitted": false,
		"subject": subject, "report_mode": reportMode, "release_tag": releaseTag, "source_revision": sourceRevision,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return evidence.WithFile("release-source.json", identity.File("/tmp/release-source.json")).
		WithNewFile("diagnostic-context.json", string(context)+"\n"), nil
}

func diagnosticSubject(subject string) (string, string, error) {
	switch subject {
	case "source", "frameworks", "frontend", "binary":
		return subject, "", nil
	case "sdk-go", "sdk-python", "sdk-java", "sdk-csharp", "sdk-ts", "sdk-swift":
		return "sdk", strings.TrimPrefix(subject, "sdk-"), nil
	case "image-amd64", "image-arm64":
		return "image", strings.TrimPrefix(subject, "image-"), nil
	case "archive-linux_amd64":
		return "archive", "Linux_x86_64", nil
	case "archive-linux_arm64":
		return "archive", "Linux_arm64", nil
	case "archive-darwin_amd64":
		return "archive", "Darwin_x86_64", nil
	case "archive-darwin_arm64":
		return "archive", "Darwin_arm64", nil
	default:
		return "", "", fmt.Errorf("unsupported diagnostic subject %q", subject)
	}
}

func diagnosticMode(kind, mode string) bool {
	if kind == "source" {
		return mode == "vulnerabilities"
	}
	if kind == "frameworks" {
		return mode == "constraints"
	}
	return mode == "vulnerabilities" || mode == "sbom-raw" || mode == "sbom-assessment"
}

// A bare binary is not an archive. Scan its exact bytes without passing it to
// the archive extractor, retaining the tool exit independently of admission.
func (m *Xoscal) binaryDiagnosticReports(binary *dagger.File) *dagger.Directory {
	return m.trivyBase().WithFile("/input/xoscal-server", binary).
		WithExec([]string{"sh", "-c", "mkdir -p /evidence\nsha256sum /input/xoscal-server > /evidence/artifact.sha256\nset +e\ntrivy fs /input --scanners vuln --format json --list-all-pkgs --exit-code 0 --output /evidence/trivy-results.json > /evidence/trivy.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/trivy.status"}).Directory("/evidence")
}
