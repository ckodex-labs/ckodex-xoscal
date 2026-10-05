# Release analysis admission policy

Dagger owns collection and admission. Raw reports are evidence, including failed
reports; the existence of a report grants no release eligibility. Policy version
1 is enforced by `scripts/analysis-gate.py` and tested by
`python3 scripts/test_analysis_gate.py` and `python3 scripts/test_archive_extract.py`.

| Analysis | Collection | Admission |
| --- | --- | --- |
| Go source | `dagger call security-reports --source=. export --path=analysis/source` | `dagger call security --source=.` |
| Exact supplied SBOM | `dagger call sbom-validation-reports --source=. --sbom=path/to/sbom.json export --path=analysis/sbom` | `dagger call sbom-artifact-validation --source=. --sbom=path/to/sbom.json` |
| Staged image tar | `dagger call image-analysis-reports --archive=image.tar export --path=analysis/image` | `dagger call image-analysis --source=. --archive=image.tar` |
| SDK/package ZIP or platform tar | `dagger call artifact-analysis-reports --archive=package.zip export --path=analysis/package` | `dagger call artifact-analysis --source=. --archive=package.zip` |
| CI image tar | same collection | `dagger call image-analysis-ci --source=. --archive=image.tar` |

`ArtifactSbom` inventories supplied binary/archive bytes. `ImageSbom` uses
Syft's explicit Docker archive source against supplied image bytes. Feed each
result to `SbomArtifactValidation`, and include input SBOM, assessment, raw
scanner reports, statuses and logs in the release inventory. The promotion
coordinator must evaluate all methods before signing or making artifacts
consumer-visible. On failure, export the collection method independently; its
output is available even when admission fails. Exporting raw evidence must not
trigger promotion. All collector and policy files must remain integrity-covered.

Source policy requires successful govulncheck symbol analysis and blocks every
called vulnerability, preserving the prior source scan behavior. govulncheck
JSON output does not use a nonzero findings status, so the policy inspects
symbol traces explicitly. gosec blocks HIGH severity with HIGH confidence,
retains lower confidence/severity findings and rejects Go processing errors,
missing reports, empty scan statistics or inconsistent JSON/SARIF counts.
Tracked JSON and SARIF preserve all source suppressions and their rationales.
A source suppression can admit a high finding only with an explicit nonempty
rationale; external broad suppressions cannot bypass the policy. There are no
default gosec exclusions beyond the existing generated proto folder exclusion.

Image release policy preserves the existing CRITICAL threshold with
`ignore-unfixed` semantics: CRITICAL findings with an available fix block.
Image CI policy preserves HIGH and CRITICAL regardless of fix availability.
Raw Trivy JSON contains all severities and unfixed findings; no findings are
removed to render a passing report. Trivy operational failure, unexpected tool
version, absent scan targets or malformed vulnerability records block both.
Pinned Trivy v0.72.0 uses official upstream binary archives, checked against
committed SHA-256 pins from the release's checksum file:

- Linux amd64: `bbb64b9695866ce4a7a8f5c9592002c5961cab378577fa3f8a040df362b9b2ea`.
- Linux arm64: `2ca2c023109c2db6b2b77366b6717291452d4531167377d95c79547f0c8e3467`.

