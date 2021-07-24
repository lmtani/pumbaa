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

func (c *Commands) MetadataWorkflow(operation string) error {
	params := url.Values{}
	params.Add("excludeKey", "executionEvents")
	params.Add("excludeKey", "submittedFiles")
	params.Add("excludeKey", "jes")
	params.Add("excludeKey", "inputs")
	resp, err := c.CromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse{Metadata: resp}
	output.NewTable(os.Stdout).Render(mtr)
	if len(resp.Failures) > 0 {
		c.writer.Error(hasFailureMsg(resp.Failures))
		recursiveFailureParse(resp.Failures, c.writer)
	}

	return nil
}

func hasFailureMsg(fails []cromwell.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}

func recursiveFailureParse(f []cromwell.Failure, w output.IWriter) {
	for idx := range f {
		w.Primary(" - " + f[idx].Message)
		recursiveFailureParse(f[idx].CausedBy, w)
	}
}

func (mtr MetadataTableResponse) Header() []string {
	return []string{"task", "attempt", "elapsed", "status"}
}

func (mtr MetadataTableResponse) Rows() [][]string {
	rows := [][]string{}
	for call, elements := range mtr.Metadata.Calls {
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
