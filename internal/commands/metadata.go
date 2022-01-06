package commands

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) MetadataWorkflow(operation string) error {
	params := cromwell.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	resp, err := c.CromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse{Metadata: resp}
	c.Writer.Table(mtr)
	if len(resp.Failures) > 0 {
		c.Writer.Error(hasFailureMsg(resp.Failures))
		recursiveFailureParse(resp.Failures, c.Writer)
	}

	showCustomOptions(resp.SubmittedFiles, c.Writer)
	return nil
}

func showCustomOptions(s cromwell.SubmittedFiles, w Writer) {
	var f cromwell.Options
	json.Unmarshal([]byte(s.Options), &f)
	if f != (cromwell.Options{}) {
		w.Accent("🔧 Custom options")
		v := reflect.ValueOf(f)
		typeOfS := v.Type()
		for i := 0; i < v.NumField(); i++ {
			item := v.Field(i).Interface()
			if item != "" {
				w.Primary(fmt.Sprintf("- %s: %s", typeOfS.Field(i).Name, v.Field(i).Interface()))
			}
		}

	}
}

func hasFailureMsg(fails []cromwell.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}

func recursiveFailureParse(f []cromwell.Failure, w Writer) {
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
