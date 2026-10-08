# Static-first correction and release plan

Starting revision: `23c2d9f85c001a9a29344867c0dc5664df180921` on isolated
`fix/static-first-portal`. Existing working directories and immutable releases
remain intact. This plan extends the verified v0.3.2 release contract; it does
not treat local execution as hosted signing evidence.

## 1. Complete presentation production

Produce the Blueprint from tracked templates and fixture data with
`scripts/render-blueprint.py`. Check generated `site/portal.html` and
`portal-preview.html` for drift. Render all six views, 18 sections, cards,
tables, journeys and navigation before JavaScript runs. Native anchors and
visible default views provide no-JavaScript navigation. Optional JavaScript
selects existing views, applies filters and selects themes. Illustrative data
remains explicitly unverified; actual downloads link to the release inventory.

Produce the API reference from the actual four-service `openapi.json` through
`dagger/render_api_reference.py`: all 96 operations, descriptions, models and
stable anchors exist in HTML. Preserve Scalar as an enhancement. Transform
exactly pinned upstream source components before bundling to eliminate nested
main landmarks, retaining upstream bytes, transformation recipe, independent
replay hashes and receipts in mandatory frontend evidence.

Integrate both producers in `dagger/site.go`, `dagger/candidate.go` and
`dagger/contracts.go` before inventory, analysis and signing. Extend
`docs/release-policy.json`, frontend replay validation, semantic accessibility
lint and source-generator checks. Dependencies: existing pinned frontend npm
lock, four-service API producer, Python standard library and Dagger 0.21.7.

Acceptance: source drift and incomplete HTML are rejected; missing or altered
transform evidence blocks admission; generated pages work without JavaScript;
exact bundled Scalar has one main landmark throughout repeated dialog and
navigation lifecycles. No mutation occurs after the final signed manifest.

## 2. Close confirmed local filesystem boundaries

Retain the individual disposition of all 13 active gosec findings in the task
evidence. Five confirmed cases concern symlink reads outside an operator's
selected evidence or repository directory, in
`server/internal/{vectorstate,scaffold,bundle}`. Obtain an independent read-only
boundary and compatibility review before patching. Use filesystem-root
confinement rather than string-prefix checks; permit legitimate in-root paths
and reject escapes and unreadable required evidence. Ensure checksum errors
cannot disclose arbitrary sidecar contents or silently claim verification.

Add focused positive and adversarial tests at the real evaluator, repository
scanner and audit-bundle boundaries. Review related archive verification and
error propagation only where necessary to preserve the same contract. Correct
stale security/version/signing documentation without weakening analysis policy
or adding suppressions. Obtain a fresh independent postpatch bypass/regression
review and resolve supported findings before completion.

Acceptance: actual outside-file, outside-sidecar and repository-metadata
symlinks are rejected; ordinary and in-root supported inputs remain usable;
missing/tampered proof never produces a verified claim. No assertion of remote
exploitation is made for these local CLI findings.

## 3. Verify actual outputs

The user's additional duplication and UI-consistency requests add a dedicated
independent review of all eight routes: Overview, CLI Guide, API Docs, Downloads,
Live Review, Transparency, SBOM Validation and Design Blueprint, plus the home
alias. Preserve intentional shared navigation and responsive hidden controls.
Consolidate repeated downloadable cards into the canonical Downloads page;
retain complete compact inventory and proof metadata in Transparency. Give
pages distinct headings and keep raw assessments in native disclosures.

Use one canonical visible API presentation at a time, retaining complete plain
HTML and an explicitly selected interactive explorer. Align the Blueprint with
the shared primary navigation and page framing, and separate its sample
downloads navigation from release downloads. Enforce common typography, spacing,
theme tokens, cards, controls, focus, mobile behavior and reduced motion. Native
no-JavaScript views must flow without overlapping; tests must check geometry
and occlusion as well as DOM visibility. Inspect screenshots, including Vault,
high-contrast, mobile and no-JavaScript states, before the final release gate.

Run source checks, generator drift/negative tests, Go formatting, prescribed
Go tests/race/vet/build, protobuf checks, design/accessibility lint and Dagger's
authoritative contract/security/SBOM gates. Export a complete Dagger site and
production candidate using the appropriate engine. Test the promised SDK
formats with their clean consumers and preserve raw reports and exceptions.
Do not disable package signature verification or reduce vulnerability
thresholds to recover an environment failure.

Against the actual exported site, visit every page and source section at
desktop and mobile widths, with JavaScript both enabled and disabled. Exercise
keyboard focus, deep links, history, filters, themes, reduced motion, forced
colors, repeated Scalar dialogs, local assets, exact links and negative proof
cases. Record console/network failures, screenshots, coverage and exact source
revision. Obtain independent implementation review before publication.

Acceptance: all required checks pass on the exact candidate bytes; source
sections have no unexplained no-JavaScript coverage exclusions; no nested main
landmarks or blank card/table placeholders remain. Report every failed and
not-run check separately from passed checks.

## 4. Review, immutable release and live acceptance

Prepare focused commits and a draft PR with source/evidence links, then review
the exact head, reconcile required CI checks and advance only within the user's
explicit merge/release/deployment authorization. Select the next unused release
version after checking remote tags. Never overwrite existing tags/assets or
change signing identities, credentials, permissions or environment policies.

Run the existing hosted identity boundary for payload, final manifest and
transport attestations. Verify exact subjects and every mandatory file before
promotion. Deploy the admitted bytes through the existing authorized Pages
boundary, using its legitimate main-ref workflow when required by the current
environment policy. Independently verify public signatures, all release assets,
the complete OCI graph, SDK/install consumers and every live HTTP body against
the signed inventory; rerun browser acceptance on those live bytes.

Acceptance: the exact merged head has green required checks and independent
review; the new immutable release and live Pages bytes pass their complete
proof, download and presentation contracts. Hosted-only evidence is recorded
as such. An automatic approval denial is a blocker, never permission to bypass.

## Migration, rollback and risks

There is no state migration: static HTML and frontend evidence contracts gain
new producer outputs. Old signed v0.3.2 artifacts retain their original policy
and proofs. New releases enforce their own signed policy and added receipt
requirements. Rollback deploys a previously admitted immutable release through
the existing authorized workflow; it never rewrites a tag or signed asset.

Main risks are pinned upstream source drift, expanded initial HTML size,
filesystem-root compatibility, toolchain/cache/network failures, frontend
accessibility regressions and hosted workflow/environment constraints. Pin
transform preimages, test real consumers and negative paths, measure Pages
capacity, preserve diagnostics, and distinguish infrastructure recovery from
security-policy exceptions. Required shared skill reference resources that the
platform cannot resolve are recorded as unavailable rather than assumed read.
