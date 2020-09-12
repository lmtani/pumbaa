package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
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

func createQueryTable(workflows QueryResponse) {
	rows := [][]string{}
	timePattern := "2006-01-02 15h04m"
	for _, elem := range workflows.Results {
		rows = append(rows, []string{elem.ID, elem.Name, elem.Start.Format(timePattern), elem.End.Format(timePattern), elem.Status})
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Operation", "Name", "Start", "End", "Status"})
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.AppendBulk(rows)
	table.Render()
}

func QueryWorkflow(c Client, n string) error {
	resp, err := c.Query(n)
	if err != nil {
		return err
	}
	zap.S().Infow(fmt.Sprintf("Found %d workflows", resp.TotalResultsCount))
	createQueryTable(resp)
	return err
}
