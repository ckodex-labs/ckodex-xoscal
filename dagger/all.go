package main

import (
	"dagger/xoscal/internal/dagger"
)

func (m *Xoscal) All(source *dagger.Directory) *dagger.Directory {
	fw := m.OscalFrameworks(source)
	lint := m.Lint(source)
	test := m.Test(source)
	race := m.TestRace(source)
	sec := m.Security(source)
	proto := m.ProtoCheck(source)
	specreg := m.SpecRegistryCheck(source)
	schemaval := m.OscalSchemaValidation(source, fw)
	constraints := m.OscalConstraintValidation(source, fw)

	return dag.Directory().
		WithFile("lint.ok", lint.File("/tmp/lint.ok")).
		WithFile("test.ok", test.File("/tmp/test.ok")).
		WithFile("race.ok", race.File("/tmp/race.ok")).
		WithFile("proto.ok", proto.File("/tmp/proto.ok")).
		WithFile("specreg.ok", specreg.File("/tmp/specreg.ok")).
		WithFile("schema.ok", schemaval.File("/tmp/schema.ok")).
		WithFile("constraints.ok", constraints.File("/tmp/constraints.ok")).
		WithFile("gosec-results.sarif", sec)
}
