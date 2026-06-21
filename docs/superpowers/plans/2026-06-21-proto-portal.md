# Proto Portal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a static GitHub Pages portal that documents the OSCAL protos, shows build-time supply-chain provenance for the protos/SDKs, and offers downloadable SDK clients and OSCAL-converted frameworks.

**Architecture:** All artifacts are assembled by new Dagger functions (logic stays in Dagger, not GHA) from things the repo already generates (buf SDKs/OpenAPI, ingest→OSCAL, CI provenance). Two thin Go CLIs fill real gaps: provenance-manifest assembly and an end-to-end framework→OSCAL exporter. A `pages.yml` workflow only calls Dagger and deploys.

**Tech Stack:** Go 1.25, Dagger (Go SDK), buf v2, Scalar UI (static), CKODEX-DS-3 CSS, GitHub Pages.

## Global Constraints

- Go: 1.25; `gofmt` clean; `go vet ./...` clean; errors wrapped `fmt.Errorf("op: %w", err)`.
- No business logic in GitHub Actions — assembly lives in Dagger functions (`dagger/main.go`).
- Provenance honesty: a field is emitted only when its evidence exists; absent signing ⇒ `"signed": false`, never a fabricated attestation (P-VW-001 / Rule 12).
- Every downloadable file has a sibling `<file>.sha256`; index digests must equal file digests.
- Dagger calls run against the local engine: prefix with `_EXPERIMENTAL_DAGGER_RUNNER_HOST=tcp://127.0.0.1:1234`.
- DS-3: square `.ck-quiet` containers; digest law `sha256:9f3c…a217`; only attested artifacts earn `proof.violet`/`.ck-sealed`.
- Branch: `chore/oscal-reconcile-loop`. Commit after every task.

## File Structure

- `server/internal/portal/provenance.go` — pure provenance-manifest assembly + honesty rule.
- `server/internal/portal/manifest.go` — parse `data/frameworks/manifest.yaml` + derive GitHub owner/repo/path from `source_base_url`.
- `server/cmd/xoscal-provenance/main.go` — thin CLI: inputs → `provenance.json`.
- `server/cmd/xoscal-export-frameworks/main.go` — fetch→ingest→snapshot→generate OSCAL catalogs per framework.
- `site/{index,docs,transparency,downloads}.html`, `site/ds3.css` — static DS-3 shell.
- `dagger/main.go` — `+ sdkBundles, oscalFrameworks, provenanceManifest, Site`.
- `.github/workflows/pages.yml` — build via Dagger + deploy Pages.

---

### Task 1: Provenance manifest package (honesty rule)

**Files:**
- Create: `server/internal/portal/provenance.go`
- Test: `server/internal/portal/provenance_test.go`

**Interfaces:**
- Produces: `type Artifact struct { Name, Digest, SBOMRef, SLSARef, CosignBundle, RekorID string }`; `func BuildManifest(arts []Artifact) ([]byte, error)`. Output JSON: `[{ "name","digest","sbom_ref"?,"slsa_ref"?,"cosign_bundle"?,"rekor_id"?,"signed":bool }]`. `signed` is true iff `CosignBundle != "" && RekorID != ""`. Empty optional refs are omitted from JSON.

- [ ] **Step 1: Write the failing test**

```go
package portal

import (
	"encoding/json"
	"testing"
)

func TestBuildManifest_UnsignedHasNoProof(t *testing.T) {
	out, err := BuildManifest([]Artifact{{Name: "go-sdk", Digest: "sha256:abc"}})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[0]["signed"] != false {
		t.Errorf("signed = %v, want false", got[0]["signed"])
	}
	if _, ok := got[0]["cosign_bundle"]; ok {
		t.Errorf("cosign_bundle present on unsigned artifact")
	}
}

func TestBuildManifest_SignedWhenBundleAndRekor(t *testing.T) {
	out, _ := BuildManifest([]Artifact{{
		Name: "proto-set", Digest: "sha256:def",
		CosignBundle: "cosign.bundle", RekorID: "1234567",
	}})
	var got []map[string]any
	_ = json.Unmarshal(out, &got)
	if got[0]["signed"] != true {
		t.Errorf("signed = %v, want true", got[0]["signed"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./server/internal/portal/ -run TestBuildManifest -v`
