package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func (c *Commands) MetadataWorkflow(operation string) error {
	params := cromwell.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	resp, err := c.CromwellClient.Metadata(operation, &params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse{Metadata: resp}
	c.Writer.Table(mtr)
	if len(resp.Failures) > 0 {
		c.Writer.Error(hasFailureMsg(resp.Failures))
		recursiveFailureParse(resp.Failures, c.Writer)
	}

	err = c.showCustomOptions(resp.SubmittedFiles)
	return err
}

func (c *Commands) showCustomOptions(s cromwell.SubmittedFiles) error {
	var options map[string]interface{}
	err := json.Unmarshal([]byte(s.Options), &options)
	if err != nil {
		return err
	}

	keys := sortOptionsKeys(options)

	if len(keys) > 0 {
		c.writeOptions(keys, options)
	}

	return nil
}

func (c *Commands) writeOptions(keys []string, o map[string]interface{}) {
	c.Writer.Accent("🔧 Custom options")
	for _, v := range keys {
		if o[v] != "" {
			c.Writer.Primary(fmt.Sprintf("- %s: %v", v, o[v]))
		}
	}
}

func sortOptionsKeys(f map[string]interface{}) []string {
	keys := make([]string, 0)
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func hasFailureMsg(fails []cromwell.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}

func recursiveFailureParse(f []cromwell.Failure, w output.Writer) {
	for idx := range f {
		w.Primary(" - " + f[idx].Message)
		recursiveFailureParse(f[idx].CausedBy, w)
	}
}

type rowSlice [][]string

func (c rowSlice) Len() int           { return len(c) }
func (c rowSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c rowSlice) Less(i, j int) bool { return c[i][0] < c[j][0] }
