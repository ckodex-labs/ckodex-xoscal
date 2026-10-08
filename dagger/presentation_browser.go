package main

import "dagger/xoscal/internal/dagger"

const presentationBrowserImage = "mcr.microsoft.com/playwright:v1.63.0-noble@sha256:eff16c30e6f3f4af0a03fa4b706120d5e9b0891c344a27d64559aff5900a4a27"

// presentationBrowserProducer runs only read-only requests against a shared
// candidate service. Browser tooling remains in this analysis container.
func (m *Xoscal) presentationBrowserProducer(source, candidate *dagger.Directory) *dagger.Container {
	service := m.verifyTools().WithDirectory("/site", candidate).WithWorkdir("/site").
		WithExposedPort(8080).AsService(dagger.ContainerAsServiceOpts{
		Args: []string{"python3", "-m", "http.server", "8080", "--bind", "0.0.0.0"},
	})
	return dag.Container().From(presentationBrowserImage).
		WithDirectory("/tools", dag.CurrentModule().Source().Directory("presentation-browser")).
		WithWorkdir("/tools").
		WithEnvVariable("PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD", "1").
		WithExec([]string{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund", "--registry=https://registry.npmjs.org", "--userconfig=/dev/null", "--globalconfig=/tmp/xoscal-empty-global-npmrc"}).
		WithFile("/tools/check-presentation-browser.mjs", source.File("scripts/check-presentation-browser.mjs")).
		WithFile("/tools/check-api-landmarks.mjs", source.File("scripts/check-api-landmarks.mjs")).
		WithFile("/tools/browser-toolchain.json", dag.CurrentModule().Source().File("presentation-browser/toolchain.json")).
		WithDirectory("/candidate", candidate).
		WithServiceBinding("presentation", service).
		WithExec([]string{"node", "/tools/check-presentation-browser.mjs", "--root", "/candidate", "--base", "http://presentation:8080/", "--output", "/evidence", "--image", presentationBrowserImage, "--report-only", "true"})
}

// PresentationBrowserReports exports raw failures as well as successful checks.
// Report production alone never admits a candidate.
func (m *Xoscal) PresentationBrowserReports(source, candidate *dagger.Directory) *dagger.Directory {
	return m.presentationBrowserProducer(source, candidate).Directory("/evidence")
}

// PresentationBrowserAnalysis rejects failed or stale exact-byte receipts.
// Call this mandatory gate before immutable final manifest creation.
func (m *Xoscal) PresentationBrowserAnalysis(source, candidate *dagger.Directory) *dagger.Container {
	return m.presentationBrowserProducer(source, candidate).
		WithExec([]string{"node", "/tools/check-presentation-browser.mjs", "--root", "/candidate", "--output", "/evidence", "--require-pass", "true"})
}
