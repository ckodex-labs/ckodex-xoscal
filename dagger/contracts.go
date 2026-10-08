package main

import "dagger/xoscal/internal/dagger"

// ReleaseContractCheck exercises missing/tampered evidence, exact subject
// binding, package receipts and installer refusal in the pipeline itself.
func (m *Xoscal) ReleaseContractCheck(source *dagger.Directory) *dagger.Container {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").
		WithExec([]string{"python3", "scripts/render-blueprint.py", "--check"}).
		WithExec([]string{"python3", "-m", "unittest", "discover", "-s", "scripts", "-p", "test_*.py"}).
		WithExec([]string{"python3", "scripts/test-install.py"}).
		WithExec([]string{"python3", "scripts/sdk-package-test.py"}).
		WithExec([]string{"sh", "-c", "printf 'release-contract-ok\\n' > /tmp/release-contract.ok"})
}

// PresentationAnalysis validates the actual generated HTML and served asset
// contract before final bytes enter their immutable signed manifest.
func (m *Xoscal) PresentationAnalysis(source *dagger.Directory, candidate *dagger.Directory) *dagger.Container {
	return m.verifyTools().WithDirectory("/src", source).WithWorkdir("/src").
		WithFile("/tools/render_api_reference.py", dag.CurrentModule().Source().File("render_api_reference.py")).
		WithDirectory("/src/site", candidate).
		WithExec([]string{"python3", "scripts/check-blueprint.py", "--root", "/src/site"}).
		WithExec([]string{"python3", "scripts/check-release-presentation.py", "--root", "/src/site"}).
		WithExec([]string{"python3", "/tools/render_api_reference.py", "--root", "/src/site", "--check"}).
		WithExec([]string{"python3", "scripts/check-frontend-assets.py", "--root", "/src/site", "--exceptions", "docs/analysis-exceptions.json"}).
		WithExec([]string{"python3", "scripts/pages-capacity.py", "/src/site"}).
		WithExec([]string{"python3", "scripts/design-lint.py"}).
		WithExec([]string{"python3", "scripts/a11y-lint.py"}).
		WithExec([]string{"python3", "scripts/site-smoke.py", "--root", "/src/site", "--generated"}).
		WithDirectory("/tmp/presentation-browser", m.PresentationBrowserAnalysis(source, candidate).Directory("/evidence")).
		WithExec([]string{"python3", "scripts/presentation_browser_verify.py", "--root", "/src/site", "--stage", "/tmp/presentation-browser"}).
		WithExec([]string{"sh", "-c", "printf 'presentation-analysis-ok\\n' > /tmp/presentation.ok"})
}
