package commands

import (
	"fmt"
	"time"

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

func queryResponseToTable(workflows QueryResponse) ([]string, [][]string) {
	header := []string{"Operation", "Name", "Start", "End", "Status"}
	rows := [][]string{}
	timePattern := "2006-01-02 15h04m"
	for _, elem := range workflows.Results {
		rows = append(rows, []string{elem.ID, elem.Name, elem.Start.Format(timePattern), elem.End.Format(timePattern), elem.Status})
	}
	return header, rows
}

func QueryWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Query(c.String("name"))
	if err != nil {
		return err
	}
	zap.S().Debugw(fmt.Sprintf("Found %d workflows", resp.TotalResultsCount))
	header, rows := queryResponseToTable(resp)
	CreateTable(header, rows)
	return err
}
