package app

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

func query() *cli.Command {
	return &cli.Command{
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
	}
}

func wait() *cli.Command {
	return &cli.Command{
		Name:    "wait",
		Aliases: []string{"w"},
		Usage:   "Wait for operation until it is complete",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			&cli.IntFlag{Name: "sleep", Aliases: []string{"s"}, Required: false, Value: 60, Usage: "Sleep time in seconds"},
		},
		Action: func(c *cli.Context) error {
			return cmds.Wait(c.String("operation"), c.Int("sleep"))
		},
	}
}

func submit() *cli.Command {
	return &cli.Command{
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
	}
}

func inputs() *cli.Command {
	return &cli.Command{
		Name:    "inputs",
		Aliases: []string{"i"},
		Usage:   "Recover inputs from the specified workflow (JSON)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
		},
		Action: func(c *cli.Context) error {
			return cmds.Inputs(c.String("operation"))
		},
	}
}

func kill() *cli.Command {
	return &cli.Command{
		Name:    "kill",
		Aliases: []string{"k"},
		Usage:   "Kill a running job",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
		},
		Action: func(c *cli.Context) error {
			return cmds.KillWorkflow(c.String("operation"))
		},
	}
}

func metadata() *cli.Command {
	return &cli.Command{
		Name:    "metadata",
		Aliases: []string{"m"},
		Usage:   "Inspect workflow details (table)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
		},
		Action: func(c *cli.Context) error {
			return cmds.MetadataWorkflow(c.String("operation"))
		},
	}
}

func outputs() *cli.Command {
	return &cli.Command{
		Name:    "outputs",
		Aliases: []string{"o"},
		Usage:   "Query workflow outputs (JSON)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
		},
		Action: func(c *cli.Context) error {
			return cmds.OutputsWorkflow(c.String("operation"))
		},
	}
}

func navigate() *cli.Command {
	return &cli.Command{
		Name:    "navigate",
		Aliases: []string{"n"},
		Usage:   "Navigate through metadata data",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
		},
		Action: func(c *cli.Context) error {
			return cmds.Navigate(c.String("operation"))
		},
	}
}

func gcp() *cli.Command {
	return &cli.Command{
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
	}
}

func getVersion(version string) *cli.Command {
	return &cli.Command{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Cromwell-CLI version",
		Action: func(c *cli.Context) error {
			fmt.Printf("Version: %s\n", version)
			return nil
		},
	}
}
