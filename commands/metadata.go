package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type MetadataResponse struct {
	WorkflowName string
	Calls        map[string][]CallItem
	Outputs      map[string]string
	Start        time.Time
	End          time.Time
}

type CallItem struct {
	ExecutionStatus string
	Stdout          string
	Stderr          string
	Attempt         int
	Start           time.Time
	End             time.Time
}

func prepareTableInput(resp MetadataResponse) ([]string, [][]string) {
	header := []string{"task", "attempt", "elapsed", "status"}
	rows := [][]string{}
	for call, elements := range resp.Calls {
		substrings := strings.Split(call, ".")
		for _, elem := range elements {
			elapsedTime := elem.End.Sub(elem.Start)
			row := []string{substrings[len(substrings)-1], fmt.Sprintf("%d", elem.Attempt), elapsedTime.String(), elem.ExecutionStatus}
			rows = append(rows, row)
		}
	}
	return header, rows
}

func MetadataWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	fmt.Println("\n======" + resp.WorkflowName + "======")
	header, rows := prepareTableInput(resp)
	CreateTable(header, rows)
	return err
}
