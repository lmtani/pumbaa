package main

import (
	cli "github.com/lmtani/cromwell-cli/cli"
	"os"
)

var Version = "development"

func main() {
	os.Exit(cli.Run(Version))
}