Expected: FAIL — `undefined: BuildManifest` / `undefined: Artifact`.

- [ ] **Step 3: Write minimal implementation**

```go
// Package portal assembles static-portal artifacts (provenance, framework export).
package portal

import "encoding/json"

// Artifact is one provenance subject (the proto set or an SDK bundle).
type Artifact struct {
	Name         string
	Digest       string
	SBOMRef      string
	SLSARef      string
	CosignBundle string
	RekorID      string
}

type record struct {
	Name         string `json:"name"`
	Digest       string `json:"digest"`
	SBOMRef      string `json:"sbom_ref,omitempty"`
	SLSARef      string `json:"slsa_ref,omitempty"`
	CosignBundle string `json:"cosign_bundle,omitempty"`
	RekorID      string `json:"rekor_id,omitempty"`
	Signed       bool   `json:"signed"`
}

// BuildManifest renders provenance records. An artifact is "signed" only when it
// carries both a cosign bundle and a Rekor entry; otherwise signed=false and no
// proof fields are emitted (no fabricated attestation).
func BuildManifest(arts []Artifact) ([]byte, error) {
	out := make([]record, 0, len(arts))
	for _, a := range arts {
		out = append(out, record{
			Name: a.Name, Digest: a.Digest,
			SBOMRef: a.SBOMRef, SLSARef: a.SLSARef,
			CosignBundle: a.CosignBundle, RekorID: a.RekorID,
			Signed: a.CosignBundle != "" && a.RekorID != "",
		})
	}
	return json.MarshalIndent(out, "", "  ")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./server/internal/portal/ -run TestBuildManifest -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add server/internal/portal/provenance.go server/internal/portal/provenance_test.go
git commit -m "feat(portal): provenance manifest with honest signed flag"
```

---

### Task 2: Manifest parser + source URL derivation

**Files:**
- Create: `server/internal/portal/manifest.go`
- Test: `server/internal/portal/manifest_test.go`

**Interfaces:**
- Produces: `type Framework struct { RefID, Jurisdiction, Filename string }`; `type Manifest struct { Version, SourceBaseURL string; Frameworks []Framework }`; `func ParseManifest(b []byte) (*Manifest, error)`; `func (m *Manifest) Source() (owner, repo, path string, err error)` parsing a `raw.githubusercontent.com/{owner}/{repo}/{branch}/{path...}` URL; `func (m *Manifest) RefIDs() []string`.

- [ ] **Step 1: Write the failing test**

```go
package portal

import "testing"

const sampleManifest = `version: "1.0"
source_base_url: "https://raw.githubusercontent.com/intuitem/ciso-assistant-community/main/backend/library/libraries"
frameworks:
  - ref_id: nist-csf-2.0
    jurisdiction: usa
    filename: nist-csf-2.0.yaml
  - ref_id: dora
    jurisdiction: eu
    filename: dora.yaml
`

func TestParseManifest_Source(t *testing.T) {
	m, err := ParseManifest([]byte(sampleManifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	owner, repo, path, err := m.Source()
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	if owner != "intuitem" || repo != "ciso-assistant-community" || path != "backend/library/libraries" {
		t.Errorf("got %s/%s/%s", owner, repo, path)
	}
	if ids := m.RefIDs(); len(ids) != 2 || ids[0] != "nist-csf-2.0" {
		t.Errorf("RefIDs = %v", ids)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./server/internal/portal/ -run TestParseManifest -v`
Expected: FAIL — `undefined: ParseManifest`.

- [ ] **Step 3: Write minimal implementation**

