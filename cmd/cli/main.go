package main

import (
	"log"
	"os"

	"github.com/lmtani/pumbaa/internal/infra/cli"
)

func main() {
	app := cli.NewCli()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
