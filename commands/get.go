package commands

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/output"
)

func (c *Commands) QueryWorkflow(name string, days time.Duration) error {
	date := time.Now().Add(-time.Hour * 24 * days)
	params := url.Values{}
	if name != "" {
		params.Add("name", name)
	}
	if days != 0 {
		c.Writer.Accent(fmt.Sprintf("Querying workflows from %s until now", date.Format(time.Stamp)))
		params.Add("submission", date.Format("2006-01-02T00:00:00.000Z"))
	}

	params.Add("includeSubworkflows", "false")
	resp, err := c.CromwellClient.Query(params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	output.NewTable(os.Stdout).Render(qtr)
	c.Writer.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
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
			fmt.Sprintf("%.35s", r.Name),
			r.Start.Format(timePattern),
			elapsedTime.Round(time.Second).String(),
			r.Status,
		})
	}
	return rows
}