```go
package portal

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// Framework is one entry in data/frameworks/manifest.yaml.
type Framework struct {
	RefID        string `yaml:"ref_id"`
	Jurisdiction string `yaml:"jurisdiction"`
	Filename     string `yaml:"filename"`
}

// Manifest is the parsed framework manifest.
type Manifest struct {
	Version       string      `yaml:"version"`
	SourceBaseURL string      `yaml:"source_base_url"`
	Frameworks    []Framework `yaml:"frameworks"`
}

// ParseManifest parses the framework manifest YAML.
func ParseManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// Source derives the GitHub owner, repo, and directory path from a
// raw.githubusercontent.com source_base_url of the form
// https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path...}.
func (m *Manifest) Source() (owner, repo, path string, err error) {
	u, err := url.Parse(m.SourceBaseURL)
	if err != nil {
		return "", "", "", fmt.Errorf("parse source_base_url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("source_base_url path too short: %q", u.Path)
	}
	return parts[0], parts[1], strings.Join(parts[3:], "/"), nil
}

// RefIDs returns the ref_id of each framework in manifest order.
func (m *Manifest) RefIDs() []string {
	ids := make([]string, 0, len(m.Frameworks))
	for _, f := range m.Frameworks {
		ids = append(ids, f.RefID)
	}
	return ids
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./server/internal/portal/ -v`
Expected: PASS (all portal tests).

- [ ] **Step 5: Commit**

```bash
git add server/internal/portal/manifest.go server/internal/portal/manifest_test.go
git commit -m "feat(portal): framework manifest parser + source URL derivation"
```

---

### Task 3: `xoscal-export-frameworks` CLI

**Files:**
- Create: `server/cmd/xoscal-export-frameworks/main.go`

**Interfaces:**
- Consumes: `portal.ParseManifest`, `portal.Manifest.Source/RefIDs` (Task 2); `fetcher.NewGitHubFetcher(owner,repo,path)`, `kg.NewSQLiteStore(dsn, dbutil.PoolConfig{})`, `reconciler.NewReconciler(store)`, `ingestion.NewBulkIngestor(fetcher, rec, store)`, `(*BulkIngestor).Run(ctx, filter)`, `store.CreateSnapshot(ctx, name)`, `oscal.NewGenerator(store)`, `gen.GenerateAllArtifacts(ctx, snapshot, framework, nil)`, `oscal.ExportCatalogJSON(*catalogv1.Catalog)`.
- Produces: writes `<out>/<ref_id>/catalog.json` + `<out>/<ref_id>/catalog.json.sha256` per framework; flags `-manifest`, `-out`, `-dsn`.

- [ ] **Step 1: Write the implementation**

