// Package main provides the entry point for the pumbaa CLI.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
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

			// Store start time for telemetry
			c.App.Metadata["startTime"] = time.Now()
			return nil
		},
		After: func(c *cli.Context) error {
			// Don't track if no command command was run (e.g. root help)
			if c.Command == nil || c.Command.Name == "" {
				return nil
			}

			// Calculate duration
			startTime, ok := c.App.Metadata["startTime"].(time.Time)
			if !ok {
				startTime = time.Now()
			}
			duration := time.Since(startTime).Milliseconds()

			// Construct command name from args to ensure we capture subcommands
			// c.Command.FullName() sometimes returns just the app name in global After hook
			cmdName := c.Command.FullName()
			if cmdName == "" || cmdName == c.App.Name {
				// Fallback: try to reconstruct from args, excluding flags
				args := []string{c.App.Name}
				for _, arg := range os.Args[1:] {
					if !strings.HasPrefix(arg, "-") {
						args = append(args, arg)
					}
				}
				if len(args) > 1 {
					cmdName = strings.Join(args, " ")
				}
			}

			cont.TelemetryService.Track(telemetry.Event{
				Command:   cmdName,
				Duration:  duration,
				Success:   true, // We assume success here as we can't easily access error in After
				Version:   Version,
				OS:        runtime.GOOS,
				Arch:      runtime.GOARCH,
				Timestamp: time.Now().Unix(),
			})
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

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
