package commands

import (
	"fmt"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) QueryWorkflow(name string, days time.Duration) error {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := cromwell.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	resp, err := c.CromwellClient.Query(&params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	c.Writer.Table(qtr)
	c.Writer.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}
