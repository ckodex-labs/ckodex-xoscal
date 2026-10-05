package main

import (
	"dagger/xoscal/internal/dagger"
	"dagger/xoscal/modulecheck"
)

// ModuleTidyCheck admits the supplied root manifests only when tidy preserves
// both byte-for-byte. Its reference files never depend on Git working state.
func (m *Xoscal) ModuleTidyCheck(source *dagger.Directory) *dagger.Container {
	return m.base(source).
		WithFile("/tmp/tidy-input/go.mod", source.File("go.mod")).
		WithFile("/tmp/tidy-input/go.sum", source.File("go.sum")).
		WithExec([]string{"sh", "-c", modulecheck.Script, "module-tidy-check",
			"/tmp/tidy-input/go.mod", "/tmp/tidy-input/go.sum", "/tmp/tidy.ok"})
}
