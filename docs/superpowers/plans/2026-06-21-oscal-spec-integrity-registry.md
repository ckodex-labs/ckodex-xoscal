# OSCAL Spec Integrity Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep the protos provably lock-step with the OSCAL specs via a committed per-model schema-hash registry that detects upstream drift per model.

**Architecture:** A pure `specregistry` package (parse/hash/compare), a `xoscal-spec-registry` CLI (`populate` writes hashes from a pinned OSCAL release, `verify` re-fetches and compares → drift exit code), a committed `data/oscal/spec-registry.yaml`, and reconcile/CI wiring (a dagger gate + a workflow verify step). Release-asset downloads use plain `net/http` (the existing fetcher only handles repo blobs).

**Tech Stack:** Go 1.25, gopkg.in/yaml.v3, net/http, Dagger (Go SDK).

## Global Constraints

- Go 1.25; `gofmt` clean; `go vet ./...` clean; errors wrapped `fmt.Errorf("op: %w", err)`.
- Digest format: `sha256:<hex>` (lowercase hex), matching the repo convention.
- Schema asset URL: `https://github.com/usnistgov/OSCAL/releases/download/v{version}/{asset}`.
- Honesty (Rule 12 / P-VW-003): a missing/renamed upstream asset is a HARD failure naming the model + URL — never silently skipped. The registry pins bytes and detects upstream change; it is NOT a semantic proto↔schema conformance proof.
- Exit-code contract for `verify` (mirrors `poll-feeds.sh`): 0 = all match, 3 = drift, 1 = operational error.
- `common.proto` is excluded (no upstream OSCAL model). Models covered: catalog, profile, ssp, component-definition, assessment-plan, assessment-results, poam, mapping.
- Branch: `feat/oscal-spec-integrity-registry`. Commit after every task.
- Network-dependent steps (populate/verify against GitHub, dagger calls) run in CI/engine, not the local sandbox.

## File Structure

- `server/internal/specregistry/registry.go` — types + Parse/Serialize/SchemaURL/Hash (pure).
- `server/internal/specregistry/verify.go` — ModelStatus/VerifyResult + Aggregate (pure).
- `server/cmd/xoscal-spec-registry/main.go` — CLI: populate + verify (net/http I/O).
- `data/oscal/spec-registry.yaml` — the committed per-model registry.
- `recipes/oscal.yaml` — add `spec_registry:` pointer.
- `data/oscal/upstream.lock.yaml` — remove the `schema_sha256` field.
- `dagger/main.go` — add `SpecRegistryCheck`, wire into `All`.
- `.github/workflows/oscal-reconcile.yml` — add a hash-verify step.

---

### Task 1: `specregistry` core (types, parse, URL, hash)

**Files:**
- Create: `server/internal/specregistry/registry.go`
- Test: `server/internal/specregistry/registry_test.go`

**Interfaces:**
- Produces: `type Model struct { Model, SchemaAsset, SchemaSHA256, Proto string }` (yaml+json tags `model`,`schema_asset`,`schema_sha256`,`proto`); `type Registry struct { Version, OSCALVersion, GeneratedAt string; Models []Model }` (yaml tags `version`,`oscal_version`,`generated_at`,`models`); `func Parse(b []byte) (*Registry, error)`; `func (r *Registry) Serialize() ([]byte, error)`; `func SchemaURL(oscalVersion, asset string) string`; `func Hash(data []byte) string`.

- [ ] **Step 1: Write the failing test**

