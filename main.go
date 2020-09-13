package main

import (
	"context"
	"os"

	"github.com/lmtani/cromwell-cli/commands"
	"github.com/mitchellh/mapstructure"
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

func toClient(i interface{}) commands.Client {
	c := commands.Client{}
	mapstructure.Decode(i, &c)
	return c
}

type contextKey string

func main() {
	const keyCromwell contextKey = "cromwellCli"
	logger, _ := startLogger()

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
				Usage:   "Query a workflow by its name",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: false},
				},
				Action: func(c *cli.Context) error {
					cromwellClient := toClient(c.Context.Value(keyCromwell))
					err := commands.QueryWorkflow(cromwellClient, c.String("name"))
					if err != nil {
						return err
					}
					return nil
				},
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
				Action: func(c *cli.Context) error {
					cromwellClient := toClient(c.Context.Value(keyCromwell))
					err := commands.SubmitWorkflow(cromwellClient, c.String("wdl"), c.String("inputs"), c.String("dependencies"))
					if err != nil {
						return err
					}
					return nil
				},
			},

			{
				Name:    "kill",
				Aliases: []string{"k"},
				Usage:   "Kill a running job",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					cromwellClient := toClient(c.Context.Value(keyCromwell))
					err := commands.KillWorkflow(cromwellClient, c.String("operation"))
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:    "metadata",
				Aliases: []string{"m"},
				Usage:   "Inspect workflow details",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "operation", Aliases: []string{"o"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					cromwellClient := toClient(c.Context.Value(keyCromwell))
					err := commands.MetadataWorkflow(cromwellClient, c.String("operation"))
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Error(err.Error())
	}
}
