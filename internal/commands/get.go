package commands

import (
	"fmt"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) QueryWorkflow(name string, days time.Duration) error {
	params := cromwell.ParamsQueryGet{
		Submission: time.Now().Add(-time.Hour * 24 * days),
		Name:       name,
	}
	resp, err := c.CromwellClient.Query(params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	c.Writer.Table(qtr)
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
