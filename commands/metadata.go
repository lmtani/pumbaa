package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
)

type MetadataResponse struct {
	WorkflowName string
	Calls        map[string][]CallItem
	Inputs       map[string]interface{}
	Outputs      map[string]interface{}
	Start        time.Time
	End          time.Time
}

type CallItem struct {
	ExecutionStatus   string
	Stdout            string
	Stderr            string
	Attempt           int
	Start             time.Time
	End               time.Time
	Labels            Label
	MonitoringLog     string
	CommandLine       string
	DockerImageUsed   string
	RuntimeAttributes RuntimeAttributes
}

type RuntimeAttributes struct {
	BootDiskSizeGb string
	CPU            string
	Disks          string
	Docker         string
	Memory         string
}

type Label struct {
	CromwellWorkflowID string `json:"cromwell-workflow-id"`
	WdlTaskName        string `json:"wdl-task-name"`
}

type MetadataTableResponse struct {
	WorkflowName string
	Calls        map[string][]CallItem
	Inputs       map[string]interface{}
	Outputs      map[string]interface{}
	Start        time.Time
	End          time.Time
}

func (mtr MetadataTableResponse) Header() []string {
	return []string{"task", "attempt", "elapsed", "status"}
}

func (mtr MetadataTableResponse) Rows() [][]string {
	rows := make([][]string, len(mtr.Calls))
	for call, elements := range mtr.Calls {
		substrings := strings.Split(call, ".")
		for _, elem := range elements {
			if elem.End.IsZero() {
				elem.End = time.Now()
			}
			elapsedTime := elem.End.Sub(elem.Start)
			row := []string{substrings[len(substrings)-1], fmt.Sprintf("%d", elem.Attempt), elapsedTime.String(), elem.ExecutionStatus}
			rows = append(rows, row)
		}
	}
	return rows
}

func MetadataWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse(resp)
	output.NewTable(os.Stdout).Render(mtr)
	return err
}
