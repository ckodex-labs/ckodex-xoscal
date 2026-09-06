package main

import (
	"dagger/xoscal/internal/dagger"
)

const buildDownloadsIndex = `
{
  echo '{"sdks":['
  first=1
  for z in /out/sdk/*.zip; do
    [ -e "$z" ] || continue
    d=$(cat "$z.sha256" 2>/dev/null || echo "")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"sdk/%s","digest":"%s"}' "$(basename "$z")" "$(basename "$z")" "$d"
  done
  echo '],"frameworks":['
  first=1
  for c in /out/frameworks/*/catalog.json; do
    [ -e "$c" ] || continue
    d=$(cat "$c.sha256" 2>/dev/null || echo "")
    name=$(basename "$(dirname "$c")")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"frameworks/%s/catalog.json","digest":"%s"}' "$name" "$name" "$d"
  done
  echo ']}'
} > /out/downloads.json
`

// releaseAssets fetches the latest GitHub release assets and emits
// release-assets.json (name, url, size, digest) + downloads the
// sbom-assessment-results.json from the release. This pins the site
// to the actual release artifacts instead of regenerating independently.
// Uses GITHUB_TOKEN if available for authenticated API access (5000 req/hr
// vs 60 unauthenticated). Wrapped in a subshell so failures don't abort
// the site build.
const releaseAssetsScript = `
(
  api="https://api.github.com/repos/ckodex-labs/ckodex-xoscal/releases/latest"
  echo "Fetching latest release" >&2
  if [ -n "$GH_TOKEN" ]; then
    curl -fsSL -H "Accept: application/vnd.github+json" -H "Authorization: Bearer $GH_TOKEN" "$api" -o /tmp/release.json 2>/tmp/curl_err || {
      echo "API fetch failed: $(cat /tmp/curl_err)" >&2
      exit 1
    }
  else
    curl -fsSL -H "Accept: application/vnd.github+json" "$api" -o /tmp/release.json 2>/tmp/curl_err || {
      echo "API fetch failed: $(cat /tmp/curl_err)" >&2
      exit 1
    }
  fi
  ls -la /tmp/release.json >&2
  # Use python3 to parse — jq 1.6 chokes on control chars in release body
  tag=$(python3 -c "import json; print(json.load(open('/tmp/release.json'))['tag_name'])" 2>/tmp/py_err) || {
    echo "JSON parse failed: $(head -c 300 /tmp/py_err)" >&2
    head -c 300 /tmp/release.json >&2
    exit 1
  }
  [ "$tag" != "null" ] && [ -n "$tag" ] || { echo "No release found" >&2; exit 1; }
  echo "Latest release: $tag" >&2

  # Emit release-assets.json using python3
  python3 -c "
import json
rel = json.load(open('/tmp/release.json'))
assets = [{'name': a['name'], 'url': a['browser_download_url'], 'size': a['size'], 'digest': a.get('digest', '')} for a in rel.get('assets', [])]
json.dump(assets, open('/out/release-assets.json', 'w'), indent=2)
" 2>/tmp/py_err || {
    echo "Failed to emit release-assets.json: $(head -c 200 /tmp/py_err)" >&2
    exit 1
  }
  test -s /out/release-assets.json && echo "release-assets.json written ($(wc -c < /out/release-assets.json) bytes)" >&2

  # Download sbom-assessment-results.json from the release if it exists
  ar_url=$(python3 -c "
import json
rel = json.load(open('/tmp/release.json'))
for a in rel.get('assets', []):
    if a['name'] == 'sbom-assessment-results.json':
        print(a['browser_download_url'])
        break
" 2>/dev/null)
  if [ -n "$ar_url" ]; then
    echo "Downloading sbom-assessment-results.json from release $tag" >&2
    curl -fsSL "$ar_url" -o /out/sbom-assessment-results.json
    test -s /out/sbom-assessment-results.json
  else
    echo "No sbom-assessment-results.json in release $tag — keeping regenerated version" >&2
  fi
) || echo 'release-assets.json: fetch failed, using fallback' >&2
`