The runtime checks the actual CLI version before scanning and limits Go runtime
parallelism to two workers. No source compilation is needed on the release path.
ZIP and tar inputs are extracted by `dagger/extract_archive.py`, which rejects
traversal, absolute paths, links, device members, duplicate normalized paths,
encrypted ZIP entries and oversized inventories. Extraction preserves only
ordinary execute bits recorded by tar or Unix ZIP entries, using owner-only
read/write permissions and never restoring setuid, setgid or sticky bits. ZIP
entries from non-Unix creators do not supply Unix execute permissions. This
retains byte-backed executable classification without executing artifact code.
The pinned [Trivy Go binary analyzer](https://raw.githubusercontent.com/aquasecurity/trivy/v0.72.0/pkg/fanal/analyzer/language/golang/binary/binary.go)
selects regular executable files and reads their embedded Go build metadata;
stripping those bits would silently omit ELF and MachO inventories. Failed extraction is preserved
with its status and log. Artifact admission requires actual versioned packages;
empty or unsupported inventories cannot pass. `ArtifactAnalysis` collects all
severities and uses the same existing CRITICAL/fixed release threshold.

Artifact collection invokes both `trivy fs` and `trivy rootfs` against the same
safe extraction. In pinned Trivy 0.72.0, [filesystem mode disables individual
package analyzers, while rootfs disables lockfile analyzers](https://raw.githubusercontent.com/aquasecurity/trivy/v0.72.0/pkg/commands/artifact/run.go).
Using only one would omit either delivered binaries or dependency declarations.
The collector retains both `trivy-{fs,rootfs}-results.json`, statuses and logs.
`dagger/artifact_inventory.py` combines their complete targets without dropping
findings or conflating overlapping observations; each target records its scan
mode and the combined document binds both original reports by SHA-256.
`artifact_gate` independently replays that combination and requires both tools'
successful statuses, the exact report subject and pinned version. A legitimate
empty individual mode is retained, but an empty or unsupported combined inventory
still blocks. Existing `trivy.status` and `trivy.log` describe the merge operation;
a successful merge alone grants no eligibility. These six additional raw files
are mandatory release evidence and the helper is integrity-covered. Scan counts
include observations from both modes, so overlapping packages are not distinct
component counts. All findings remain raw and the CRITICAL/fixed threshold is
unchanged.

`ArtifactSbomAnalysis` and `ImageSbomAnalysis` preserve original payload digest in
`subject.sha256` (one lowercase SHA-256 and newline), pinned Syft producer version
in `sbom-producer.txt`, the exact generated `sbom.cyclonedx.json`, raw assessment,
validation exit/log and admission receipt. The receipt carries both
`subject_sha256` and `sbom_sha256`. Release verification must recompute both from
the retained original payload and SBOM; a digest file is not a signature by
itself. Generic `SbomArtifactAnalysis(source, sbom)` assesses supplied SBOM bytes
but cannot establish an original-payload binding without these producer records.

SBOM policy requires a nonempty CycloneDX component inventory and a completed
NTIA assessment with an explicit roll-up. Every unsatisfied individual target
blocks admission. sbom-tools exit 1 may represent completed compliance errors;
other nonzero statuses are errors for this invocation. An error status, missing
verdict or contradictory roll-up blocks even if the report exists.
Warnings reported as unsatisfied targets also require review. No supplier,
manufacturer or other metadata is invented to make an SBOM pass.

`docs/SBOM-DISTRIBUTOR.json` identifies the release repository owner as the
actual distributor/repackager of its delivered bytes. The
[CycloneDX supplier definition](https://raw.githubusercontent.com/CycloneDX/specification/1.7/schema/bom-1.7.schema.json)
allows this role separately from manufacturer. `dagger/enrich_sbom.py` fills
missing supplier fields only for hashed delivered files and narrowly allowed
catalogers that observe compiled/installed package bytes. It preserves existing
upstream suppliers, manufacturers, package identities and dependency edges.
It never assigns this identity to POM, go.mod, package.json or other declaration
catalogers. Root containment edges reference only observed packaged components;
all references must resolve to the retained inventory.

Bound SBOM analysis retains `sbom.raw.cyclonedx.json`, `sbom-distributor.json`
and `sbom-enrichment.json`. The receipt binds the original subject, raw Syft
report, final SBOM and distributor policy by SHA-256. Admission independently
replays enrichment and compares the complete resulting document and component
lists. These proof files require the same manifest coverage as the assessment.
`dagger/publisher_metadata.py` resolves declaration-only authors from exact
version primary metadata: PyPI author fields, npm author fields, NuGet nuspec
authors, Maven POM developers or producer organization (including an evidenced
parent POM chain), and actual Go source AUTHORS/explicit named Authors notices.
It retains each document, canonical package/version source URL and digest under
`publishers/`. Missing fields, unsupported identities, wrong versions, unsafe
XML entities, oversized metadata and failed fetches never invent an author.
The enrichment gate re-extracts each field from those retained documents.
Unpublished SDK identities can use their delivered `SDK-PRODUCER.json` only when
its canonical repository entity, exact package name and version agree. The
consumer verifier and release manifest bind that producer document to the ZIP.
No namespace, maintainer username or copyright holder alone becomes a supplier. Local unregistered image archives receive
[generic PURLs](https://raw.githubusercontent.com/package-url/purl-spec/main/types/generic-definition.json)
from their observed file name and content digest version; observed installed OS
release names/versions receive generic identities. These identifiers have no
registry claim. This repairs the pinned validator's unsupported SWID parsing
without discarding the retained source SWID or inventing an NVD CPE.

Unresolved dependency-declaration authors/suppliers remain actual compliance
blockers; no package or finding is removed to obtain a passing result. Trivy's
unversioned `Relationship=root` project entries remain in the raw report and
receipt, while admission requires a nonempty named and versioned dependency
inventory. Such roots do not count as scanned dependencies; an unversioned
dependency still blocks admission.

The versioned `docs/analysis-exceptions.json` initially contains no exceptions.
A reviewer may approve an individual SBOM finding only by recording all of:
`id`, `approved_by`, `rationale`, ISO `expires` date, exact `sbom_sha256`, and
`finding_sha256`. The finding digest is SHA-256 over canonical sorted JSON of
its `title`, `description`, and `target` (compact separators), independent of
run-specific UUIDs. Whitespace changes to the input SBOM change its digest.
An exception never covers a new input or a different finding. Expired,
duplicate, malformed or unattributed exceptions fail closed. Exception changes
require code review; the pipeline has no auto-waiver or broad rule exception.

Tool contract sources inspected for this policy:

- [gosec v2.28.0 source and exit behavior](https://github.com/securego/gosec/tree/v2.28.0), also available in the local Go module cache.
- [govulncheck v1.7.0 output and exit documentation](https://github.com/golang/vuln/blob/v1.7.0/cmd/govulncheck/doc.go), also inspected from the Go module cache.
- [sbom-tools validation and exit contract](https://github.com/sbom-tool/sbom-tools#exit-codes), retaining the repository's checksum-pinned v0.2.0 binary.
- [Trivy v0.72.0](https://github.com/aquasecurity/trivy/releases/tag/v0.72.0), with `pkg/flag`, `pkg/types/report.go`, `pkg/version/app` and `goreleaser.yml` inspected from the version-specific Go module cache.

SDK archive analysis covers the exact delivered ZIP/native package bytes and
the dependency declarations or lockfile versions identifiable in those bytes.
It does not certify every future consumer resolution, a different package feed,
a changed runtime/framework, or transitive dependencies absent from those
files. In particular, a Maven POM with direct dependencies is not a complete
runtime dependency lock. Clean consumer installation evidence is a separate
proof of the tested environment; its success is not a vulnerability scan of
all future consumers. Preserve raw target/package lists to show the actual
analysis scope and fail the stated vulnerability policy when those findings
exceed its threshold.

Local policy tests prove admission behavior. Hosted tool execution, current
vulnerability database acquisition and artifact signing remain separate proof
requirements; none is inferred from these tests.

The runtime uses the official pinned `static-debian13:nonroot` multi-platform
image because all release binaries use `CGO_ENABLED=0`. This removes the unused
OpenSSL runtime library present in the previous Debian 12 base. The actual
previous-image CI scan found HIGH CVE-2026-84782; the
[Debian tracker](https://security-tracker.debian.org/tracker/CVE-2026-84782)
reports bookworm-security affected while trixie-security is fixed. The
[distroless static Go example](https://github.com/GoogleContainerTools/distroless)
uses this runtime class. Scan the newly built exact image again; a base choice
or digest alone is not a successful vulnerability gate.


Exact source dependency attribution retains original registry metadata and package coordinates.
For Go dependencies whose source does not identify original authors, the producer may
identify `Go module mirror (proxy.golang.org)` only after actually downloading the
exact module source ZIP and verifying Go's `h1` content hash against the delivered
SDK `go.sum`. The complete ZIP and original `go.sum` remain evidence. This identifies
an actual source distributor; the unknown upstream author/manufacturer stays unknown.
The [Go module reference](https://go.dev/ref/mod#module-proxy) defines the versioned
ZIP protocol and [Go modules services](https://proxy.golang.org/) describes the mirror.
For Swift, the original source author header is fetched at the exact commit/version
in the delivered `Package.resolved`; that lockfile and original source bytes survive
independent replay. Native SDK producer metadata only identifies the repository's
own package, including its genuine release version; it never labels third-party dependencies.
Python direct dependency edges come from the exact delivered, SHA-256-backed
`requirements.txt` declarations. These source edges assert declarations, not bundled
third-party runtime code or an inferred transitive graph.

The pinned validator's NTIA roll-up is satisfied when it reports zero errors, even
when it has warnings. Admission verifies its counts against observation severities
and still blocks every unsatisfied target, including warnings. A satisfied roll-up
alone never substitutes for strict admission.

When an npm version lacks an original author, its explicit `_npmUser.name` can
identify a supplier as `npm publishing account: <name>`; this retains the actual
publishing actor and registry record, and leaves original authors/manufacturers
unknown. This can be an automation account. The [npm package metadata specification](https://github.com/npm/registry/blob/main/docs/responses/package-metadata.md)
defines that exact-version publisher field; current maintainers and repository
names are never substituted for it.

## Catalog and profile semantic admission

`OscalConstraintReports` records each catalog/profile CLI exit status, exact input
SHA-256 and full raw log. `OscalConstraintValidation` now blocks every nonzero
status and empty inventories. The prior nonblocking semantic warning mode is
removed. All 35 catalogs and the PBMM profile require successful JSON Schema
and Metaschema checks; required-release admission replays the complete 36-input
result set and hashes against the actually delivered framework bytes. Raw
collection remains independently exportable when admission fails.

## Frontend browser assembly

`FrontendProductionReports` verifies the exact `@scalar/api-reference@1.72.4`
registry metadata and tarball SRI, plus the original standalone JS/map SHA-256.
It rebuilds published ES modules with the reviewed public-registry npm lock,
architecture-pinned Node 22 toolchain and esbuild 0.25.12. Exact overrides pin
nanoid 5.1.16, unhead 2.1.13 and @ai-sdk/provider-utils 4.0.33. Dependency scripts
are disabled during `npm ci`; no user registry configuration or credentials are
required. These are repository-produced browser bytes, not the publisher's CDN
bundle. [npm ci](https://docs.npmjs.com/cli/v11/commands/npm-ci/) enforces the frozen
lock and registry integrity; [npm overrides](https://docs.npmjs.com/cli/v11/configuring-npm/package-json/#overrides)
make the three security replacements explicit.

The runtime inventory uses esbuild output inputs with `bytesInOutput > 0`, for
both JS and the CSS injected into its pre-compilation banner. Parsed modules
removed by tree shaking are retained as build evidence, but are not claimed as
runtime constituents. Nested installed packages keep their actual name/version
from the closest package manifest and matching exact lock entry. Peer-context
names from the original pnpm source map are never mistaken for bundled modules.
This is an observed assembly inventory; it does not assert a complete upstream
transitive dependency graph. The [esbuild metafile contract](https://esbuild.github.io/api/#metafile)
provides the input/output attribution used here.

The source-map producer deterministically removes upstream `sourceMappingURL`
comments before compiling JS inputs; the final map contains the exact compiled
input text under that declared transformation. Raw input bytes are retained in a
bounded deterministic tar, and JS/CSS maps are replayed against them. This avoids
claiming complete upstream TypeScript maps when publisher maps omit source text.
The exact CSS output is checked against the injected JS banner. No JS bytes are
modified after source-map generation. Agent and telemetry are explicitly disabled,
with no request proxy configured. The existing local `data-url` mount remains.

The exact registry tar, reviewed build scripts/lock, toolchain/time, metafile,
installed manifests, raw input archive, SHA records, original Scalar license
source, actual included package notices, assembled notices, SBOM, scanner log,
scanner status and all-severity Trivy JSON remain in `evidence/frontend`.
Original npm tar integrity is replayed; it is not a claim that npm provenance
signatures were verified. The Scalar upstream license is pinned to commit
0962afc399ddc5a76f4116a67dda30286a859d56 and its exact SHA-256/git blob SHA-1.
Unknown upstream authors remain unknown. The SBOM supplier identifies ckodex-labs
as the actual distributor of the assembled browser output, not each constituent's
original author or manufacturer.

`frontend-release` rejects missing/failed build or scanner results, malformed
or empty inventories, unreviewed builder/lock inputs, unmapped package sources,
tampered bytes and incomplete scan coverage. Pinned Trivy 0.72.0 scans the observed
CycloneDX components and preserves every severity, including unfixed findings.
Admission uses the existing release `CRITICAL/fixed` threshold; lower findings
are not suppressed or waived on guessed browser/Node applicability. Separate
strict SBOM admission assesses the exact generated frontend SBOM. Required release
verification replays both analyses and compares the served JS/maps/notices with
their admitted evidence before publication.
