// Package main provides the entry point for the pumbaa CLI.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/container"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
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

	// Create container with initial config
	cont := container.New(cfg, Version)
	defer cont.TelemetryService.Close()

	// Log app start for telemetry breadcrumb trail
	cont.TelemetryService.AddBreadcrumb("app", fmt.Sprintf("pumbaa %s started", Version))

	var startTime time.Time

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
				cont.CromwellClient.BaseURL = c.String("host")
			}

			// Log command execution breadcrumb
			if len(c.Args().Slice()) > 0 || c.Command != nil {
				cmdName := "pumbaa"
				if c.Command != nil {
					cmdName = c.Command.FullName()
				}
				cont.TelemetryService.AddBreadcrumb("navigation", fmt.Sprintf("executing: %s", cmdName))
			}

			// Store start time for telemetry
			startTime = time.Now()
			return nil
		},
	}

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
		cont.ChatHandler.Command(),
		cont.ConfigHandler.Command(),
	}

	// Run the app and track telemetry at the end
	err := app.Run(os.Args)

	// Track command execution (only if a command was invoked)
	if len(os.Args) > 1 && !startTime.IsZero() {
		cont.TelemetryService.TrackCommand(telemetry.CommandContext{
			AppName:   app.Name,
			Args:      os.Args[1:],
			StartTime: startTime,
		}, err)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