// Site assembles the static GitHub Pages portal: docs (Scalar+OpenAPI),
// transparency (provenance.json), downloads (SDK zips + OSCAL catalogs),
// and SBOM validation (OSCAL assessment-results from the latest release).
//
// +optional
// githubToken — when provided, authenticates GitHub API calls (5000 req/hr
// vs 60 unauthenticated). Pass env:GITHUB_TOKEN in CI for reliable
// release-assets.json generation.
func (m *Xoscal) Site(source *dagger.Directory,
	// +optional
	githubToken *dagger.Secret,
) *dagger.Directory {
	sdks := m.SdkBundles(source)
	frameworks := m.OscalFrameworks(source)
	prov := m.provenanceManifest(source, sdks)
	sbomValidation := m.SbomValidation(source)

	// Vendor a PINNED Scalar standalone bundle into the site (no runtime CDN).
	scalarVer := "1.25.0" // pin explicitly; bump deliberately
	scalarURL := "https://cdn.jsdelivr.net/npm/@scalar/api-reference@" + scalarVer + "/dist/browser/standalone.js"

	asm := m.base(source).
		WithDirectory("/out", source.Directory("site")).
		WithDirectory("/out/sdk", sdks).
		WithDirectory("/out/frameworks", frameworks).
		WithFile("/out/openapi.json", sdks.File("openapi.json")).
		WithFile("/out/provenance.json", prov.File("provenance.json")).
		WithFile("/out/sbom-assessment-results.json", sbomValidation).
		WithExec([]string{"sh", "-c",
			"curl -fsSL '" + scalarURL + "' -o /out/scalar.js && test -s /out/scalar.js"}).
		WithExec([]string{"sh", "-c", buildDownloadsIndex})

	// Wire GitHub token if provided (authenticated API access).
	if githubToken != nil {
		asm = asm.WithSecretVariable("GH_TOKEN", githubToken)
	}

	return asm.
		// Fetch release-assets.json from the latest GitHub release.
		// If the API call fails (rate limit, no release), the site still
		// builds — the downloads page falls back to the hardcoded data.
		WithExec([]string{"sh", "-c",
			"apt-get update && apt-get install -y --no-install-recommends jq >/dev/null 2>&1; " +
				"which jq && jq --version >&2; " +
				releaseAssetsScript}).
		// Cache-busting: append ?v=<short-sha> to CSS/JS asset URLs in all
		// HTML files. Prevents CDN staleness after deploys.
		WithExec([]string{"sh", "-c",
			"sha=$(git -C /src rev-parse --short HEAD 2>/dev/null || echo dev); " +
				"for f in /out/*.html; do " +
				"sed -i \"s/\\(styles\\.css\\)\\([\\\"']\\)/\\1?v=$sha\\2/g; " +
				"s/\\(ds3\\.css\\)\\([\\\"']\\)/\\1?v=$sha\\2/g; " +
				"s/\\(scalar\\.js\\)\\([\\\"']\\)/\\1?v=$sha\\2/g\" \"$f\"; " +
				"done"}).
		Directory("/out")
}

// Serve builds the portal site and serves it as a static HTTP service on :8080.
// One command to preview the real site locally (built on the engine, so buf
// generate + framework fetch have network):
//
//	dagger call serve --source=. up --ports 8080:8080
//
// then open http://localhost:8080.
func (m *Xoscal) Serve(source *dagger.Directory) *dagger.Service {
	return dag.Container().
		From("python:3.13-alpine").
		WithDirectory("/site", m.Site(source, nil)).
		WithWorkdir("/site").
		WithExposedPort(8080).
		AsService(dagger.ContainerAsServiceOpts{
			Args: []string{"python", "-m", "http.server", "8080"},
		})
}

// Security runs govulncheck and gosec, returning the SARIF report file.
// Reuses the cached toolBase layer so tools are not reinstalled on every run.
