// Package main provides the entry point for the pumbaa CLI.
package main

import (
	"fmt"
	"os"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/container"
	"github.com/urfave/cli/v2"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "none"
	// Date is set at build time
	Date = "unknown"
)

func main() {
	// Initialize configuration
	cfg := config.Load()

	app := &cli.App{
		Name:    "pumbaa",
		Usage:   "A CLI tool for interacting with Cromwell workflow engine and WDL files",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Aliases: []string{"H"},
				Usage:   "Cromwell server host URL",
				EnvVars: []string{"CROMWELL_HOST"},
				Value:   "http://localhost:8000",
			},
		},
		Before: func(c *cli.Context) error {
			// Update config with CLI flags
			if c.IsSet("host") {
				cfg.CromwellHost = c.String("host")
			}
			return nil
		},
	}

	// Create container with initial config
	cont := container.New(cfg)

	// Setup commands
	app.Commands = []*cli.Command{
		{
			Name:    "workflow",
			Aliases: []string{"wf"},
			Usage:   "Workflow operations",
			Subcommands: []*cli.Command{
				cont.SubmitHandler.Command(),
				cont.MetadataHandler.Command(),
				cont.AbortHandler.Command(),
				cont.QueryHandler.Command(),
				cont.DebugHandler.Command(),
			},
		},
		cont.BundleHandler.Command(),
		cont.DashboardHandler.Command(),
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
