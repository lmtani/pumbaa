package main

import (
	"fmt"
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
	logger, _ := startLogger()
	cromwellClient := commands.New("http://localhost:8000")
	app := &cli.App{
		Name:  "cromwell-cli",
		Usage: "Command line interface for Cromwell Server",
		Commands: []*cli.Command{
			{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "Query a workflow by its name",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					err := commands.QueryWorkflow(cromwellClient, c.String("name"))
					if err != nil {
						logger.Fatal(err.Error())
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
				},
				Action: func(c *cli.Context) error {
					logger.Info("Submitting workflow...")
					err := commands.SubmitWorkflow(cromwellClient, c.String("wdl"), c.String("inputs"))
					if err != nil {
						logger.Fatal(fmt.Sprintf("%s", err))
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
					err := commands.KillWorkflow(cromwellClient)
					if err != nil {
						logger.Fatal(err.Error())
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
