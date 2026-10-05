package main

import (
	"dagger/xoscal/internal/dagger"
	"fmt"
	"strings"
)

// imagePlatform cross-compiles the same source for each declared OCI platform.
func (m *Xoscal) imagePlatform(source *dagger.Directory, arch string) *dagger.Container {
	bin := m.base(source).WithEnvVariable("GOOS", "linux").WithEnvVariable("GOARCH", arch).
		WithExec([]string{"go", "build", "-trimpath", fmt.Sprintf("-ldflags=-s -w -X main.version=%s", m.Version), "-o", "/tmp/xoscal-server", "./server/cmd/xoscal-server"}).File("/tmp/xoscal-server")
	return dag.Container(dagger.ContainerOpts{Platform: dagger.Platform("linux/" + arch)}).
		From(distrolessBase).WithWorkdir("/data").WithFile("/xoscal-server", bin).
		WithExposedPort(50051).WithExposedPort(9090).
		WithEntrypoint([]string{"/xoscal-server"}).WithDefaultArgs([]string{"-config", "/etc/xoscal/config.yaml"})
}

// ReleaseCandidate produces and analyzes every payload before any publication.
// Explicit identity is checked by the inventory and later hosted verification.
func (m *Xoscal) ReleaseCandidate(source *dagger.Directory, releaseTag string, sourceRevision string) *dagger.Directory {
	identity := m.ReleaseIdentity(source, releaseTag, sourceRevision)
	tracked := identity.Directory("/tmp/release-source")
	return m.releaseCandidate(tracked, releaseTag, sourceRevision, m.Release(tracked), identity.File("/tmp/release-source.json"))
}

// PreviewCandidate exercises the complete production/admission graph with
// snapshot archives, without creating a Git tag, hosted proof or publication.
func (m *Xoscal) PreviewCandidate(source *dagger.Directory, sourceRevision string) *dagger.Directory {
	return m.releaseCandidate(source, m.Version, sourceRevision, m.Snapshot(source), nil)
}

func (m *Xoscal) releaseCandidate(source *dagger.Directory, releaseTag, sourceRevision string, archives *dagger.Directory, identity *dagger.File) *dagger.Directory {
	frameworks := m.OscalFrameworks(source)
	sdks := m.SdkBundles(source)
	binary := m.Build(source)
	binaryEvidence := m.ArtifactSbomAnalysis(source, binary)
	amd := m.imagePlatform(source, "amd64")
	arm := m.imagePlatform(source, "arm64")
	amdArchive, armArchive := amd.AsTarball(), arm.AsTarball()
	image := amd.AsTarball(dagger.ContainerAsTarballOpts{PlatformVariants: []*dagger.Container{arm}})
	c := m.siteShell(source).WithEnvVariable("RELEASE_TAG", releaseTag).WithEnvVariable("SOURCE_REVISION", sourceRevision).
		WithDirectory("/out/sdk", sdks).WithDirectory("/out/frameworks", frameworks).
		WithFile("/out/evidence/policy/release-policy.json", source.File("docs/release-policy.json")).
		WithFile("/out/evidence/policy/analysis-exceptions.json", source.File("docs/analysis-exceptions.json")).
		WithFile("/out/evidence/policy/framework-manifest.yaml", source.File("data/frameworks/manifest.yaml")).
		WithDirectory("/out/evidence/ci", m.All(source)).WithDirectory("/out/evidence/security", m.SecurityAnalysis(source)).
		WithFile("/out/openapi.json", m.Openapi(source)).WithFile("/out/release/image.tar", image).
		WithFile("/out/release/xoscal-server", binary).
		WithFile("/out/release/image-amd64.tar", amdArchive).WithFile("/out/release/image-arm64.tar", armArchive).
		WithFile("/out/sbom-assessment-results.json", binaryEvidence.File("sbom-assessment-results.oscal.json")).
		WithDirectory("/out/evidence/binary-sbom", binaryEvidence).
		WithDirectory("/out/evidence/image-amd64", m.ImageAnalysis(source, amdArchive)).
		WithDirectory("/out/evidence/image-arm64", m.ImageAnalysis(source, armArchive)).
		WithDirectory("/out/evidence/image-amd64-sbom", m.ImageSbomAnalysis(source, amdArchive)).
		WithDirectory("/out/evidence/image-arm64-sbom", m.ImageSbomAnalysis(source, armArchive)).
		WithDirectory("/staged", archives).
		WithExec([]string{"sh", "-c", "set -eu; mkdir -p /out/release; cp /staged/*.tar.gz /staged/*.sbom.json /staged/checksums.txt /out/release/"}).
		WithExec([]string{"python3", "scripts/inspect-image-archive.py", "/out/release/image.tar", "/out/release"})
	for _, language := range []string{"go", "python", "java", "csharp", "ts", "swift"} {
		artifact := sdks.File(language + ".zip")
		stage := "/out/evidence/sdk-" + language
		c = c.WithDirectory(stage, m.ArtifactAnalysis(source, artifact)).
			WithDirectory(stage+"-sbom", m.ArtifactSbomAnalysis(source, artifact))
	}
	for _, target := range []struct{ os, fileOS, arch, fileArch string }{
		{"linux", "Linux", "amd64", "x86_64"}, {"linux", "Linux", "arm64", "arm64"},
		{"darwin", "Darwin", "amd64", "x86_64"}, {"darwin", "Darwin", "arm64", "arm64"},
	} {
		name := fmt.Sprintf("ckodex-xoscal_%s_%s_%s.tar.gz", strings.TrimPrefix(releaseTag, "v"), target.fileOS, target.fileArch)
		artifact := archives.File(name)
		stage := "/out/evidence/archive-" + target.os + "_" + target.arch
		c = c.WithDirectory(stage, m.ArtifactAnalysis(source, artifact)).
			WithDirectory(stage+"-sbom", m.ArtifactSbomAnalysis(source, artifact))
	}
	if identity != nil {
		c = c.WithFile("/out/evidence/release-source.json", identity)
	}
	return c.WithExec([]string{"sh", "-c", "python3 scripts/release-contract.py create --root /out --tag \"$RELEASE_TAG\" --revision \"$SOURCE_REVISION\" --required"}).
		WithExec([]string{"python3", "scripts/render-release-site.py", "--root", "/out", "--manifest", "/out/payload-manifest.json"}).Directory("/out")
}

