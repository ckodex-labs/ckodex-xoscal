package main

import (
	"dagger/xoscal/internal/dagger"
)

// Badges pre-renders or packages offline SVG badges powered by the shieldcn-zig / shieldcn engine.
// These static badge assets are vendored for air-gapped distribution and documentation portals,
// strictly honoring Principle 40 (Air-gap is a design property; no runtime network dependency).
func (m *Xoscal) Badges(source *dagger.Directory) *dagger.Directory {
	return m.base(source).
		WithExec([]string{"mkdir", "-p", "/out/badges"}).
		WithExec([]string{"sh", "-c", `
set -e
BASE="https://shieldcn.dev"
badges="
release:github/release/ckodex-labs/ckodex-xoscal.svg?variant=secondary
ci:github/ci/ckodex-labs/ckodex-xoscal.svg?workflow=ci.yml&branch=main&variant=secondary
license:github/license/ckodex-labs/ckodex-xoscal.svg?variant=secondary
oscal:badge/OSCAL-1.2.3%20Complete-18181b.svg?variant=secondary
go:badge/Go-1.24+-18181b.svg?logo=go&variant=secondary
dagger:badge/Dagger-CI%2FCD-18181b.svg?logo=dagger&variant=secondary
slsa:badge/SLSA-Level%203-18181b.svg?variant=secondary
airgap:badge/airgap-ready-18181b.svg?variant=secondary
shieldcn:badge/badges%20by-shieldcn--zig-18181b.svg?logo=zig&variant=secondary
"
# If local assets already exist in source site/assets/badges, prefer them (offline air-gap)
if [ -d "/src/site/assets/badges" ]; then
    cp -a /src/site/assets/badges/*.svg /out/badges/ 2>/dev/null || true
fi

# Ensure all standard badges are populated
for entry in $badges; do
    name="${entry%%:*}"
    path="${entry#*:}"
    out="/out/badges/${name}.svg"
    if [ ! -s "$out" ]; then
        echo "Fetching badge $name from $path"
        curl -fsSL "$BASE/$path" -o "$out" || {
            echo "Fallback offline SVG for $name"
            printf '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="24"><rect width="120" height="24" rx="4" fill="#18181b"/><text x="10" y="16" fill="#fff" font-family="monospace" font-size="11">%s</text></svg>' "$name" > "$out"
        }
    fi
done
`}).
		Directory("/out/badges")
}
