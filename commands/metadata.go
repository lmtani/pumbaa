package commands

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func (c *Commands) MetadataWorkflow(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	params := url.Values{}
	params.Add("excludeKey", "executionEvents")
	params.Add("excludeKey", "submittedFiles")
	params.Add("excludeKey", "jes")
	params.Add("excludeKey", "inputs")
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
	rows := [][]string{}
	for call, elements := range mtr.Calls {
		substrings := strings.Split(call, ".")
		for _, elem := range elements {
			if elem.ExecutionStatus == "" {
				continue
			}
			if elem.End.IsZero() {
				elem.End = time.Now()
			}
			elapsedTime := elem.End.Sub(elem.Start)
			row := []string{substrings[len(substrings)-1], fmt.Sprintf("%d", elem.Attempt), elapsedTime.String(), elem.ExecutionStatus}
			rows = append(rows, row)
		}
	}
	rs := rowSlice(rows)
	sort.Sort(rs)
	return rs
}

type rowSlice [][]string

func (c rowSlice) Len() int           { return len(c) }
func (c rowSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c rowSlice) Less(i, j int) bool { return strings.Compare(c[i][0], c[j][0]) == -1 }