```go
package specregistry

import (
	"strings"
	"testing"
)

func TestSchemaURL(t *testing.T) {
	got := SchemaURL("1.2.2", "oscal_catalog_schema.json")
	want := "https://github.com/usnistgov/OSCAL/releases/download/v1.2.2/oscal_catalog_schema.json"
	if got != want {
		t.Errorf("SchemaURL = %q, want %q", got, want)
	}
}

func TestHash_Format(t *testing.T) {
	h := Hash([]byte("hello"))
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("Hash missing sha256: prefix: %q", h)
	}
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	if h != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("Hash = %q", h)
	}
}

func TestParseSerialize_Roundtrip(t *testing.T) {
	in := []byte("version: \"1.0\"\noscal_version: \"1.2.2\"\ngenerated_at: \"\"\nmodels:\n  - model: catalog\n    schema_asset: oscal_catalog_schema.json\n    schema_sha256: \"\"\n    proto: proto/oscal/catalog/v1/catalog.proto\n")
	r, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.OSCALVersion != "1.2.2" || len(r.Models) != 1 || r.Models[0].Model != "catalog" {
		t.Fatalf("parsed wrong: %+v", r)
	}
	out, err := r.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if r2, err := Parse(out); err != nil || r2.Models[0].SchemaAsset != "oscal_catalog_schema.json" {
		t.Fatalf("roundtrip failed: %v %+v", err, r2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ -v`
Expected: FAIL — `undefined: SchemaURL` / `undefined: Hash` / `undefined: Parse`.

- [ ] **Step 3: Write minimal implementation**

```go
// Package specregistry holds the OSCAL spec integrity registry: per-model
// upstream schema hashes bound to the protos that implement them. Pure logic;
// the CLI wraps it with network I/O.
package specregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Model binds one OSCAL model's upstream schema (and its hash) to a proto file.
type Model struct {
	Model        string `yaml:"model" json:"model"`
	SchemaAsset  string `yaml:"schema_asset" json:"schema_asset"`
	SchemaSHA256 string `yaml:"schema_sha256" json:"schema_sha256"`
	Proto        string `yaml:"proto" json:"proto"`
}

// Registry is the committed per-model integrity ledger for one OSCAL version.
type Registry struct {
	Version      string  `yaml:"version"`
	OSCALVersion string  `yaml:"oscal_version"`
	GeneratedAt  string  `yaml:"generated_at"`
	Models       []Model `yaml:"models"`
}

// Parse decodes a registry from YAML.
func Parse(b []byte) (*Registry, error) {
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &r, nil
}

// Serialize encodes the registry to YAML.
func (r *Registry) Serialize() ([]byte, error) {
	b, err := yaml.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("serialize registry: %w", err)
	}
	return b, nil
}

// SchemaURL builds the GitHub release-asset URL for an OSCAL model schema.
func SchemaURL(oscalVersion, asset string) string {
	return fmt.Sprintf("https://github.com/usnistgov/OSCAL/releases/download/v%s/%s", oscalVersion, asset)
}

// Hash returns the sha256 of data in the repo's "sha256:<hex>" format.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add server/internal/specregistry/registry.go server/internal/specregistry/registry_test.go
git commit -m "feat(specregistry): registry types, parse, schema URL, hash"
```

---

### Task 2: `specregistry` verify aggregation

**Files:**
- Create: `server/internal/specregistry/verify.go`
- Test: `server/internal/specregistry/verify_test.go`

**Interfaces:**
- Consumes: `Hash` (Task 1).
- Produces: `type ModelStatus struct { Model, Expected, Actual, Status string }` (json tags `model`,`expected`,`actual`,`status`; Status is `"match"` or `"drift"`); `type VerifyResult struct { OSCALVersion string; Drift bool; Models []ModelStatus }` (json tags `oscal_version`,`drift`,`models`); `func CompareModel(model, expected string, fetched []byte) ModelStatus` (Status="drift" unless `Hash(fetched)==expected`); `func Aggregate(oscalVersion string, statuses []ModelStatus) VerifyResult` (Drift=true if any status is "drift"); `func (v VerifyResult) ExitCode() int` (0 if no drift, 3 if drift).

- [ ] **Step 1: Write the failing test**

