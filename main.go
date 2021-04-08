package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lmtani/cromwell-cli/commands"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Version = "development"

func startLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.Level.SetLevel(zap.InfoLevel)
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	zap.ReplaceGlobals(logger)
	return logger, nil
}

func main() {
	keyCromwell := "cromwell"
	logger, err := startLogger()
	if err != nil {
		log.Fatalf("could not initialize custom logger; got %v", err)
	}

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
			cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
			c.Context = context.WithValue(c.Context, keyCromwell, cromwellClient)
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
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: false},
				},
				Action: commands.QueryWorkflow,
			},
			{
				Name:    "wait",
				Aliases: []string{"w"},
				Usage:   "Wait for operation until it is complete",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
					&cli.IntFlag{Name: "sleep", Aliases: []string{"s"}, Required: false, Value: 60},
					&cli.BoolFlag{Name: "alarm", Aliases: []string{"a"}, Required: false},
				},
				Action: commands.Wait,
			},
			{
				Name:    "submit",
				Aliases: []string{"s"},
				Usage:   "Submit a workflow and its inputs to Cromwell",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true},
					&cli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: false},
					&cli.StringFlag{Name: "options", Aliases: []string{"o"}, Required: false},
				},
				Action: commands.SubmitWorkflow,
			},
			{
				Name:    "inputs",
				Aliases: []string{"i"},
				Usage:   "Recover inputs from the specified workflow (JSON)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: commands.Inputs,
			},
			{
				Name:    "kill",
				Aliases: []string{"k"},
				Usage:   "Kill a running job",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: commands.KillWorkflow,
			},
			{
				Name:    "metadata",
				Aliases: []string{"m"},
				Usage:   "Inspect workflow details (table)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: commands.MetadataWorkflow,
			},
			{
				Name:    "outputs",
				Aliases: []string{"o"},
				Usage:   "Query workflow outputs (JSON)",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: commands.OutputsWorkflow,
			},
			{
				Name:    "navigate",
				Aliases: []string{"n"},
				Usage:   "Navigate through metadata data",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: commands.Navigate,
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
							&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
						},
						Action: commands.ResourcesUsed,
					},
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		logger.Error("cromwell.command.error",
			zap.NamedError("err", err))
	}
}
