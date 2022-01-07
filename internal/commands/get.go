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