```go
package specregistry

import "testing"

func TestCompareModel_MatchAndDrift(t *testing.T) {
	data := []byte("schema-bytes")
	expected := Hash(data)
	if s := CompareModel("catalog", expected, data); s.Status != "match" {
		t.Errorf("match case Status = %q", s.Status)
	}
	s := CompareModel("ssp", expected, []byte("different"))
	if s.Status != "drift" {
		t.Errorf("drift case Status = %q", s.Status)
	}
	if s.Expected != expected || s.Actual != Hash([]byte("different")) {
		t.Errorf("status hashes wrong: %+v", s)
	}
}

func TestAggregate_AndExitCode(t *testing.T) {
	clean := Aggregate("1.2.2", []ModelStatus{{Status: "match"}, {Status: "match"}})
	if clean.Drift || clean.ExitCode() != 0 {
		t.Errorf("clean: drift=%v exit=%d", clean.Drift, clean.ExitCode())
	}
	dirty := Aggregate("1.2.2", []ModelStatus{{Status: "match"}, {Status: "drift"}})
	if !dirty.Drift || dirty.ExitCode() != 3 {
		t.Errorf("dirty: drift=%v exit=%d", dirty.Drift, dirty.ExitCode())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ -run 'Compare|Aggregate' -v`
Expected: FAIL — `undefined: CompareModel` / `undefined: Aggregate`.

- [ ] **Step 3: Write minimal implementation**

```go
package specregistry

// ModelStatus is the per-model outcome of a verify run.
type ModelStatus struct {
	Model    string `json:"model"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Status   string `json:"status"` // "match" | "drift"
}

// VerifyResult aggregates per-model statuses for one OSCAL version.
type VerifyResult struct {
	OSCALVersion string        `json:"oscal_version"`
	Drift        bool          `json:"drift"`
	Models       []ModelStatus `json:"models"`
}

// CompareModel hashes the fetched schema and compares it to the expected hash.
func CompareModel(model, expected string, fetched []byte) ModelStatus {
	actual := Hash(fetched)
	status := "drift"
	if actual == expected {
		status = "match"
	}
	return ModelStatus{Model: model, Expected: expected, Actual: actual, Status: status}
}

// Aggregate folds per-model statuses into a result; Drift is true if any drifted.
func Aggregate(oscalVersion string, statuses []ModelStatus) VerifyResult {
	drift := false
	for _, s := range statuses {
		if s.Status == "drift" {
			drift = true
		}
	}
	return VerifyResult{OSCALVersion: oscalVersion, Drift: drift, Models: statuses}
}

