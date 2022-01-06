package app

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/cromwell-cli/internal/commands"
	"github.com/urfave/cli/v2"
)

var Version = "development"

func CLI(args []string) int {
	cmds := commands.New()

	app := &cli.App{
		Name:  "cromwell-cli",
		Usage: "Command line interface for Cromwell Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "iap",
				Required: false,
				Usage:    "Uses your defauld Google Credentials to obtains an access token to this audience.",
			},
			&cli.StringFlag{
				Name:  "host",
				Value: "http://127.0.0.1:8000",
				Usage: "Url for your Cromwell Server",
			},
		},
		Before: func(c *cli.Context) error {
			cmds.CromwellClient.Host = c.String("host")
			cmds.CromwellClient.Iap = c.String("iap")
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Cromwell-CLI version",
				Action: func(c *cli.Context) error {
					fmt.Printf("Version: %s\n", Version)
					return nil
				},
			},
			{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "Query workflows",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: false, Value: "", Usage: "Filter by workflow name"},
					&cli.Int64Flag{Name: "days", Aliases: []string{"d"}, Required: false, Value: 7, Usage: "Show workflows from the last N days. Use 0 to show all workflows"},
				},
				Action: func(c *cli.Context) error {
					return cmds.QueryWorkflow(c.String("name"), time.Duration(c.Int64("days")))
				},
			},
			{
				Name:    "wait",
				Aliases: []string{"w"},
				Usage:   "Wait for operation until it is complete",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
					&cli.IntFlag{Name: "sleep", Aliases: []string{"s"}, Required: false, Value: 60, Usage: "Sleep time in seconds"},
					&cli.BoolFlag{Name: "alarm", Aliases: []string{"a"}, Required: false, Usage: "Notify your OS when it is ready"},
				},
				Action: func(c *cli.Context) error {
					return cmds.Wait(c.String("operation"), c.Int("sleep"), c.Bool("alarm"))
				},
			},
			{
				Name:    "submit",
				Aliases: []string{"s"},
				Usage:   "Submit a workflow and its inputs to Cromwell",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true, Usage: "Path to the WDL file"},
					&cli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: true, Usage: "Path to the inputs JSOM file"},
					&cli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: false, Usage: "Path to the dependencies .zip file"},
					&cli.StringFlag{Name: "options", Aliases: []string{"o"}, Required: false, Usage: "Path to the options JSON file"},
				},
				Action: func(c *cli.Context) error {
					return cmds.SubmitWorkflow(c.String("wdl"), c.String("inputs"), c.String("dependencies"), c.String("options"))
				},
			},
			{
				Name:    "inputs",
				Aliases: []string{"i"},
				Usage:   "Recover inputs from the specified workflow (JSON)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				},
				Action: func(c *cli.Context) error {
					return cmds.Inputs(c.String("operation"))
				},
			},
			{
				Name:    "kill",
				Aliases: []string{"k"},
				Usage:   "Kill a running job",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				},
				Action: func(c *cli.Context) error {
					return cmds.KillWorkflow(c.String("operation"))
				},
			},
			{
				Name:    "metadata",
				Aliases: []string{"m"},
				Usage:   "Inspect workflow details (table)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				},
				Action: func(c *cli.Context) error {
					return cmds.MetadataWorkflow(c.String("operation"))
				},
			},
			{
				Name:    "outputs",
				Aliases: []string{"o"},
				Usage:   "Query workflow outputs (JSON)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				},
				Action: func(c *cli.Context) error {
					return cmds.OutputsWorkflow(c.String("operation"))
				},
			},
			{
				Name:    "navigate",
				Aliases: []string{"n"},
				Usage:   "Navigate through metadata data",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				},
				Action: func(c *cli.Context) error {
					return cmds.Navigate(c.String("operation"))
				},
			},
			{
				Name:    "gcp",
				Aliases: []string{"g"},
				Usage:   "Use commands specific for Google backend",
				Subcommands: []*cli.Command{
					{
						Name:  "resources",
						Usage: "View resource usage (cpu, mem or disk), normalized by hour.",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
						},
						Action: func(c *cli.Context) error {
							return cmds.ResourcesUsed(c.String("operation"))
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0

}