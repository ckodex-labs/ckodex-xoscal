// Command xoscal-ctl is the compliance operator CLI for xOSCAL.
package main

import (
	"os"

	"github.com/mchorfa/xoscal/server/internal/cli"
)

var version = "dev"

// Aliases for test compatibility.
var (
	runSeed      = cli.RunSeed
	runIngest    = cli.RunIngest
	runGenerate  = cli.RunGenerate
	runTabular   = cli.RunTabular
	resolveKind  = cli.ResolveKind
	kindFromName = cli.KindFromName
)

func main() {
	cli.Execute(os.Args, version)
}