// ExitCode maps a result to the verify exit-code contract (0 ok, 3 drift).
func (v VerifyResult) ExitCode() int {
	if v.Drift {
		return 3
	}
	return 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ -v`
Expected: PASS (all tests).

- [ ] **Step 5: Commit**

```bash
git add server/internal/specregistry/verify.go server/internal/specregistry/verify_test.go
git commit -m "feat(specregistry): per-model compare + drift aggregation + exit codes"
```

---

### Task 3: The committed registry file

**Files:**
- Create: `data/oscal/spec-registry.yaml`

**Interfaces:**
- Produces: a registry parseable by `specregistry.Parse` with all 8 models, empty `schema_sha256` (filled by `populate`), correct proto bindings.

- [ ] **Step 1: Create the file**

```yaml
# OSCAL spec integrity registry.
#
# Per-model upstream OSCAL schema hashes bound to the protos that implement them.
# Keeps the protos provably lock-step with a pinned OSCAL release: `verify`
# re-fetches each model's schema and compares to schema_sha256; a mismatch is
# per-model drift (re-tag or silent upstream change).
#
# Boundary (honest): this pins the exact upstream schema BYTES and detects
# upstream change per model. It does NOT prove a proto semantically conforms to
# its schema — that stays the oscal-schema-reconciler agent + buf. The registry
# is the trigger/anchor ("model X drifted -> re-verify proto X").
#
# common.proto is intentionally excluded (shared types, no upstream OSCAL model).
# schema_sha256 fields are empty until `xoscal-spec-registry -mode populate` runs
# against a pinned release (network required; runs in CI/engine, not the sandbox).
version: "1.0"
oscal_version: "1.2.2"
generated_at: ""
models:
  - model: catalog
    schema_asset: oscal_catalog_schema.json
    schema_sha256: ""
    proto: proto/oscal/catalog/v1/catalog.proto
  - model: profile
    schema_asset: oscal_profile_schema.json
    schema_sha256: ""
    proto: proto/oscal/profile/v1/profile.proto
  - model: ssp
    schema_asset: oscal_ssp_schema.json
    schema_sha256: ""
    proto: proto/oscal/ssp/v1/ssp.proto
  - model: component-definition
    schema_asset: oscal_component_schema.json
    schema_sha256: ""
    proto: proto/oscal/component_definition/v1/component.proto
  - model: assessment-plan
    schema_asset: oscal_assessment-plan_schema.json
    schema_sha256: ""
    proto: proto/oscal/assessment_plan/v1/assessment_plan.proto
  - model: assessment-results
    schema_asset: oscal_assessment-results_schema.json
    schema_sha256: ""
    proto: proto/oscal/assessment_results/v1/assessment_results.proto
  - model: poam
    schema_asset: oscal_poam_schema.json
    schema_sha256: ""
    proto: proto/oscal/poam/v1/poam.proto
  - model: mapping
    schema_asset: oscal_mapping_schema.json
    schema_sha256: ""
    proto: proto/oscal/mapping/v1/mapping.proto
```

- [ ] **Step 2: Verify it parses and lints**

Run:
```bash
yamllint -d relaxed data/oscal/spec-registry.yaml && echo YAMLLINT_OK
GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ >/dev/null && echo "pkg ok"
```
Then a focused parse check (proves the committed file is consumable):
```bash
GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go run ./server/cmd/xoscal-spec-registry -mode print -registry data/oscal/spec-registry.yaml 2>/dev/null || echo "(CLI added in Task 4; parse via package test for now)"
```
Expected: `YAMLLINT_OK` and `pkg ok`. (The CLI does not exist yet; the file is validated structurally here and consumed in Task 4.)

- [ ] **Step 3: Commit**

```bash
git add data/oscal/spec-registry.yaml
git commit -m "feat(specregistry): committed per-model registry skeleton (OSCAL 1.2.2)"
```

---

### Task 4: `xoscal-spec-registry` CLI

**Files:**
- Create: `server/cmd/xoscal-spec-registry/main.go`

**Interfaces:**
- Consumes: `specregistry.{Parse, SchemaURL, Hash, CompareModel, Aggregate, Registry, Model, VerifyResult}` (Tasks 1–2).
- Produces: CLI `xoscal-spec-registry -mode populate|verify -registry <path> [-version <v>]`. `populate` rewrites the registry with fetched hashes + `generated_at`. `verify` prints a `VerifyResult` JSON and exits 0/3/1.

- [ ] **Step 1: Write the implementation**

```go
// Command xoscal-spec-registry keeps the protos lock-step with the OSCAL specs.
//
//	-mode populate : fetch each model's schema for the pinned version, hash it,
//	                 write schema_sha256 + generated_at into the registry.
//	-mode verify   : re-fetch and compare to the registry; print a JSON report.
//	                 exit 0 = all match, 3 = drift, 1 = operational error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mchorfa/xoscal/server/internal/specregistry"
)

func main() {
	mode := flag.String("mode", "", "populate | verify")
	registryPath := flag.String("registry", "data/oscal/spec-registry.yaml", "Registry path")
	version := flag.String("version", "", "OSCAL version (default: registry's oscal_version)")
	flag.Parse()

	code, err := run(*mode, *registryPath, *version)
	if err != nil {
		log.Print(err)
	}
	os.Exit(code)
}

func run(mode, registryPath, version string) (int, error) {
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return 1, fmt.Errorf("read registry: %w", err)
	}
	reg, err := specregistry.Parse(raw)
	if err != nil {
		return 1, err
	}
	if version == "" {
		version = reg.OSCALVersion
	}

	switch mode {
	case "populate":
		return populate(reg, registryPath, version)
	case "verify":
		return verify(reg, version)
	default:
		return 1, fmt.Errorf("invalid -mode %q (want populate|verify)", mode)
	}
}

func populate(reg *specregistry.Registry, path, version string) (int, error) {
	for i := range reg.Models {
		data, err := fetch(specregistry.SchemaURL(version, reg.Models[i].SchemaAsset))
		if err != nil {
			return 1, fmt.Errorf("populate %s: %w", reg.Models[i].Model, err)
		}
		reg.Models[i].SchemaSHA256 = specregistry.Hash(data)
	}
	reg.OSCALVersion = version
	reg.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	out, err := reg.Serialize()
	if err != nil {
		return 1, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return 1, fmt.Errorf("write registry: %w", err)
	}
	fmt.Printf("populated %d models for OSCAL v%s\n", len(reg.Models), version)
	return 0, nil
}

