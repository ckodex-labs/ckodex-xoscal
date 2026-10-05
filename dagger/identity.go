package main

import "dagger/xoscal/internal/dagger"

// ReleaseIdentity rejects dirty/mismatched source and derives a tracked-only
// production tree with fresh credential-free Git metadata for all producers.
func (m *Xoscal) ReleaseIdentity(source *dagger.Directory, releaseTag, sourceRevision string) *dagger.Container {
	return m.verifyTools().WithEnvVariable("PYTHONDONTWRITEBYTECODE", "1").WithDirectory("/src", source).WithWorkdir("/src").
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends", "git"}).
		WithExec([]string{"python3", "scripts/release-source.py", releaseTag, sourceRevision, m.Version, "/tmp/release-source.json", "/tmp/release-source"})
}
