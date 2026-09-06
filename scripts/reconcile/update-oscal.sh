#!/usr/bin/env bash
#
# update-oscal.sh — verify and materialize an official OSCAL release.
#
# Safe automation boundary:
#   * release assets and GitHub-published SHA-256 digests are verified first;
#   * constraint/vocabulary/documentation-only schema changes are updated;
#   * protobuf-shape changes exit 4 and emit a review handoff report;
#   * no branch, commit, push, PR, or merge operation happens here.
#
# Exit codes: 0 = in sync/updated, 4 = proto-shape review required,
#             1 = operational or verification failure.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REGISTRY="${ROOT}/data/oscal/spec-registry.yaml"
LOCK="${ROOT}/data/oscal/upstream.lock.yaml"
VERSION_FILE="${ROOT}/server/internal/oscalversion/VERSION"
SCHEMA_ROOT="${ROOT}/server/internal/schemavalidate/schemas"
REPORT="${ROOT}/reconcile-report.json"
API_ROOT="https://api.github.com/repos/usnistgov/OSCAL"

requested_version="latest"
source_dir=""
check_only=0
allow_structural=0

usage() {
	cat <<'EOF'
Usage: update-oscal.sh [--version <semver|latest>] [--source-dir <release-dir>]
                       [--check] [--allow-structural]

--check              verify release/digests and report without writing
--allow-structural   operator-only override after manual proto reconciliation
--source-dir         use extracted release assets (directly or under json/schema)
EOF
}

