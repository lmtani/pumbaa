package commands

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/output"
)

func (c *Commands) QueryWorkflow(name string) error {
	params := url.Values{}
	if name != "" {
		params.Add("name", name)
	}
	params.Add("includeSubworkflows", "false")
	resp, err := c.CromwellClient.Query(params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	output.NewTable(os.Stdout).Render(qtr)
	c.writer.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}

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
