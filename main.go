package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "cromwell-cli",
		Usage: "Command line interface for Cromwell Server",
		Commands: []*cli.Command{
			{
				Name:    "submit",
				Aliases: []string{"s"},
				Usage:   "Submit a new job for Cromwell Server",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "wdl", Aliases: []string{"w"}, Required: true},
					&cli.StringFlag{Name: "inputs", Aliases: []string{"i"}, Required: true},
					&cli.StringFlag{Name: "dependencies", Aliases: []string{"d"}, Required: true},
				},
				Action: func(c *cli.Context) error {
					fmt.Println("added task: ", c.Args().First())
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
					fmt.Println("completed task: ", c.Args().First())
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
