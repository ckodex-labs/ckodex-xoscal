// Command xoscal is the primary developer CLI for xOSCAL.
package main

import (
	"os"

	"github.com/mchorfa/xoscal/server/internal/cli"
)

var version = "dev"

func main() {
	cli.Execute(os.Args, version)
}
