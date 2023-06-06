package cmd

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func QueryWorkflow(name string, days time.Duration, c *cromwell_client.Client, w Writer) error {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := cromwell_client.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	resp, err := c.Query(&params)
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)

	w.Table(qtr)
	w.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}
