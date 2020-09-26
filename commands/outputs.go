package commands

import (
	"fmt"
	"os"

	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
)

type OutputsResponse struct {
	ID      string
	Outputs map[string]interface{}
}

type OutputsTableResponse struct {
	Outputs map[string]interface{}
}

func (otr OutputsTableResponse) Header() []string {
	return []string{"Name", "Value"}
}

func (otr OutputsTableResponse) Rows() [][]string {
	rows := make([][]string, len(otr.Outputs))
	for k, v := range otr.Outputs {
		rows = append(rows, []string{k, fmt.Sprint(v)})
	}
	return rows
}

func OutputsWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Outputs(c.String("operation"))
	if err != nil {
		return err
	}
	var otr = OutputsTableResponse{
		Outputs: resp.Outputs,
	}
	output.NewTable(os.Stdout).Render(otr)
	return err
}