```go
// Command xoscal-export-frameworks fetches the frameworks listed in the manifest,
// ingests them into a temporary KG, snapshots it, and emits one OSCAL catalog per
// framework with a sha256 sidecar.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/portal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func main() {
	manifestPath := flag.String("manifest", "data/frameworks/manifest.yaml", "Framework manifest path")
	outDir := flag.String("out", "./oscal-frameworks", "Output directory")
	dsn := flag.String("dsn", "portal-export.db", "Temp SQLite DSN")
	flag.Parse()

	if err := run(*manifestPath, *outDir, *dsn); err != nil {
		log.Fatal(err)
	}
}

func run(manifestPath, outDir, dsn string) error {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	m, err := portal.ParseManifest(raw)
	if err != nil {
		return err
	}
	owner, repo, path, err := m.Source()
	if err != nil {
		return err
	}

	store, err := kg.NewSQLiteStore(dsn, dbutil.PoolConfig{})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()
	gh := fetcher.NewGitHubFetcher(owner, repo, path)
	rec := reconciler.NewReconciler(store)
	bulk := ingestion.NewBulkIngestor(gh, rec, store)

	if _, err := bulk.Run(ctx, strings.Join(m.RefIDs(), ",")); err != nil {
		return fmt.Errorf("bulk ingest: %w", err)
	}
	if _, err := store.CreateSnapshot(ctx, "portal"); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	gen := oscal.NewGenerator(store)
	for _, refID := range m.RefIDs() {
		res, err := gen.GenerateAllArtifacts(ctx, "portal", refID, nil)
		if err != nil {
			return fmt.Errorf("generate %s: %w", refID, err)
		}
		if res.Catalog == nil {
			log.Printf("warn: %s produced no catalog, skipping", refID)
			continue
		}
		data, err := oscal.ExportCatalogJSON(res.Catalog)
		if err != nil {
			return fmt.Errorf("export %s: %w", refID, err)
		}
		dir := filepath.Join(outDir, refID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		catPath := filepath.Join(dir, "catalog.json")
		if err := os.WriteFile(catPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", catPath, err)
		}
		sum := sha256.Sum256(data)
		if err := os.WriteFile(catPath+".sha256", []byte("sha256:"+hex.EncodeToString(sum[:])+"\n"), 0o644); err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify it builds and vets**

Run: `GOCACHE=$TMPDIR/gc go vet ./server/cmd/xoscal-export-frameworks/`
Expected: exit 0 (no undefined symbols — confirms every consumed signature is real).

- [ ] **Step 3: Commit**

```bash
git add server/cmd/xoscal-export-frameworks/main.go
git commit -m "feat(portal): xoscal-export-frameworks CLI (manifest -> OSCAL catalogs)"
```

---

### Task 4: `xoscal-provenance` CLI

**Files:**
- Create: `server/cmd/xoscal-provenance/main.go`

**Interfaces:**
- Consumes: `portal.Artifact`, `portal.BuildManifest` (Task 1).
- Produces: reads a JSON array of `Artifact` from `-in` (or stdin), writes `provenance.json` to `-out`. CLI contract: `xoscal-provenance -in artifacts.json -out provenance.json`.

- [ ] **Step 1: Write the implementation**

```go
// Command xoscal-provenance renders a provenance.json from an artifacts JSON file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mchorfa/xoscal/server/internal/portal"
)

func main() {
	in := flag.String("in", "", "Input artifacts JSON (array of {name,digest,...})")
	out := flag.String("out", "provenance.json", "Output path")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		log.Fatal(err)
	}
}

func run(in, out string) error {
	raw, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("read artifacts: %w", err)
	}
	var arts []portal.Artifact
	if err := json.Unmarshal(raw, &arts); err != nil {
		return fmt.Errorf("parse artifacts: %w", err)
	}
	data, err := portal.BuildManifest(arts)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
```

Note: `portal.Artifact` fields are exported strings, so JSON keys are `Name`, `Digest`, `SBOMRef`, `SLSARef`, `CosignBundle`, `RekorID` (Go default). The Dagger step in Task 7 emits this exact shape.

- [ ] **Step 2: Verify it builds and vets**

Run: `GOCACHE=$TMPDIR/gc go vet ./server/cmd/xoscal-provenance/`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add server/cmd/xoscal-provenance/main.go
git commit -m "feat(portal): xoscal-provenance CLI"
```

---

### Task 5: Static site shell (DS-3)

**Files:**
- Create: `site/index.html`, `site/docs.html`, `site/transparency.html`, `site/downloads.html`, `site/ds3.css`

**Interfaces:**
- Produces: static pages. `docs.html` loads Scalar from CDN against `./openapi.json`. `transparency.html` fetches `./provenance.json` and renders cards. `downloads.html` fetches `./downloads.json` (built in Task 8) and lists SDK zips + OSCAL catalogs with digests.

- [ ] **Step 1: Create `site/ds3.css`** (DS-3 subset)