while (($# > 0)); do
	case "$1" in
		--version)
			[[ $# -ge 2 ]] || { usage >&2; exit 1; }
			requested_version="$2"
			shift 2
			;;
		--source-dir)
			[[ $# -ge 2 ]] || { usage >&2; exit 1; }
			source_dir="$2"
			shift 2
			;;
		--check)
			check_only=1
			shift
			;;
		--allow-structural)
			allow_structural=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			printf 'FATAL: unknown argument: %s\n' "$1" >&2
			usage >&2
			exit 1
			;;
	esac
done

for dep in curl jq yq go buf git; do
	command -v "$dep" >/dev/null 2>&1 || {
		printf "FATAL: missing dependency '%s'\n" "$dep" >&2
		exit 1
	}
done

if command -v sha256sum >/dev/null 2>&1; then
	hash_file() { sha256sum "$1" | awk '{print $1}'; }
	verify_manifest() { sha256sum -c "$1"; }
elif command -v shasum >/dev/null 2>&1; then
	hash_file() { shasum -a 256 "$1" | awk '{print $1}'; }
	verify_manifest() { shasum -a 256 -c "$1"; }
else
	printf 'FATAL: sha256sum or shasum is required\n' >&2
	exit 1
fi

api_headers=(-H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28")
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
	api_headers+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

api_get() {
	curl -fsSL "${api_headers[@]}" "$1"
}

if [[ "$requested_version" == "latest" ]]; then
	release_json="$(api_get "${API_ROOT}/releases/latest")"
else
	[[ "$requested_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
		printf 'FATAL: invalid OSCAL version: %s\n' "$requested_version" >&2
		exit 1
	}
	release_json="$(api_get "${API_ROOT}/releases/tags/v${requested_version}")"
fi

tag="$(jq -er '.tag_name' <<<"$release_json")"
target_version="${tag#v}"
[[ "$target_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
	printf 'FATAL: release tag is not OSCAL semver: %s\n' "$tag" >&2
	exit 1
}
if [[ "$requested_version" != "latest" && "$target_version" != "$requested_version" ]]; then
	printf 'FATAL: requested %s but release API returned %s\n' "$requested_version" "$target_version" >&2
	exit 1
fi
if [[ "$(jq -r '.draft or .prerelease' <<<"$release_json")" == "true" ]]; then
	printf 'FATAL: refusing draft or prerelease OSCAL release %s\n' "$tag" >&2
	exit 1
fi

release_url="$(jq -er '.html_url' <<<"$release_json")"
published_at="$(jq -er '.published_at' <<<"$release_json")"

# Resolve annotated tags to the commit they name.
tag_ref="$(api_get "${API_ROOT}/git/ref/tags/${tag}")"
source_type="$(jq -er '.object.type' <<<"$tag_ref")"
source_commit="$(jq -er '.object.sha' <<<"$tag_ref")"
while [[ "$source_type" == "tag" ]]; do
	tag_object="$(api_get "${API_ROOT}/git/tags/${source_commit}")"
	source_type="$(jq -er '.object.type' <<<"$tag_object")"
	source_commit="$(jq -er '.object.sha' <<<"$tag_object")"
done
[[ "$source_type" == "commit" ]] || {
	printf 'FATAL: tag %s resolved to unsupported object type %s\n' "$tag" "$source_type" >&2
	exit 1
}

tmpdir="$(mktemp -d /tmp/ckodex-oscal-update.XXXXXX)"
trap 'rm -r -- "$tmpdir"' EXIT
mkdir -p "$tmpdir/schemas"

assets_file="$tmpdir/assets.ndjson"
: > "$assets_file"
manifest="$tmpdir/MANIFEST.sha256"
: > "$manifest"

asset_names="$(
	{
		yq -r '.models[].schema_asset' "$REGISTRY"
		printf '%s\n' oscal_complete_schema.json
	} | sort -u
)"

while IFS= read -r asset; do
	[[ -n "$asset" ]] || continue
	asset_json="$(jq -ec --arg asset "$asset" '.assets[] | select(.name == $asset)' <<<"$release_json")" || {
		printf 'FATAL: release %s is missing asset %s\n' "$tag" "$asset" >&2
		exit 1
	}
	digest="$(jq -er '.digest' <<<"$asset_json")"
	[[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || {
		printf 'FATAL: asset %s has no usable GitHub SHA-256 digest\n' "$asset" >&2
		exit 1
	}
	destination="$tmpdir/schemas/$asset"
	if [[ -n "$source_dir" ]]; then
		candidate="$source_dir/$asset"
		[[ -f "$candidate" ]] || candidate="$source_dir/json/schema/$asset"
		[[ -f "$candidate" ]] || {
			printf 'FATAL: source directory does not contain %s\n' "$asset" >&2
			exit 1
		}
		cp "$candidate" "$destination"
	else
		download_url="$(jq -er '.browser_download_url' <<<"$asset_json")"
		curl -fsSL "${api_headers[@]}" "$download_url" -o "$destination"
	fi
	actual="$(hash_file "$destination")"
	expected="${digest#sha256:}"
	[[ "$actual" == "$expected" ]] || {
		printf 'FATAL: digest mismatch for %s: got %s want %s\n' "$asset" "$actual" "$expected" >&2
		exit 1
	}
	printf '%s  %s\n' "$actual" "$asset" >> "$manifest"
	jq -nc --arg name "$asset" --arg digest "$digest" '{name:$name,digest:$digest}' >> "$assets_file"
done <<< "$asset_names"
sort -o "$manifest" "$manifest"

zip_name="oscal-${target_version}.zip"
release_zip_sha256="$(jq -er --arg asset "$zip_name" '.assets[] | select(.name == $asset) | .digest' <<<"$release_json")"
[[ "$release_zip_sha256" =~ ^sha256:[0-9a-f]{64}$ ]] || {
	printf 'FATAL: release ZIP %s has no usable GitHub SHA-256 digest\n' "$zip_name" >&2
	exit 1
}

current_version="$(tr -d '[:space:]' < "$VERSION_FILE")"
shape_delta="initial"
current_schema="${SCHEMA_ROOT}/v${current_version}/oscal_complete_schema.json"
if [[ -f "$current_schema" ]]; then
	normalize='walk(if type == "object" then del(."$id", .title, .description, ."$comment", .enum, .pattern, .format, .minLength, .maxLength, .minimum, .maximum, .exclusiveMinimum, .exclusiveMaximum, .multipleOf) else . end)'
	jq -S "$normalize" "$current_schema" > "$tmpdir/current-shape.json"
	jq -S "$normalize" "$tmpdir/schemas/oscal_complete_schema.json" > "$tmpdir/target-shape.json"
	if cmp -s "$tmpdir/current-shape.json" "$tmpdir/target-shape.json"; then
		shape_delta="constraint-only"
	else
		shape_delta="proto-review-required"
	fi
fi

jq -s \
	--arg current "$current_version" \
	--arg target "$target_version" \
	--arg tag "$tag" \
	--arg release_url "$release_url" \
	--arg published_at "$published_at" \
	--arg source_commit "$source_commit" \
	--arg release_zip_sha256 "$release_zip_sha256" \
	--arg shape_delta "$shape_delta" \
	'{source:"oscal",current:$current,target:$target,tag:$tag,release_url:$release_url,
	  published_at:$published_at,source_commit:$source_commit,
	  release_zip_sha256:$release_zip_sha256,shape_delta:$shape_delta,
	  assets:.,status:(if $current == $target then "in_sync"
	                   elif $shape_delta == "proto-review-required" then "review_required"
	                   else "verified_for_update" end)}' \
	"$assets_file" > "$tmpdir/reconcile-report.json"

if [[ "$check_only" -eq 1 ]]; then
	cat "$tmpdir/reconcile-report.json"
	[[ "$shape_delta" == "proto-review-required" ]] && exit 4
	exit 0
fi

if [[ "$current_version" == "$target_version" ]]; then
	cat "$tmpdir/reconcile-report.json"
	exit 0
fi

if [[ -n "$(git -C "$ROOT" status --porcelain --untracked-files=no)" && "${ALLOW_DIRTY:-0}" != "1" ]]; then
	printf 'FATAL: tracked worktree changes present; run the updater in a clean checkout\n' >&2
	exit 1
fi

if [[ "$shape_delta" == "proto-review-required" && "$allow_structural" -ne 1 ]]; then
	cp "$tmpdir/reconcile-report.json" "$REPORT"
	printf 'OSCAL %s requires protobuf-shape review; wrote %s\n' "$target_version" "$REPORT" >&2
	exit 4
fi

# Keep the small set of user-facing, current-version references synchronized.
# Historical evidence (locks, old schema directories, and release plans) is
# intentionally excluded from this mechanical replacement.
version_ref_files=(
	"AGENTS.md"
	"ORCHESTRATOR.md"
	"docs/BETA-CONTRACT.md"
	"docs/OSCAL-UPDATES.md"
	"portal-preview.html"
	"site/portal.html"
	"server/cmd/xoscal-validate-constraints/main_test.go"
	"server/internal/schemavalidate/validator_test.go"
)
current_pattern="${current_version//./\\.}"
for rel in "${version_ref_files[@]}"; do
	[[ -f "$ROOT/$rel" ]] || continue
	if sed --version >/dev/null 2>&1; then
		sed -i "s/${current_pattern}/${target_version}/g" "$ROOT/$rel"
	else
		sed -i '' "s/${current_pattern}/${target_version}/g" "$ROOT/$rel"
	fi
done

target_schema_dir="${SCHEMA_ROOT}/v${target_version}"
mkdir -p "$target_schema_dir"
while IFS= read -r asset; do
	[[ -n "$asset" ]] || continue
	install -m 0644 "$tmpdir/schemas/$asset" "$target_schema_dir/$asset"
done <<< "$asset_names"
install -m 0644 "$manifest" "$target_schema_dir/MANIFEST.sha256"

cat > "$target_schema_dir/SOURCE.md" <<EOF
# OSCAL ${target_version} schema provenance

- Release: \`${release_url}\`
- Published: \`${published_at}\`
- Tagged commit: \`${source_commit}\`
- Official release ZIP: \`${release_zip_sha256}\`

The files in this directory were verified against the per-asset SHA-256
digests returned by the GitHub Releases API. \`MANIFEST.sha256\` is the offline
verification anchor used by the update workflow.
EOF

printf '%s\n' "$target_version" > "$VERSION_FILE"
reconciled_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
TARGET_VERSION="$target_version" SOURCE_COMMIT="$source_commit" \
	RELEASE_ZIP_SHA256="$release_zip_sha256" RECONCILED_AT="$reconciled_at" \
	yq -i '
		.sources.oscal.oscal_metaschema_version = strenv(TARGET_VERSION) |
		.sources.oscal.source_commit = strenv(SOURCE_COMMIT) |
		.sources.oscal.release_zip_sha256 = strenv(RELEASE_ZIP_SHA256) |
		.sources.oscal.reconciled_at = strenv(RECONCILED_AT) |
		.sources.oscal.parity = "verified"
	' "$LOCK"

go run "$ROOT/server/cmd/xoscal-spec-registry" \
	-mode populate -registry "$REGISTRY" -version "$target_version"

(
	cd "$target_schema_dir"
	verify_manifest MANIFEST.sha256
)
(
	cd "$ROOT"
	buf lint
	buf breaking --against '.git#branch=main'
	go test ./server/internal/oscal/... ./server/internal/schemavalidate/... \
		./server/internal/oscalversion/... ./server/cmd/xoscal-validate-schema/... \
		./server/cmd/xoscal-validate-constraints/...
)

jq '.status = "updated"' "$tmpdir/reconcile-report.json" > "$REPORT"
cat "$REPORT"
