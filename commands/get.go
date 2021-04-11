package commands

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
)

func (qtr QueryTableResponse) Header() []string {
	return []string{"Operation", "Name", "Start", "Duration", "Status"}
}

func (qtr QueryTableResponse) Rows() [][]string {
	rows := make([][]string, len(qtr.Results))
	timePattern := "2006-01-02 15h04m"
	for _, r := range qtr.Results {
		if r.End.IsZero() {
			r.End = time.Now()
		}
		elapsedTime := r.End.Sub(r.Start)
		rows = append(rows, []string{
			r.ID,
			r.Name,
			r.Start.Format(timePattern),
			elapsedTime.Round(time.Second).String(),
			r.Status,
		})
	}
	return rows
}

func QueryWorkflow(c *cli.Context) error {
	params := url.Values{}
	if c.String("name") != "" {
		params.Add("name", c.String("name"))
	}
	params.Add("includeSubworkflows", "false")
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
	resp, err := cromwellClient.Query(params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	output.NewTable(os.Stdout).Render(qtr)
	color.Cyan(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}
