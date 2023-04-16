package cli

import (
	"fmt"
	"log"
	"os"
	"time"

	cromwell2 "github.com/lmtani/cromwell-cli/cromwell"

	"github.com/lmtani/cromwell-cli/cmd"

	urfaveCli "github.com/urfave/cli/v2"
)

// Define global variables to be injected
var (
	cmdService *cmd.Commands
)

func setupApp(version string) *urfaveCli.App {
	// Start service
	cmdService = cmd.New()

	// Override the default cromwell server host
	beforeFunc := func(c *urfaveCli.Context) error {
		cmdService.CromwellClient.Host = c.String("host")
		cmdService.CromwellClient.Iap = c.String("iap")
		return nil
	}

	// Define global flags
	flags := []urfaveCli.Flag{
		&urfaveCli.StringFlag{
			Name:     "iap",
			Required: false,
			Usage:    "Uses your default Google Credentials to obtains an access token to this audience.",
		},
		&urfaveCli.StringFlag{
			Name:  "host",
			Value: "http://127.0.0.1:8000",
			Usage: "Url for your Cromwell Server",
		},
	}

	// Define the urfaveCli.App.Commands slice
	generalCategory := "General"
	googleCategory := "Google"
	setupCategory := "Setup"
	cmds := []*urfaveCli.Command{
		{
			Name:     "version",
			Aliases:  []string{"v"},
			Usage:    "Cromwell-CLI version",
			Category: generalCategory,
			Action: func(c *urfaveCli.Context) error {
				fmt.Printf("Version: %s\n", version)
				return nil
			},
		},
		{
			Name:     "query",
			Aliases:  []string{"q"},
			Usage:    "Query workflows",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: false, Value: "", Usage: "Filter by workflow name"},
				&urfaveCli.Int64Flag{Name: "days", Aliases: []string{"d"}, Required: false, Value: 7, Usage: "Show workflows from the last N days. Use 0 to show all workflows"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.QueryWorkflow(c.String("name"), time.Duration(c.Int64("days")))
			},
		},
		{
			Name:     "wait",
			Aliases:  []string{"w"},
			Category: generalCategory,
			Usage:    "Wait for operation until it is complete",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
				&urfaveCli.IntFlag{Name: "sleep", Aliases: []string{"s"}, Required: false, Value: 60, Usage: "Sleep time in seconds"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.Wait(c.String("operation"), c.Int("sleep"))
			},
		},
		{
			Name:     "submit",
			Aliases:  []string{"s"},
			Usage:    "Submit a workflow and its inputs to Cromwell",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true, Usage: "Path to the WDL file"},
				&urfaveCli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: true, Usage: "Path to the inputs JSOM file"},
				&urfaveCli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: false, Usage: "Path to the dependencies .zip file"},
				&urfaveCli.StringFlag{Name: "options", Aliases: []string{"o"}, Required: false, Usage: "Path to the options JSON file"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.SubmitWorkflow(c.String("wdl"), c.String("inputs"), c.String("dependencies"), c.String("options"))
			},
		},
		{
			Name:     "inputs",
			Aliases:  []string{"i"},
			Usage:    "Recover inputs from the specified workflow (JSON)",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.Inputs(c.String("operation"))
			},
		},
		{
			Name:     "kill",
			Aliases:  []string{"k"},
			Category: generalCategory,
			Usage:    "Kill a running job",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.KillWorkflow(c.String("operation"))
			},
		},
		{
			Name:     "metadata",
			Aliases:  []string{"m"},
			Category: generalCategory,
			Usage:    "Inspect workflow details (table)",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.MetadataWorkflow(c.String("operation"))
			},
		},
		{
			Name:     "outputs",
			Aliases:  []string{"o"},
			Category: generalCategory,
			Usage:    "Query workflow outputs (JSON)",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.OutputsWorkflow(c.String("operation"))
			},
		},
		{
			Name:     "navigate",
			Aliases:  []string{"n"},
			Category: generalCategory,
			Usage:    "Navigate through metadata data",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cmdService.Navigate(c.String("operation"))
			},
		},
		{
			Name:     "gcp",
			Aliases:  []string{"g"},
			Category: googleCategory,
			Usage:    "Use commands specific for Google backend",
			Subcommands: []*urfaveCli.Command{
				{
					Name:  "resources",
					Usage: "View resource usage (cpu, mem or disk), normalized by hour.",
					Flags: []urfaveCli.Flag{
						&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
					},
					Action: func(c *urfaveCli.Context) error {
						return cmdService.ResourcesUsed(c.String("operation"))
					},
				},
			},
		},
		{
			Name:     "local-deploy",
			Aliases:  []string{"ld"},
			Category: setupCategory,
			Usage:    "Install Cromwell Server locally with default configuration and start it",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "mysql-host", Required: false, Value: "127.0.0.1", Usage: "Your MySQL host"},
				&urfaveCli.StringFlag{Name: "mysql-passwd", Aliases: []string{"d"}, Required: false, Value: "1234", Usage: "Your MySQL password"},
				&urfaveCli.StringFlag{Name: "mysql-user", Required: false, Value: "root", Usage: "Your MySQL user"},
				&urfaveCli.Int64Flag{Name: "mysql-port", Required: false, Value: 3306, Usage: "Your MySQL port"},
				&urfaveCli.Int64Flag{Name: "port", Required: false, Value: 8000, Usage: "Port to bind Cromwell Server"},
				&urfaveCli.Int64Flag{Name: "max-jobs", Required: false, Value: 1, Usage: "Maximum number of jobs to run in parallel"},
				&urfaveCli.Int64Flag{Name: "replace-config", Required: false, Value: 1, Usage: "Maximum number of jobs to run in parallel"},
				&urfaveCli.BoolFlag{Name: "override", Required: false, Usage: "Override the existing configuration file"},
			},
			Action: func(c *urfaveCli.Context) error {
				config := cromwell2.ParseCliParams(c)
				return cromwell2.StartCromwellServer(config, c.Bool("override"))
			},
		},
		{
			Name:     "build",
			Aliases:  []string{"b"},
			Category: setupCategory,
			Usage:    "Edit import statements in WDLs and build a zip file with all dependencies",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "wdl", Required: true, Usage: "Main workflow"},
			},
			Action: func(c *urfaveCli.Context) error {
				return cromwell2.BuildWorkflowDist(c.String("wdl"))
			},
		},
	}

	return &urfaveCli.App{
		Name:     "cromwell-cli",
		Usage:    "Command line interface for Cromwell Server",
		Flags:    flags,
		Before:   beforeFunc,
		Commands: cmds,
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
