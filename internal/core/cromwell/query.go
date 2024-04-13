package cromwell

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Query struct {
	c ports.Cromwell
	w ports.Writer
}

func NewQuery(c ports.Cromwell, w ports.Writer) *Query {
	return &Query{c: c, w: w}
}

func (q *Query) QueryWorkflow(name string, days time.Duration) error {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := types.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	resp, err := q.c.Query(&params)
	if err != nil {
		return err
	}
	var qtr = types.QueryTableResponse(resp)

	q.w.Table(qtr)
	q.w.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}
