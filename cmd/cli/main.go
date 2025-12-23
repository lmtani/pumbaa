// Package main provides the entry point for the pumbaa CLI.
package main

import (
	"fmt"
	"os"
	"runtime"
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

			// Extract command name from args, matching only known commands/subcommands
			cmdName := extractCommandName(c.App.Name, os.Args[1:])

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

// extractCommandName extracts just the command/subcommand from args,
// ignoring flags and arguments.
func extractCommandName(appName string, args []string) string {
	// Known top-level commands and their subcommands
	knownCommands := map[string][]string{
		"workflow":  {"submit", "metadata", "abort", "query", "debug"},
		"wf":        {"submit", "metadata", "abort", "query", "debug"},
		"bundle":    {},
		"dashboard": {},
		"chat":      {},
		"config":    {},
	}

	if len(args) == 0 {
		return appName
	}

	// Check if first arg is a known command
	firstArg := args[0]
	subcommands, isKnown := knownCommands[firstArg]
	if !isKnown {
		// Not a known command, might be a flag like --help
		return appName
	}

	// Normalize aliases
	cmdName := firstArg
	if cmdName == "wf" {
		cmdName = "workflow"
	}

	// Check for subcommand
	if len(args) > 1 && len(subcommands) > 0 {
		secondArg := args[1]
		for _, sub := range subcommands {
			if secondArg == sub {
				return appName + " " + cmdName + " " + sub
			}
		}
	}

	return appName + " " + cmdName
}