```css
:root { --ck-bg-0:#F6F1E8; --ck-fg-1:#211B14; --ck-fg-3:#6E6457; --ck-accent:#B4532A; --ck-proof:#6D28D9; }
* { box-sizing: border-box; }
body { margin:0; background:var(--ck-bg-0); color:var(--ck-fg-1);
  font-family:"Geist",system-ui,sans-serif; line-height:1.5; }
header,main { padding:24px 32px; max-width:1100px; }
h1 { font-family:"Instrument Serif",Georgia,serif; font-weight:400; }
nav a { color:var(--ck-accent); margin-right:16px; text-decoration:none; }
.ck-quiet { border:1px solid var(--ck-fg-3); padding:16px; margin:12px 0; background:#fff; }
.ck-sealed { border-width:2px; clip-path:polygon(0 0,calc(100% - 10px) 0,100% 10px,100% 100%,0 100%); }
.ck-hash { font-family:"JetBrains Mono",monospace; font-variant-ligatures:none; font-size:13px; }
.ck-proof { color:var(--ck-proof); }
.unsigned { color:var(--ck-fg-3); }
```

- [ ] **Step 2: Create `site/index.html`**

```html
<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>xOSCAL — Proto Portal</title><link rel="stylesheet" href="ds3.css"></head>
<body><header><h1>xOSCAL Proto Portal</h1>
<nav><a href="docs.html">API Docs</a><a href="transparency.html">Transparency</a>
<a href="downloads.html">Downloads</a></nav></header>
<main><div class="ck-quiet">OSCAL gRPC contracts: documentation, supply-chain
provenance, SDK clients, and frameworks converted to OSCAL.</div></main></body></html>
```

- [ ] **Step 3: Create `site/docs.html`** (Scalar over OpenAPI — vendored, no runtime CDN)

The Scalar bundle is vendored into the site at build time (Task 8), not loaded
from a CDN at runtime. This avoids CDN-compromise / SRI risk and keeps the site
offline-safe (governance: egress deny-by-default, supply-chain integrity).

```html
<!doctype html><html lang="en"><head><meta charset="utf-8">
<title>xOSCAL — API Docs</title></head><body>
<script id="api-reference" data-url="./openapi.json"></script>
<script src="./scalar.js"></script>
</body></html>
```

- [ ] **Step 4: Create `site/transparency.html`** (renders provenance.json honestly)

```html
<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>xOSCAL — Transparency</title><link rel="stylesheet" href="ds3.css"></head>
<body><header><h1>Supply-chain Transparency</h1>
<nav><a href="index.html">Home</a></nav></header><main id="cards"></main>
<script>
fetch("./provenance.json").then(r=>r.json()).then(rows=>{
  document.getElementById("cards").innerHTML = rows.map(a=>{
    const cls = a.signed ? "ck-quiet ck-sealed" : "ck-quiet";
    const mark = a.signed
      ? `<span class="ck-proof">◆ attested · cosign+rekor #${a.rekor_id}</span>`
      : `<span class="unsigned">○ unsigned</span>`;
    return `<div class="${cls}"><strong>${a.name}</strong> ${mark}
      <div class="ck-hash">${a.digest}</div></div>`;
  }).join("");
});
</script></body></html>
```

- [ ] **Step 5: Create `site/downloads.html`** (SDK + OSCAL list)

```html
<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>xOSCAL — Downloads</title><link rel="stylesheet" href="ds3.css"></head>
<body><header><h1>Downloads</h1><nav><a href="index.html">Home</a></nav></header>
<main><h2>SDK Clients</h2><div id="sdks"></div>
<h2>OSCAL Frameworks</h2><div id="frameworks"></div></main>
<script>
fetch("./downloads.json").then(r=>r.json()).then(d=>{
  const row = x => `<div class="ck-quiet"><a href="${x.path}">${x.name}</a>
    <div class="ck-hash">${x.digest}</div></div>`;
  document.getElementById("sdks").innerHTML = d.sdks.map(row).join("");
  document.getElementById("frameworks").innerHTML = d.frameworks.map(row).join("");
});
</script></body></html>
```

- [ ] **Step 6: Verify structure**

Run: `for f in index docs transparency downloads; do test -f site/$f.html || echo "MISSING $f"; done; test -f site/ds3.css && echo OK`
Expected: `OK` (no MISSING lines).

- [ ] **Step 7: Commit**

```bash
git add site/
git commit -m "feat(portal): static DS-3 site shell (docs, transparency, downloads)"
```

---

### Task 6: Dagger `sdkBundles`

**Files:**
- Modify: `dagger/main.go`

**Interfaces:**
- Produces: `func (m *Xoscal) sdkBundles(source *dagger.Directory) *dagger.Directory` — returns a dir of `<lang>.zip` + `<lang>.zip.sha256` for go,python,java,csharp,ts,swift, plus `openapi.json` (first OpenAPI file from `gen/openapi`).

- [ ] **Step 1: Add the function** (after `ProtoCheck`)

```go
// sdkBundles runs buf generate and zips each language SDK with a sha256 sidecar.
// Also extracts the OpenAPI document for the docs page.
func (m *Xoscal) sdkBundles(source *dagger.Directory) *dagger.Directory {
	langs := "go python java csharp ts swift"
	gen := m.toolBase().
		WithExec([]string{"sh", "-c", "apt-get install -y --no-install-recommends zip"}).
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"buf", "generate"}).
		WithExec([]string{"mkdir", "-p", "/out"}).
		WithExec([]string{"sh", "-c",
			"for l in " + langs + "; do " +
				"d=proto/oscal/gen/$l; [ -d \"$d\" ] || { echo \"missing $d\" >&2; exit 1; }; " +
				"(cd \"$d\" && zip -qr /out/$l.zip .); " +
				"sha256sum /out/$l.zip | awk '{print \"sha256:\"$1}' > /out/$l.zip.sha256; " +
				"done"}).
		WithExec([]string{"sh", "-c",
			"cp \"$(find proto/oscal/gen/openapi -name '*.json' -o -name '*.yaml' | head -n1)\" /out/openapi.json"})
	return gen.Directory("/out")
}
```

- [ ] **Step 2: Verify it vets**

Run: `cd dagger && GOCACHE=$TMPDIR/gc go vet ./...`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add dagger/main.go
git commit -m "feat(portal): dagger sdkBundles (zip SDKs + sha256 + openapi)"
```

