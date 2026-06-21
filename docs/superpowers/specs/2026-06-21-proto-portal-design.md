# Proto Portal — Design

- Date: 2026-06-21
- Status: Approved (pending spec review)
- Branch: `chore/oscal-reconcile-loop`
- Scope: a static GitHub Pages portal that documents the OSCAL protos, shows
  build-time supply-chain provenance for the protos/SDKs, offers SDK client
  downloads, and offers the ciso-assistant frameworks converted to OSCAL for
  download.

## 1. Problem

The repo produces several artifacts (buf-generated OpenAPI/JSON-Schema, six
language SDKs, OSCAL conversions of framework libraries, and CI supply-chain
evidence) but exposes none of them to consumers. There is no single surface to:

1. read the proto/API documentation,
2. verify the supply-chain provenance ("transparency stock") of the protos and
   their SDKs,
3. download the SDK clients, or
4. download the frameworks converted to OSCAL.

## 2. Goals / Non-goals

Goals:
- One static site, reproducibly built, published to GitHub Pages.
- Surface artifacts the repo already generates; compute nothing new in the portal.
- Evidence-native and honest: show provenance only where it exists.

Non-goals (YAGNI):
- Live Transparency Exchange (TEA) data — the transparency view is a build-time
  snapshot.
- Search, versioned documentation history, authentication, dynamic backend.

## 3. Architecture (Four Spaces)

The portal lives in **Proof** (provenance manifest) and **Presentation**
(rendering + static site). No kernel/domain logic, no new business rules.

```
buf generate ───┐
ingest+export ──┼─► dagger Site() ─► static bundle ─► pages.yml ─► GitHub Pages
SBOM/SLSA/sign ─┘     (assemble + render)              (publish only)
```

All assembly logic is in Dagger; the GitHub Actions workflow only calls Dagger
and deploys (repo rule: no business logic in GHA/GitLab).

## 4. The four sections

| Section | Source (already produced) | Renderer |
|---|---|---|
| Proto docs | `proto/oscal/gen/openapi/*`, `gen/jsonschema/*` (buf) | Scalar UI (static HTML reading the OpenAPI doc) |
| Transparency | build-time SBOM (syft), SLSA provenance, cosign signature, Rekor log id | static page rendered from a generated `provenance.json` |
| SDK downloads | `proto/oscal/gen/{go,python,java,csharp,ts,swift}` (buf) | per-language `.zip` + sha256 sidecar + downloads index |
| OSCAL frameworks | `xoscal-ingest` → `server/internal/oscal/export.go` over `data/frameworks/manifest.yaml` | per-framework OSCAL `catalog.json` + download index |

Decision: proto docs use **Scalar UI** over the generated OpenAPI (already in the
stack and in `buf.gen.yaml`), not protoc-gen-doc. Decision: OSCAL frameworks are
emitted as **catalogs** in this iteration (profiles/SSP deferred).

## 5. Build pipeline — new Dagger functions

Added to `dagger/main.go`, matching existing conventions (`toolBase`, cache
volumes, `WithExec` style):

- `Site(source) *Directory` — orchestrator; composes the four sub-bundles plus
  the static `site/` shell into one directory.
- `sdkBundles(source) *Directory` — runs `buf generate`, zips each `gen/<lang>`
  directory, writes a `<lang>.zip.sha256` sidecar per zip.
- `oscalFrameworks(source) *Directory` — builds `xoscal-ingest`, runs it over
  `data/frameworks/manifest.yaml`, emits one OSCAL `catalog.json` per framework
  plus a `sha256` sidecar.
- `provenanceManifest(source) *Directory` — emits `provenance.json`: an array of
  `{name, digest, sbom_ref?, slsa_ref?, cosign_bundle?, rekor_id?, signed: bool}`
  records, one per artifact (the proto set and each SDK zip). Fields are
  populated ONLY where the evidence exists. Missing signing renders
  `signed: false` — never a fabricated attestation (P-VW-001 / Rule 12).

The site shell (`site/`) is hand-authored static HTML/CSS themed with
CKODEX-DS-3 "Evidence Editorial":
- Evidence Margin (`<aside>`) holds the provenance receipts.
- Digest law for all hashes: `sha256:9f3c…a217` (algorithm prefix + middle
  ellipsis), via the `.ck-hash` pattern.
- Only an **attested** artifact (proof object present) earns `proof.violet`;
  unsigned artifacts stay tone, not green.
- Square containers (`.ck-quiet`); the 10px chamfer (`.ck-sealed`) appears only
  on attested surfaces.

## 6. Publishing

New `.github/workflows/pages.yml`:
- Triggers: push to `main`, `workflow_dispatch`.
- Steps: setup Dagger → `dagger call site --source=. export --path=_site` →
  `actions/upload-pages-artifact` → `actions/deploy-pages`.
- Permissions: `pages: write`, `id-token: write`, `contents: read`.

## 7. Repo layout (new/changed)

```
site/
  index.html              # landing + nav to the four sections
  ds3.css                 # CKODEX-DS-3 subset
  docs.html               # Scalar UI over OpenAPI
  transparency.html       # renders provenance.json
  downloads.html          # SDK + OSCAL framework download index
dagger/main.go            # + Site, sdkBundles, oscalFrameworks, provenanceManifest
.github/workflows/pages.yml
docs/superpowers/specs/2026-06-21-proto-portal-design.md
```

## 8. Data flow

1. `Site()` calls `buf generate` once (via `sdkBundles` / shared) to populate
   `gen/`.
2. `provenanceManifest()` digests the proto set + each SDK zip; attaches SBOM /
   SLSA / cosign / Rekor refs where the build produced them.
3. `oscalFrameworks()` runs ingest→export to emit per-framework catalogs.
4. `Site()` copies the static `site/` shell, drops in the OpenAPI doc for
   Scalar, `provenance.json`, the SDK zips, and the OSCAL catalogs, and emits
   the download indexes.
5. The workflow exports `_site` and deploys to Pages.

## 9. Error handling / honesty

- A missing artifact (e.g. an SDK language fails to generate) fails the build —
  no silent partial site (Rule 12).
- Provenance fields absent ⇒ `signed: false` and no violet; never fabricated.
- Every downloadable has a sha256 sidecar; the download index digest must match
  the file digest (verified in the build).

## 10. Verification

- `go vet` / `gofmt` on the Dagger additions (local).
- `dagger call site --source=.` yields a non-empty bundle containing all four
  sections (run on the local engine: `_EXPERIMENTAL_DAGGER_RUNNER_HOST`).
- Honesty test: an unsigned artifact renders `signed: false`, no violet mark.
- Each SDK zip and OSCAL file has a matching sha256 sidecar.

## 11. Open follow-ups (out of scope here)

- OSCAL profiles/SSP emission in addition to catalogs.
- Live TEA claims drill-through (the layered transparency option).
- Wiring portal provenance to the cosign/Rekor signing that Release performs.
