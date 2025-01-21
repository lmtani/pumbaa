package cli

import (
	"os"

	"github.com/lmtani/pumbaa/internal/adapters/cromwellclient"
	"github.com/lmtani/pumbaa/internal/adapters/filesystem"
	"github.com/lmtani/pumbaa/internal/adapters/gcp"
	"github.com/lmtani/pumbaa/internal/adapters/http"
	"github.com/lmtani/pumbaa/internal/adapters/logger"
	"github.com/lmtani/pumbaa/internal/adapters/prompt"
	"github.com/lmtani/pumbaa/internal/adapters/writer"
	urfaveCli "github.com/urfave/cli/v2"
)

// NewCli creates a new CLI
func NewCli() *urfaveCli.App {
	cromwellClient := cromwellclient.NewCromwellClient("http://localhost:8000", nil)
	writer := writer.NewColoredWriter(os.Stdout)
	googleClient := gcp.NewGoogleCloud(&gcp.Wrapper{})
	l := logger.Logger{}
	fs := filesystem.NewLocalFilesystem(&l)
	h := http.NewDefaultHTTP()

	WorkflowHandler := NewWorkflowHandler(cromwellClient, writer)
	InteractiveHandler := NewInteractiveHandler(cromwellClient, writer, &prompt.Ui{})
	googleCloudHandler := NewGoogleCloudHandler(cromwellClient, writer)
	WDLHandler := NewWdlHandler(fs, h, googleClient)

	generalCategory := "General"
	cmds := []*urfaveCli.Command{
		{
			Name:     "query",
			Aliases:  []string{"q"},
			Usage:    "Query workflows",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: false, Value: "", Usage: "Filter by workflow name"},
				&urfaveCli.Int64Flag{Name: "days", Aliases: []string{"d"}, Required: false, Value: 7, Usage: "Show workflows from the last N days. Use 0 to show all workflows"},
			},
			Action: WorkflowHandler.Query,
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
			Action: WorkflowHandler.Submit,
		},
		{
			Name:     "kill",
			Aliases:  []string{"k"},
			Category: generalCategory,
			Usage:    "Kill a running job",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: WorkflowHandler.Kill,
		},
		{
			Name:     "metadata",
			Aliases:  []string{"m"},
			Category: generalCategory,
			Usage:    "Inspect workflow details (table)",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: WorkflowHandler.Metadata,
		},
		{
			Name:     "outputs",
			Aliases:  []string{"o"},
			Category: generalCategory,
			Usage:    "Query workflow outputs",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: WorkflowHandler.Outputs,
		},
		{
			Name:     "inputs",
			Aliases:  []string{"i"},
			Usage:    "Recover inputs from the specified workflow (JSON)",
			Category: generalCategory,
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: WorkflowHandler.Inputs,
		},
		{
			Name:     "navigate",
			Aliases:  []string{"n"},
			Category: generalCategory,
			Usage:    "Navigate through metadata data",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
			},
			Action: InteractiveHandler.Navigate,
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
			Action: WorkflowHandler.Wait,
		},
		{
			Name:    "local-deploy",
			Aliases: []string{"ld"},
			Usage:   "Install Cromwell Server locally with default configuration and start it",
			Flags: []urfaveCli.Flag{
				&urfaveCli.StringFlag{Name: "mysql-host", Required: false, Value: "127.0.0.1", Usage: "Your MySQL host"},
				&urfaveCli.StringFlag{Name: "mysql-passwd", Aliases: []string{"d"}, Required: false, Value: "1234", Usage: "Your MySQL password"},
				&urfaveCli.StringFlag{Name: "mysql-user", Required: false, Value: "root", Usage: "Your MySQL user"},
				&urfaveCli.Int64Flag{Name: "mysql-port", Required: false, Value: 3306, Usage: "Your MySQL port"},
				&urfaveCli.Int64Flag{Name: "port", Required: false, Value: 8000, Usage: "Port to bind Cromwell Server"},
				&urfaveCli.Int64Flag{Name: "max-jobs", Required: false, Value: 1, Usage: "Maximum number of jobs to run in parallel"},
				&urfaveCli.BoolFlag{Name: "override", Required: false, Usage: "Override the existing configuration file"},
			},
			Action: WDLHandler.Deploy,
		},
		{
			Name:    "gcp",
			Aliases: []string{"g"},
			Usage:   "Use commands specific for Google backend",
			Subcommands: []*urfaveCli.Command{
				{
					Name:  "resources",
					Usage: "View resource usage (cpu, mem or disk), normalized by hour.",
					Flags: []urfaveCli.Flag{
						&urfaveCli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true, Usage: "Operation ID"},
					},
					Action: googleCloudHandler.GetComputeUsageForPricing,
				},
			},
		},
	}
	return &urfaveCli.App{
		Name:     "Pumbaa",
		Usage:    "Command line interface for Cromwell Server",
		Commands: cmds,
	}
}
