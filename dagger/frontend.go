package main

import "dagger/xoscal/internal/dagger"

// FrontendProductionReports builds published ES modules with a reviewed lock
// and exact security overrides. Original npm tar/SRI/JS/map remain replayable;
// the new assembly is explicitly a repository build, not publisher CDN bytes.
// Reviewed landmark adapters run on hash-pinned upstream modules before
// bundling; raw inputs, transformed-source receipt and final maps are replayed.
func (m *Xoscal) FrontendProductionReports() *dagger.Directory {
	publisher := m.trivyBase().
		WithFile("/tools/frontend_inventory.py", dag.CurrentModule().Source().File("frontend_inventory.py")).
		WithExec([]string{"sh", "-c", "set -eu; mkdir -p /evidence; curl -fsSL 'https://registry.npmjs.org/%40scalar%2Fapi-reference/1.72.4' -o /evidence/npm-metadata.json; curl -fsSL 'https://registry.npmjs.org/@scalar/api-reference/-/api-reference-1.72.4.tgz' -o /evidence/npm-package.tgz"}).
		WithExec([]string{"python3", "/tools/frontend_inventory.py", "--root", "/evidence"}).Directory("/evidence").WithoutFile("scalar.js").WithoutFile("standalone.js.map")
	builder := sdkToolchain("ts").
		WithDirectory("/build", dag.CurrentModule().Source().Directory("frontend")).
		WithWorkdir("/build").WithEnvVariable("NODE_OPTIONS", "--max-old-space-size=2048").
		WithExec([]string{"sh", "-c", "set -eu; mkdir -p out; npm ci --ignore-scripts --no-audit --no-fund --registry=https://registry.npmjs.org --userconfig=/dev/null --globalconfig=/tmp/xoscal-empty-global-npmrc > out/npm-ci.log 2>&1; printf '0\\n' > out/npm-ci.status; node --test landmark-transform.test.mjs > out/build.log 2>&1; node build.mjs >> out/build.log 2>&1; printf '0\\n' > out/build.status"})
	evidence := publisher.WithDirectory("/", builder.Directory("/build/out")).
		WithFile("build-package.json", dag.CurrentModule().Source().File("frontend/package.json")).
		WithFile("package-lock.json", dag.CurrentModule().Source().File("frontend/package-lock.json")).
		WithFile("entry.js", dag.CurrentModule().Source().File("frontend/entry.js")).
		WithFile("build.mjs", dag.CurrentModule().Source().File("frontend/build.mjs")).
		WithFile("landmark-transform.mjs", dag.CurrentModule().Source().File("frontend/landmark-transform.mjs")).
		WithFile("landmark-transform.test.mjs", dag.CurrentModule().Source().File("frontend/landmark-transform.test.mjs")).
		WithFile("landmark-transforms.json", dag.CurrentModule().Source().File("frontend/landmark-transforms.json")).
		WithFile("scalar-LICENSE", dag.CurrentModule().Source().File("frontend/scalar-LICENSE")).
		WithFile("scalar-license-source.json", dag.CurrentModule().Source().File("frontend/scalar-license-source.json"))
	return m.trivyBase().WithDirectory("/evidence", evidence).
		WithFile("/tools/frontend_inventory.py", dag.CurrentModule().Source().File("frontend_inventory.py")).
		WithExec([]string{"python3", "/tools/frontend_inventory.py", "--root", "/evidence", "--build"}).Directory("/evidence")
}

// FrontendAnalysisReports preserves all severities from the actual observed
// browser assembly SBOM, including CSS/asset contributors and nested versions.
func (m *Xoscal) FrontendAnalysisReports() *dagger.Directory {
	c := m.trivyBase().WithDirectory("/evidence", m.FrontendProductionReports())
	return m.analysisAttempt(c, "frontend-vulnerabilities").
		WithExec([]string{"sh", "-c", "set +e\ntrivy sbom /evidence/sbom.cyclonedx.json --scanners vuln --format json --list-all-pkgs --exit-code 0 --output /evidence/trivy-results.json > /evidence/trivy.log 2>&1\nprintf '%s\\n' \"$?\" > /evidence/trivy.status"}).Directory("/evidence")
}

// FrontendAssets admits exact JS/map bytes after producer replay and the
// existing release CRITICAL/fixed gate. Lower findings remain in raw evidence.
func (m *Xoscal) FrontendAssets(source *dagger.Directory) *dagger.Directory {
	return m.analysisGate(source, m.FrontendAnalysisReports(), "frontend-release").Directory("/evidence")
}
