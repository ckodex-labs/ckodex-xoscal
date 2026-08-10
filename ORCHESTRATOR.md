# ORCHESTRATOR — xOSCAL coding domain v1

This file is the repository-bound execution contract for work in
`ckodex-oscalify`. It adapts the generic orchestrator prompt to the actual
checkout: a Go gRPC service, an OSCAL 1.1.2 artifact pipeline, a Dagger build
graph, and a static GitHub Pages portal.

## 0. Runtime context

```yaml
domain: coding
repository: ckodex-oscalify
workspace_boundary: local checkout; preserve unrelated user changes
artifact_of_record: git diff and committed tree
primary_runtime: Go 1.24+ service and Dagger SDK pipeline
data_contract: OSCAL 1.1.2 JSON schema plus Metaschema constraints
presentation_runtime: static HTML/CSS/JavaScript under site/
portal_entrypoints:
  live_build: site/portal.html assembled by dagger call site
  standalone_preview: portal-preview.html
devex_entrypoints: Makefile, dagger/main.go, scripts/
design_system: CKODEX-DS-3 Evidence Editorial
proof_boundary: local tests and rendered checks are local evidence; hosted,
  release, signing, and external acceptance remain separate gates
```

The checkout is not a React, Vite, or browser-bundled application. Do not
invent those layers. UI changes target the static HTML, the canonical
`site/styles.css` entry point, the Dagger site assembly, and the standalone
preview together.

## 1. Objective

Deliver the requested bounded slice to a verified state. “Complete” means the
diff exists, the changed artifact is rendered or executed, the repository
gates pass, and remaining external gates are named. A plan, code snippet,
green unit test, or prior report is not completion evidence by itself.

For UI, “no gaps” means every page has semantic landmarks, a skip link,
keyboard-visible focus, a persistent Evidence Margin, an Authority Footer,
all four DS-3 themes, reduced-motion and forced-colors behavior, and no
invented proof state. For DevEx, it means the documented command is runnable,
deterministic, and included in the same validation path as the code it
protects.

## 2. Dispatch contract

```text
WORK = {objective, scope, constraints, current_state, evidence_needed}
DISPATCH(WORK) -> {mechanical | builder | reasoner | cross-peer}

mechanical: exact search, formatting, schema checks, deterministic transforms,
  HTML/DOM lint, artifact inventory, and command execution
builder: Go, proto, Dagger, HTML, CSS, and JavaScript implementation to a
  decided contract
reasoner: architecture, security, OSCAL semantics, evidence boundaries, and
  ambiguity resolution
cross-peer: independent review of the diff, failure paths, accessibility,
  and proof claims; use a separate worktree or an explicitly independent
  review pass when available
```

Dispatch is a policy decision. A model does not route work, manufacture test
results, or promote an artifact. Use the least expensive role that can clear
the evidence threshold; route hard reasoning to the reasoner role.

Every dispatched slice carries:

```yaml
slice:
  objective: concrete outcome
  file_context: [absolute or repository-relative file paths and line anchors]
  constraints: [security, compatibility, design, scope]
  acceptance: [command, DOM assertion, screenshot, artifact, or review result]
  budget: {tool_calls: bounded, wall_clock_s: bounded, changed_loc: bounded}
  handoff: {changed_files: [], commands_run: [], evidence_refs: [], risks: []}
```

If a required file, tool, schema, release input, or external authority is
missing, stop at that boundary and label the result `S` (specified/partial) or
`A` (aspirational). Do not replace missing evidence with a mock or a claim.

## 3. Repository map and ownership

| Surface | Owner | Required proof |
| --- | --- | --- |
| `server/` | Go implementation | `go test`, race, vet, build |
| `proto/` | API contract and generated code | `buf lint`, format, generation drift |
| `server/internal/oscal/` | OSCAL transformation | generator/e2e/schema tests |
| `server/internal/schemavalidate/` | structural validation | schema and conformance tests |
| `site/` | live static portal | design lint, site build, HTTP smoke |
| `site/styles.css` | DS-3 canonical stylesheet | theme, accessibility, glyph, asset checks |
| `portal-preview.html` | standalone UI preview | same design lint and rendered smoke |
| `dagger/main.go` | isolated CI and site assembly | `dagger call all --source=.` |
| `Makefile` and `scripts/` | local DevEx | command-level smoke and failure output |
| `docs/` | contracts and operator guidance | link/path consistency and review |

Generated artifacts are edited only through their generator or an explicitly
bounded repair. Preserve unrelated user modifications and untracked files.

## 4. Evidence and claim discipline

Use the following labels in handoffs and reports:

- `C` — implemented, typed or structurally valid, tested, and enforceable in
  this checkout.
- `S` — design locked or partially implemented; a named gate remains.
- `A` — aspirational or dependent on an external system not exercised here.

Every `C` claim names an evidence path or command. Coverage, performance,
release, signing, hosted deployment, and external service claims require their
own artifact. A local green test does not prove hosted or production state.

For UI, evidence must include both static assertions and a rendered HTTP pass:

```yaml
ui_evidence:
  static: scripts/design-lint.py output
  build: make site or dagger call site --source=.
  smoke: scripts/site-smoke.py output against the served artifact
  visual: screenshot or browser inspection when the environment permits
  boundary: local/beta proof versus hosted/release acceptance
```

Proof coloring is semantic: violet requires a real digest/signature object;
red requires an active emergency or quarantine state; routine status remains
ink or tone. Never turn sample data into an attestation.

## 5. Verification ladder

Run the narrowest relevant checks while iterating, then the full ladder before
handoff:

```sh
gofmt -d .
buf lint
buf format -d --exit-code
go test ./...
go test -race ./...
go vet ./...
go build ./...
python3 scripts/design-lint.py
python3 scripts/site-smoke.py
dagger call all --source=.
```

The repository's Dagger `All` pipeline remains authoritative for isolated
lint, tests, race, security, proto drift, spec registry, structural OSCAL
schema validation, and Metaschema validation. If a tool is unavailable, report
the exact skipped gate and its classification.

## 6. Merge and release rules

1. Inspect status, diff, branch, and exact head before editing.
2. Keep the change bounded; do not reset, clean, or overwrite unrelated work.
3. Run the relevant local gates and record their outputs.
4. Review the final diff for phantom symbols, silent stubs, fake evidence,
   unbounded claims, and accidental credentials.
5. Commit only after the requested commit boundary is explicit.
6. Push only after the requested push boundary is explicit and the exact
   pushed head is verified against the remote.
7. Merge or promote only after required checks are green on that exact head;
   hosted policy, human review, signing, and release acceptance remain named
   gates rather than inferred outcomes.

## 7. Handoff format

```text
VERDICT: C | S | A
OBJECTIVE: <bounded result>
CHANGED: <files and reason>
VERIFIED: <exact commands and outcomes>
RENDERED: <served URL, DOM checks, screenshot path, or unavailable>
EVIDENCE: <artifact paths and digests where applicable>
UNTESTED: <explicit remaining gates>
RISKS: <security, compatibility, release, or provenance risks>
NEXT: <one safe next action>
```

Do not use “done”, “production-ready”, “robust”, or equivalent completion
language without the evidence required above.
