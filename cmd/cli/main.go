package main

import (
	"log"
	"os"

	urfaveCli "github.com/urfave/cli/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Build struct {
	Version string
	Commit  string
	Date    string
}

func setupApp(b *Build) *urfaveCli.App {
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
			Usage:    "pumbaa version",
			Category: generalCategory,
			Action: func(c *urfaveCli.Context) error {
				return getVersion(b)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.Query(c)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.wait(c)
			},
		},
		{
			Name:     "submit",
			Aliases:  []string{"s"},
			Usage:    "Submit a workflow and its inputs to Cromwell",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true, Usage: "Path to the WDL file"},
				&urfaveCli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: false, Usage: "Path to the inputs JSOM file"},
				&urfaveCli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: false, Usage: "Path to the dependencies .zip file"},
				&urfaveCli.StringFlag{Name: "options", Aliases: []string{"o"}, Required: false, Usage: "Path to the options JSON file"},
			},
			Action: func(c *urfaveCli.Context) error {
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.submit(c)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.inputs(c)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.kill(c)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.metadata(c)
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
				h := NewDefaultHandler(c.String("host"), c.String("iap"))
				return h.outputs(c)
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
				return navigate(c)
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
						h := NewDefaultHandler(c.String("host"), c.String("iap"))
						return h.gcpResources(c)
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
				&urfaveCli.BoolFlag{Name: "override", Required: false, Usage: "Override the existing configuration file"},
			},
			Action: func(c *urfaveCli.Context) error {
				return localDeploy(c)
			},
		},
		{
			Name:     "build",
			Aliases:  []string{"b"},
			Category: setupCategory,
			Usage:    "Edit import statements in WDLs and build a zip file with all dependencies",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "wdl", Required: true, Usage: "Main workflow"},
				&urfaveCli.StringFlag{Name: "out", Required: false, Value: "releases", Usage: "Output directory"},
			},
			Action: func(c *urfaveCli.Context) error {
				return packDependencies(c)
			},
		},
	}
	return &urfaveCli.App{
		Name:     "Pumbaa",
		Usage:    "Command line interface for Cromwell Server",
		Flags:    flags,
		Commands: cmds,
	}
}

func Run(b *Build) int {
	app := setupApp(b)
	if err := app.Run(os.Args); err != nil {
		log.Printf("Runtime error: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	buildInfo := Build{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	os.Exit(Run(&buildInfo))
}
