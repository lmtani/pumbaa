package main

import (
	"os"

	cli "github.com/lmtani/cromwell-cli/cli"
)

var Version = "development"

func main() {
	os.Exit(cli.Run(Version))
}