---

### Task 7: Dagger `oscalFrameworks` + `provenanceManifest`

**Files:**
- Modify: `dagger/main.go`

**Interfaces:**
- Produces: `func (m *Xoscal) oscalFrameworks(source *dagger.Directory) *dagger.Directory` (runs the Task 3 CLI, returns `/out` with `<ref_id>/catalog.json` + sidecars); `func (m *Xoscal) provenanceManifest(source *dagger.Directory, sdks *dagger.Directory) *dagger.Directory` (returns `/out/provenance.json`).

- [ ] **Step 1: Add `oscalFrameworks`**

```go
// oscalFrameworks builds and runs xoscal-export-frameworks over the manifest,
// emitting one OSCAL catalog per framework. Needs network (fetches upstream).
func (m *Xoscal) oscalFrameworks(source *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/export-fw", "./server/cmd/xoscal-export-frameworks"}).
		WithExec([]string{"/bin/export-fw", "-manifest", "data/frameworks/manifest.yaml", "-out", "/out", "-dsn", "/tmp/fw.db"}).
		Directory("/out")
}
```

- [ ] **Step 2: Add `provenanceManifest`** (honest: digests real artifacts; no signing data available at this stage ⇒ signed:false)

```go
// provenanceManifest digests the proto set and each SDK zip and renders
// provenance.json. Signing fields are left empty here (no cosign/rekor in this
// build stage), so every record is honestly signed:false until Release wires them.
func (m *Xoscal) provenanceManifest(source *dagger.Directory, sdks *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithDirectory("/sdks", sdks).
		WithExec([]string{"go", "build", "-o", "/bin/prov", "./server/cmd/xoscal-provenance"}).
		WithExec([]string{"sh", "-c",
			"proto_digest=$(find proto/oscal -name '*.proto' | sort | xargs sha256sum | sha256sum | awk '{print \"sha256:\"$1}'); " +
				"printf '[{\"Name\":\"proto-set\",\"Digest\":\"%s\"}' \"$proto_digest\" > /tmp/arts.json; " +
				"for z in /sdks/*.zip; do " +
				"d=$(sha256sum \"$z\" | awk '{print \"sha256:\"$1}'); " +
				"printf ',{\"Name\":\"%s\",\"Digest\":\"%s\"}' \"$(basename "$z" .zip)-sdk\" \"$d\" >> /tmp/arts.json; " +
				"done; printf ']' >> /tmp/arts.json"}).
		WithExec([]string{"mkdir", "-p", "/out"}).
		WithExec([]string{"/bin/prov", "-in", "/tmp/arts.json", "-out", "/out/provenance.json"}).
		Directory("/out")
}
```

