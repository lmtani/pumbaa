package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

type QueryResponse struct {
	Results           []QueryResponseWorkflow
	TotalResultsCount int
}

type QueryResponseWorkflow struct {
	ID                    string
	Name                  string
	Status                string
	Submission            string
	Start                 time.Time
	End                   time.Time
	MetadataArchiveStatus string
}

type QueryTableResponse struct {
	Results           []QueryResponseWorkflow
	TotalResultsCount int
}

func (qtr QueryTableResponse) Header() []string {
	return []string{"Operation", "Name", "Start", "End", "Status"}
}

func (qtr QueryTableResponse) Rows() [][]string {
	rows := make([][]string, len(qtr.Results))
	timePattern := "2006-01-02 15h04m"
	for _, r := range qtr.Results {
		rows = append(rows, []string{
			r.ID,
			r.Name,
			r.Start.Format(timePattern),
			r.End.Format(timePattern),
			r.Status,
		})
	}
	return rows
}

func QueryWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Query(c.String("name"))
	if err != nil {
		return err
	}
	var qtr = QueryTableResponse(resp)
	output.NewTable(os.Stdout).Render(qtr)
	zap.S().Info(fmt.Sprintf("Found %d workflows", resp.TotalResultsCount))
	return err
}
