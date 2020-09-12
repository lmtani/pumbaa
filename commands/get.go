package commands

import (
	"fmt"
	"os"

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
	Start                 string
	End                   string
	MetadataArchiveStatus string
}

func createQueryTable(workflows QueryResponse) {
	rows := [][]string{}
	for _, elem := range workflows.Results {
		rows = append(rows, []string{elem.ID, elem.Name, elem.Status})
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Operation", "Name", "Status"})
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.AppendBulk(rows) // Add Bulk Data
	table.Render()
}

func QueryWorkflow(c Client, n string) error {
	resp, err := c.Query("VsaCloud")
	if err != nil {
		return err
	}
	zap.S().Infow(fmt.Sprintf("Found %d workflows", resp.TotalResultsCount))
	createQueryTable(resp)
	return err
}
