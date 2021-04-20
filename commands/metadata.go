package commands

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func MetadataWorkflow(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	params := url.Values{}
	resp, err := cromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse(resp)
	output.NewTable(os.Stdout).Render(mtr)
	return err
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
