package commands

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
)

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
	cromwellClient := cromwell.FromInterface(c.Context.Value("cromwell"))
	params := url.Values{}
	resp, err := cromwellClient.Metadata(c.String("operation"), params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse(resp)
	output.NewTable(os.Stdout).Render(mtr)
	return err
}
