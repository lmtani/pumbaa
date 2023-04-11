package app

import (
	"log"
	"os"

	"github.com/lmtani/cromwell-cli/internal/commands"
	"github.com/lmtani/cromwell-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

func setupApp(version string) *cli.App {
	cmds := commands.New()
	ui := prompt.NewTermUi()

	// Define the Before function
	beforeFunc := func(c *cli.Context) error {
		cmds.CromwellClient.Host = c.String("host")
		cmds.CromwellClient.Iap = c.String("iap")
		return nil
	}

	// Define the Commands slice
	commandsSlice := []*cli.Command{
		getVersion(version),
		query(cmds),
		wait(cmds),
		submit(cmds),
		inputs(cmds),
		kill(cmds),
		metadata(cmds),
		outputs(cmds),
		gcp(cmds),
		navigate(ui),
		localDeploy(),
		packDependencies(),
	}

	// Define the Flags slice
	flagsSlice := []cli.Flag{
		&cli.StringFlag{
			Name:     "iap",
			Required: false,
			Usage:    "Uses your default Google Credentials to obtains an access token to this audience.",
		},
		&cli.StringFlag{
			Name:  "host",
			Value: "http://127.0.0.1:8000",
			Usage: "Url for your Cromwell Server",
		},
	}

	return &cli.App{
		Name:     "cromwell-cli",
		Usage:    "Command line interface for Cromwell Server",
		Flags:    flagsSlice,
		Before:   beforeFunc,
		Commands: commandsSlice,
	}
}

func Run(version string) int {
	app := setupApp(version)
	if err := app.Run(os.Args); err != nil {
		log.Printf("Runtime error: %v\n", err)
		return 1
	}
	return 0
}