// FinalizeCandidate verifies the payload signature before rendering eligibility,
// then creates the final non-self-referential manifest of all presentation bytes.
func (m *Xoscal) FinalizeCandidate(source *dagger.Directory, candidate *dagger.Directory, payloadBundle *dagger.File, releaseTag string, sourceRevision string) *dagger.Directory {
	c := m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").WithDirectory("/out", candidate).
		WithFile("/out/proofs/payload.sigstore.json", payloadBundle).
		WithEnvVariable("RELEASE_TAG", releaseTag).WithEnvVariable("SOURCE_REVISION", sourceRevision).
		WithExec([]string{"sh", "-c", "python3 scripts/release-contract.py verify --root /out --bundle /out/proofs/payload.sigstore.json --tag \"$RELEASE_TAG\" --revision \"$SOURCE_REVISION\" --required"}).
		WithExec([]string{"sh", "-c", "python3 scripts/render-release-site.py --root /out --manifest /out/payload-manifest.json --bundle /out/proofs/payload.sigstore.json --tag \"$RELEASE_TAG\" --revision \"$SOURCE_REVISION\""}).
		WithExec([]string{"python3", "scripts/site-smoke.py", "--root", "/out", "--generated"})
	c = c.WithFile("/out/evidence/presentation.ok", m.PresentationAnalysis(source, c.Directory("/out")).File("/tmp/presentation.ok"))
	return c.WithExec([]string{"sh", "-c", "python3 scripts/release-contract.py create --root /out --tag \"$RELEASE_TAG\" --revision \"$SOURCE_REVISION\" --final --required"}).Directory("/out")
}

// VerifyCandidate validates the final signature and every unchanged file before
// the hosted promotion boundary. The detached final proof does not hash itself.
func (m *Xoscal) VerifyCandidate(source *dagger.Directory, candidate *dagger.Directory, releaseBundle *dagger.File, releaseTag string, sourceRevision string) *dagger.Directory {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").WithDirectory("/out", candidate).
		WithFile("/out/proofs/release-manifest.sigstore.json", releaseBundle).
		WithExec([]string{"python3", "scripts/pages-capacity.py", "/out"}).
		WithEnvVariable("RELEASE_TAG", releaseTag).WithEnvVariable("SOURCE_REVISION", sourceRevision).
		WithExec([]string{"sh", "-c", "python3 scripts/release-contract.py verify --root /out --bundle /out/proofs/release-manifest.sigstore.json --tag \"$RELEASE_TAG\" --revision \"$SOURCE_REVISION\" --final --required"}).
		Directory("/out")
}

// PromotionAssets verifies the unchanged candidate again and creates only a
// flat attachment view. Every attachment name and digest comes from signed data.
func (m *Xoscal) PromotionAssets(source *dagger.Directory, candidate *dagger.Directory, releaseBundle *dagger.File, releaseTag string, sourceRevision string) *dagger.Directory {
	verified := m.VerifyCandidate(source, candidate, releaseBundle, releaseTag, sourceRevision)
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").WithDirectory("/candidate", verified).
		WithExec([]string{"python3", "scripts/promotion-assets.py", "/candidate", "/attachments"}).Directory("/attachments")
}
