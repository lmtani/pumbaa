package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func generateTable() {
	data := [][]string{
		{"1/1/2014", "Domain name", "2233", "$10.98"},
		{"1/1/2014", "January Hosting", "2233", "$54.95"},
		{"1/4/2014", "February Hosting", "2233", "$51.00"},
		{"1/4/2014", "February Extra Bandwidth", "2233", "$30.00"},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Date", "Description", "CV2", "Amount"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}

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
					resp, err := http.Get("https://httpbin.org/get")
					if err != nil {
						log.Fatal(err)
					}
					defer resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatalln(err)
					}
					log.Println(string(body))
					log.Info("Work in progress")
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
					generateTable()
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
