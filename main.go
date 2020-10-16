package main

import (
	"context"
	"log"
	"os"

	"github.com/lmtani/cromwell-cli/commands"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func startLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
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
				Name:     "token",
				Aliases:  []string{"t"},
				Required: false,
				Usage:    "Bearer token to be included in HTTP requsts",
			},
			&cli.StringFlag{
				Name:  "host",
				Value: "http://127.0.0.1:8000",
				Usage: "Url for your Cromwell Server",
			},
		},
		Before: func(c *cli.Context) error {
			cromwellClient := commands.New(c.String("host"), c.String("token"))
			c.Context = context.WithValue(c.Context, keyCromwell, cromwellClient)
			return nil
		},
		Commands: []*cli.Command{
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
				Name:    "submit",
				Aliases: []string{"s"},
				Usage:   "Submit a workflow and its inputs to Cromwell",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true},
					&cli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: false},
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
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		logger.Error("cromwell.command.error",
			zap.NamedError("err", err))
	}
}
