package main

import "dagger/xoscal/internal/dagger"

// Openapi generates all four service specifications with the pinned Buf plugin,
// then validates their operation identities, routes and local references before
// returning deterministic JSON. Never select a schema-only file by glob order.
func (m *Xoscal) Openapi(source *dagger.Directory) *dagger.File {
	generated := m.Proto(source)
	return m.base(source).
		WithDirectory("/src/proto", generated).
		WithExec([]string{"go", "run", "./server/cmd/xoscal-openapi", "-output", "/tmp/openapi.json"}).
		File("/tmp/openapi.json")
}

// OpenapiCheck checks the checked-in contract against pinned generation and
// runs tampering, missing operation and route collision regression tests.
func (m *Xoscal) OpenapiCheck(source *dagger.Directory) *dagger.Container {
	generated := m.Proto(source)
	return m.base(source).
		WithDirectory("/src/proto", generated).
		WithExec([]string{"go", "run", "./server/cmd/xoscal-openapi", "-check"}).
		WithExec([]string{"go", "test", "./server/cmd/xoscal-openapi"}).
		WithExec([]string{"sh", "-c", "printf 'openapi-ok\\n' > /tmp/openapi.ok"})
}
