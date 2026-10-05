package main

import "dagger/xoscal/internal/dagger"

// Badges renders informational artwork from pinned inputs. Badge appearance
// cannot establish a CI verdict, signature or SLSA level.
func (m *Xoscal) Badges(source *dagger.Directory) *dagger.Directory {
	return sdkToolchain("python").WithFile("/render-badges.py", source.File("scripts/render-badges.py")).
		WithExec([]string{"python3", "/render-badges.py", "--output", "/badges", "--version", m.Version}).Directory("/badges")
}
