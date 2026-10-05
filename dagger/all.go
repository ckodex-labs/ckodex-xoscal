package main

import (
	"dagger/xoscal/internal/dagger"
)

func (m *Xoscal) All(source *dagger.Directory) *dagger.Directory {
	fw := m.OscalFrameworks(source)
	tidy := m.ModuleTidyCheck(source)
	lint := m.Lint(source)
	test := m.Test(source)
	race := m.TestRace(source)
	sec := m.Security(source)
	proto := m.ProtoCheck(source)
	openapi := m.OpenapiCheck(source)
	specreg := m.SpecRegistryCheck(source)
	schemaval := m.OscalSchemaValidation(source, fw)
	constraints := m.OscalConstraintValidation(source, fw)
	badges := m.Badges(source)
	contracts := m.ReleaseContractCheck(source)

	return dag.Directory().
		WithFile("tidy.ok", tidy.File("/tmp/tidy.ok")).
		WithFile("release-contract.ok", contracts.File("/tmp/release-contract.ok")).
		WithFile("lint.ok", lint.File("/tmp/lint.ok")).
		WithFile("test.ok", test.File("/tmp/test.ok")).
		WithFile("race.ok", race.File("/tmp/race.ok")).
		WithFile("proto.ok", proto.File("/tmp/proto.ok")).
		WithFile("openapi.ok", openapi.File("/tmp/openapi.ok")).
		WithFile("specreg.ok", specreg.File("/tmp/specreg.ok")).
		WithFile("schema.ok", schemaval.File("/tmp/schema.ok")).
		WithFile("constraints.ok", constraints.File("/tmp/constraints.ok")).
		WithDirectory("constraints", constraints.Directory("/tmp/constraint-evidence")).
		WithFile("gosec-results.sarif", sec).
		WithDirectory("badges", badges)
}
