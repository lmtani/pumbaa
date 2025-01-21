// package main

// import (
// 	"log"
// 	"os"

// 	urfaveCli "github.com/urfave/cli/v2"
// )

// var (
// 	version = "dev"
// 	commit  = "none"
// 	date    = "unknown"
// )

// type Build struct {
// 	Version string
// 	Commit  string
// 	Date    string
// }

// func setupApp(b *Build) *urfaveCli.App {
// 	// Define global flags
// 	flags := []urfaveCli.Flag{
// 		&urfaveCli.StringFlag{
// 			Name:     "iap",
// 			Required: false,
// 			Usage:    "Uses your default Google Credentials to obtains an access token to this audience.",
// 		},
// 		&urfaveCli.StringFlag{
// 			Name:  "host",
// 			Value: "http://127.0.0.1:8000",
// 			Usage: "Url for your Cromwell Server",
// 		},
// 	}

// 	// Define the urfaveCli.App.Commands slice
// 	generalCategory := "General"
// 	googleCategory := "Google"
// 	setupCategory := "Setup"
// 	cmds := []*urfaveCli.Command{
// 		{
// 			Name:     "version",
// 			Aliases:  []string{"v"},
// 			Usage:    "pumbaa version",
// 			Category: generalCategory,
// 			Action: func(c *urfaveCli.Context) error {
// 				return getVersion(b)
// 			},
// 		},
// 		{
// 			Name:     "build",
// 			Aliases:  []string{"b"},
// 			Category: setupCategory,
// 			Usage:    "Edit import statements in WDLs and build a zip file with all dependencies",
// 			Flags: []urfaveCli.Flag{
// 				&urfaveCli.StringFlag{Name: "wdl", Required: true, Usage: "Main workflow"},
// 				&urfaveCli.StringFlag{Name: "out", Required: false, Value: "releases", Usage: "Output directory"},
// 			},
// 			Action: func(c *urfaveCli.Context) error {
// 				return packDependencies(c)
// 			},
// 		},
// 	}
// 	return &urfaveCli.App{
// 		Name:     "Pumbaa",
// 		Usage:    "Command line interface for Cromwell Server",
// 		Flags:    flags,
// 		Commands: cmds,
// 	}
// }

// func Run(b *Build) int {
// 	app := setupApp(b)
// 	if err := app.Run(os.Args); err != nil {
// 		log.Printf("Runtime error: %v\n", err)
// 		return 1
// 	}
// 	return 0
// }

// func main() {
// 	buildInfo := Build{
// 		Version: version,
// 		Commit:  commit,
// 		Date:    date,
// 	}
// 	os.Exit(Run(&buildInfo))
// }

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
