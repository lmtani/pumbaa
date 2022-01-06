package main

import (
	"os"

	app "github.com/lmtani/cromwell-cli/cli"
)

var Version = "development"

func main() {
	os.Exit(app.CLI(os.Args))
}