func verify(reg *specregistry.Registry, version string) (int, error) {
	var statuses []specregistry.ModelStatus
	for _, m := range reg.Models {
		data, err := fetch(specregistry.SchemaURL(version, m.SchemaAsset))
		if err != nil {
			return 1, fmt.Errorf("verify %s: %w", m.Model, err)
		}
		statuses = append(statuses, specregistry.CompareModel(m.Model, m.SchemaSHA256, data))
	}
	result := specregistry.Aggregate(version, statuses)
	report, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return 1, fmt.Errorf("marshal report: %w", err)
	}
	fmt.Println(string(report))
	return result.ExitCode(), nil
}

func fetch(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: status %d (asset missing or renamed)", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", url, err)
	}
	return body, nil
}
```

Note: the `-mode print` referenced in Task 3's optional check is not implemented; that check falls back to the package test, as stated there. `populate`/`verify` are the only modes.

- [ ] **Step 2: Verify it builds and vets**

Run: `GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go vet ./server/cmd/xoscal-spec-registry/`
Expected: exit 0 (every consumed `specregistry` symbol resolves).

- [ ] **Step 3: Commit**

```bash
git add server/cmd/xoscal-spec-registry/main.go
git commit -m "feat(specregistry): xoscal-spec-registry CLI (populate + verify)"
```

---

### Task 5: Wire the registry into recipe + lock

**Files:**
- Modify: `recipes/oscal.yaml` (add `spec_registry:` after `lock_path:`)
- Modify: `data/oscal/upstream.lock.yaml` (remove the `schema_sha256` line)

**Interfaces:**
- Produces: recipe declares the registry path; lock no longer carries the empty bundle hash (the registry is the single anchor).

- [ ] **Step 1: Add `spec_registry` to the recipe**

In `recipes/oscal.yaml`, immediately after the `lock_path: ...` line, add:

```yaml

# Per-model spec integrity registry (lock-step ledger). xoscal-spec-registry
# verifies each model's upstream schema hash against this file.
spec_registry: data/oscal/spec-registry.yaml
```

- [ ] **Step 2: Remove `schema_sha256` from the lock**

In `data/oscal/upstream.lock.yaml`, delete the line:

```yaml
    schema_sha256: ""        # sha256 of the upstream JSON schema bundle at that version
```

(The per-model `data/oscal/spec-registry.yaml` is now the schema-hash anchor.)

- [ ] **Step 3: Verify both still lint**

Run:
```bash
yamllint -d relaxed recipes/oscal.yaml data/oscal/upstream.lock.yaml && echo OK
GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go test ./server/internal/specregistry/ >/dev/null && echo "pkg ok"
```
Expected: `OK` (line-length warnings acceptable) and `pkg ok`.

- [ ] **Step 4: Commit**

```bash
git add recipes/oscal.yaml data/oscal/upstream.lock.yaml
git commit -m "feat(specregistry): point recipe at registry; drop empty lock schema hash"
```

---

### Task 6: Dagger gate + reconcile workflow verify step

**Files:**
- Modify: `dagger/main.go` (add `SpecRegistryCheck`, call it in `All`)
- Modify: `.github/workflows/oscal-reconcile.yml` (add a verify step)

**Interfaces:**
- Consumes: the `xoscal-spec-registry` CLI (Task 4).
- Produces: `func (m *Xoscal) SpecRegistryCheck(source *dagger.Directory) *dagger.Container` that builds + runs `verify` (network in-container) and emits `/tmp/specreg.ok`; included in `All`'s output directory. Workflow step runs `verify` after the poller.

- [ ] **Step 1: Add `SpecRegistryCheck` to `dagger/main.go`** (after `ProtoCheck`)

```go
// SpecRegistryCheck verifies the protos are lock-step with the pinned OSCAL
// spec: it re-fetches each model's upstream schema and compares to the committed
// registry. Needs network (fetches release assets), so it runs in-container.
func (m *Xoscal) SpecRegistryCheck(source *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithExec([]string{"go", "build", "-o", "/bin/spec-registry", "./server/cmd/xoscal-spec-registry"}).
		WithExec([]string{"/bin/spec-registry", "-mode", "verify", "-registry", "data/oscal/spec-registry.yaml"}).
		WithExec([]string{"sh", "-c", "echo 'specreg-ok' > /tmp/specreg.ok"})
}
```

- [ ] **Step 2: Wire it into `All`**

In `func (m *Xoscal) All`, add the branch and output file. Change the body to:

```go
	lint := m.Lint(source)
	test := m.Test(source)
	race := m.TestRace(source)
	sec := m.Security(source)
	proto := m.ProtoCheck(source)
	specreg := m.SpecRegistryCheck(source)

	return dag.Directory().
		WithFile("lint.ok", lint.File("/tmp/lint.ok")).
		WithFile("test.ok", test.File("/tmp/test.ok")).
		WithFile("race.ok", race.File("/tmp/race.ok")).
		WithFile("proto.ok", proto.File("/tmp/proto.ok")).
		WithFile("specreg.ok", specreg.File("/tmp/specreg.ok")).
		WithFile("gosec-results.sarif", sec)
