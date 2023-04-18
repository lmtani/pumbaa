package main

import (
	"os"

	"github.com/lmtani/cromwell-cli/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	buildInfo := cli.Build{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	os.Exit(cli.Run(&buildInfo))
}
