#!/usr/bin/env bash
#
# poll-feeds.sh — detect OSCAL/framework upstream drift.
#
# For each recipe in recipes/*.yaml: fetch its RSS/Atom feed, extract the
# latest version per the recipe's version_extract regex, and compare it
# against the value recorded at lock_path in data/oscal/upstream.lock.yaml.
#
# Emits a JSON drift report on stdout and exits:
#   0  all sources in sync
#   3  drift detected (at least one source moved)        <- triggers reconcile
#   1  operational error (missing dep, unreadable feed, bad recipe)
#
# Pure shell + curl + yq (mikefarah). No model, no network writes (Rule 5).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOCK="${ROOT}/data/oscal/upstream.lock.yaml"
RECIPES_DIR="${ROOT}/recipes"

err() { printf '%s\n' "$*" >&2; }

for dep in curl yq jq; do
  command -v "$dep" >/dev/null 2>&1 || { err "FATAL: missing dependency '$dep'"; exit 1; }
done
[[ -f "$LOCK" ]] || { err "FATAL: lock file not found: $LOCK"; exit 1; }

drift_found=0
results=()

for recipe in "$RECIPES_DIR"/*.yaml; do
  [[ -e "$recipe" ]] || { err "FATAL: no recipes in $RECIPES_DIR"; exit 1; }

  name=$(yq -r '.name' "$recipe")
  feed=$(yq -r '.feed' "$recipe")
  regex=$(yq -r '.version_extract' "$recipe")
  lock_path=$(yq -r '.lock_path' "$recipe")

  # Newest feed entry title (Atom: feed.entry[0].title; commit feeds use id).
  feed_xml=$(curl -fsSL "$feed") || { err "FATAL: cannot fetch feed for '$name': $feed"; exit 1; }
  raw=$(printf '%s' "$feed_xml" | yq -p=xml -r '.feed.entry[0].title // .feed.entry[0].id // ""')

  # Extract comparable version via the recipe's regex (first capture group).
  if [[ "$raw" =~ $regex ]]; then
    latest="${BASH_REMATCH[1]}"
  else
    err "FATAL: version_extract '$regex' did not match feed entry for '$name': '$raw'"
    exit 1
  fi

  current=$(yq -r ".${lock_path} // \"\"" "$LOCK")

  status="in_sync"
  if [[ "$latest" != "$current" ]]; then
    status="drift"
    drift_found=1
  fi

  results+=("$(jq -n \
    --arg n "$name" --arg c "$current" --arg l "$latest" \
    --arg s "$status" --arg f "$feed" \
    '{source:$n, current:$c, latest:$l, status:$s, feed:$f}')")
done

# Assemble the drift report (jq -s slurps the per-source objects into an array).
printf '%s\n' "${results[@]}" | jq -s \
  --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '{generated_at: $ts,
    drift: (map(select(.status=="drift")) | length > 0),
    sources: .}'

[[ "$drift_found" -eq 1 ]] && exit 3
exit 0