```

- [ ] **Step 3: Add a verify step to the reconcile workflow**

In `.github/workflows/oscal-reconcile.yml`, after the `Poll feeds for drift` step (and before `Summarize`), add:

```yaml
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Verify OSCAL spec lock-step
        env:
          GH_TOKEN: ${{ github.token }}
          GITHUB_TOKEN: ${{ github.token }}
        run: |
          go run ./server/cmd/xoscal-spec-registry -mode verify \
            -registry data/oscal/spec-registry.yaml > spec-verify.json || code=$?
          cat spec-verify.json
          # exit 3 = per-model schema drift; surface it but let the existing
          # drift handling open the PR (do not hard-fail the scheduled run).
          if [ "${code:-0}" = "1" ]; then echo "spec verify operational error" >&2; exit 1; fi
```

- [ ] **Step 4: Verify**

Run:
```bash
cd dagger && GOCACHE=$TMPDIR/gc GOFLAGS=-mod=mod go vet ./... && gofmt -l main.go && cd ..
yamllint -d relaxed .github/workflows/oscal-reconcile.yml && echo OK
```
Expected: dagger vet exit 0, `gofmt -l` empty, `OK` from yamllint.

- [ ] **Step 5: Commit**

```bash
git add dagger/main.go .github/workflows/oscal-reconcile.yml
git commit -m "feat(specregistry): dagger SpecRegistryCheck gate + reconcile verify step"
```

---

## Self-Review

**Spec coverage:** registry file §4 → Task 3; CLI populate/verify §5 → Tasks 1,2,4; pure package §5 → Tasks 1,2; recipe `spec_registry` + lock drop §6 → Task 5; dagger gate + workflow verify §6 → Task 6; honesty boundary §7 → registry header (Task 3) + global constraints; exit-code contract → Task 2 + CLI. No gaps.

**Placeholder scan:** no TBD/TODO; all Go/YAML code is complete. The Task 3 optional `-mode print` check explicitly notes the CLI doesn't exist yet and falls back to the package test — not a dangling reference (the implemented modes are populate/verify only; Task 4's note repeats this).

**Type consistency:** `Registry`/`Model`/`ModelStatus`/`VerifyResult` and `Parse`/`Serialize`/`SchemaURL`/`Hash`/`CompareModel`/`Aggregate`/`ExitCode` are defined in Tasks 1–2 and consumed with identical names/signatures in Task 4. The registry YAML keys (`oscal_version`, `schema_asset`, `schema_sha256`, `proto`) match the struct yaml tags. CLI flags (`-mode`, `-registry`, `-version`) consistent across Tasks 4 and 6.

**Known runtime note (validate at execution, not a placeholder):** `populate`/`verify` and `SpecRegistryCheck` need network; they run in CI/engine, not the local sandbox. The exact `oscal_<model>_schema.json` asset names are confirmed when `populate` first runs against the real release — a wrong name fails loud (HTTP non-200) by design.