- [ ] **Step 3: Verify it vets**

Run: `cd dagger && GOCACHE=$TMPDIR/gc go vet ./...`
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add dagger/main.go
git commit -m "feat(portal): dagger oscalFrameworks + provenanceManifest"
```

---

### Task 8: Dagger `Site` orchestrator + download index

**Files:**
- Modify: `dagger/main.go`

**Interfaces:**
- Consumes: `sdkBundles`, `oscalFrameworks`, `provenanceManifest` (Tasks 6–7), site shell (Task 5).
- Produces: `func (m *Xoscal) Site(source *dagger.Directory) *dagger.Directory` — the full `_site` directory. Builds `downloads.json` `{ "sdks":[{name,path,digest}], "frameworks":[{name,path,digest}] }`.

- [ ] **Step 1: Add `Site`**

```go
// Site assembles the static GitHub Pages portal: docs (Scalar+OpenAPI),
// transparency (provenance.json), and downloads (SDK zips + OSCAL catalogs).
func (m *Xoscal) Site(source *dagger.Directory) *dagger.Directory {
	sdks := m.sdkBundles(source)
	frameworks := m.oscalFrameworks(source)
	prov := m.provenanceManifest(source, sdks)

	// Vendor a PINNED Scalar standalone bundle into the site (no runtime CDN).
	scalarVer := "1.25.0" // pin explicitly; bump deliberately
	scalarURL := "https://cdn.jsdelivr.net/npm/@scalar/api-reference@" + scalarVer + "/dist/browser/standalone.js"

	asm := m.base(source).
		WithDirectory("/out", source.Directory("site")).
		WithDirectory("/out/sdk", sdks).
		WithDirectory("/out/frameworks", frameworks).
		WithFile("/out/openapi.json", sdks.File("openapi.json")).
		WithFile("/out/provenance.json", prov.File("provenance.json")).
		WithExec([]string{"sh", "-c",
			"curl -fsSL '" + scalarURL + "' -o /out/scalar.js && test -s /out/scalar.js"}).
		WithExec([]string{"sh", "-c", buildDownloadsIndex}).
		Directory("/out")
	return asm
}

// buildDownloadsIndex emits /out/downloads.json from the SDK zips and OSCAL catalogs.
const buildDownloadsIndex = `
{
  echo '{"sdks":['
  first=1
  for z in /out/sdk/*.zip; do
    d=$(cat "$z.sha256" 2>/dev/null || echo "")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"sdk/%s","digest":"%s"}' "$(basename "$z")" "$(basename "$z")" "$d"
  done
  echo '],"frameworks":['
  first=1
  for c in /out/frameworks/*/catalog.json; do
    d=$(cat "$c.sha256" 2>/dev/null || echo "")
    name=$(basename "$(dirname "$c")")
    [ $first -eq 1 ] || echo ','
    first=0
    printf '{"name":"%s","path":"frameworks/%s/catalog.json","digest":"%s"}' "$name" "$name" "$d"
  done
  echo ']}'
} > /out/downloads.json
`
```

- [ ] **Step 2: Verify it vets + gofmt**

Run: `cd dagger && GOCACHE=$TMPDIR/gc go vet ./... && gofmt -l main.go`
Expected: exit 0, no files listed by gofmt.

- [ ] **Step 3: Integration check on the local engine**

Run: `_EXPERIMENTAL_DAGGER_RUNNER_HOST=tcp://127.0.0.1:1234 dagger call site --source=. export --path=_site`
Expected: `_site/` exists with `index.html`, `docs.html`, `transparency.html`, `downloads.html`, `openapi.json`, `provenance.json`, `sdk/*.zip`, `frameworks/*/catalog.json`. Verify: `test -f _site/provenance.json && jq -e '.[0].signed==false' _site/provenance.json` (honesty holds — unsigned).

- [ ] **Step 4: Commit**

```bash
git add dagger/main.go
git commit -m "feat(portal): dagger Site orchestrator + downloads index"
```

---

### Task 9: GitHub Pages workflow + README link

**Files:**
- Create: `.github/workflows/pages.yml`
- Modify: `README.md` (add a Portal link line)

**Interfaces:**
- Consumes: `dagger call site` (Task 8).
- Produces: deployed Pages site on push to `main`.

- [ ] **Step 1: Create the workflow**

```yaml
name: Pages

on:
  push:
    branches: [main]
  workflow_dispatch: {}

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Dagger
        uses: dagger/dagger-for-github@v6
        with:
          install-only: true
          version: "v0.21.0"
      - name: Build site
        run: dagger call site --source=. export --path=_site
      - uses: actions/upload-pages-artifact@v3
        with:
          path: _site
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 2: Add README link**

Add under the README's top section:

```markdown
## Portal

Proto docs, supply-chain transparency, SDK clients, and OSCAL framework
downloads are published to GitHub Pages (built by `dagger call site`).
```

- [ ] **Step 3: Lint the workflow**

Run: `yamllint -d relaxed .github/workflows/pages.yml && echo OK`
Expected: `OK` (line-length warnings acceptable).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pages.yml README.md
git commit -m "feat(portal): GitHub Pages workflow (dagger-built) + README link"
```

---

## Self-Review

**Spec coverage:** proto docs (Task 5 docs.html + Task 6 openapi) ✓; transparency provenance (Tasks 1,4,7 + Task 5 transparency.html) ✓; SDK downloads (Task 6 + Task 8 index) ✓; OSCAL framework downloads (Tasks 2,3,7 + Task 8 index) ✓; Dagger Site + Pages publish (Tasks 8,9) ✓; DS-3 theming + digest law + honesty (Tasks 1,5) ✓.

**Placeholder scan:** no TBD/TODO; all Go and shell code is complete.

**Type consistency:** `portal.Artifact`/`BuildManifest` (Task 1) consumed by Task 4 CLI and Task 7 emits matching Go-default JSON keys (`Name`,`Digest`). `Manifest.Source/RefIDs` (Task 2) consumed by Task 3. `sdkBundles`/`oscalFrameworks`/`provenanceManifest` (Tasks 6–7) consumed by `Site` (Task 8). `oscal.ExportCatalogJSON`, `gen.GenerateAllArtifacts`, `store.CreateSnapshot`, `NewBulkIngestor`, `NewGitHubFetcher`, `NewReconciler`, `kg.NewSQLiteStore` all verified against the repo.

**Known runtime risks (validate during execution, not placeholders):**
- `gen.GenerateAllArtifacts` returns a result struct with a `.Catalog` field (confirmed in `xoscal-generate/main.go`); the exact result type name should be taken from `oscal/generator.go` at implementation time if a local var type is needed.
- `oscalFrameworks` and `sdkBundles` require network inside Dagger (buf.build plugins; GitHub fetch). Provide `GITHUB_TOKEN` as a Dagger secret if rate-limited (the fetcher reads `GITHUB_TOKEN`).
- The pinned Scalar standalone URL/version in Task 8 must be confirmed to resolve before relying on it (P-VW-003 — do not ship an unresolved CDN path). The `test -s /out/scalar.js` guard fails the build if the URL is wrong; verify the exact `@scalar/api-reference` dist path + a current version at implementation time, then pin it. For full integrity, follow up by committing the vendored bundle and a sha256 instead of fetching.
