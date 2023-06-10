package operation

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func MetadataWorkflow(operation string, c *cromwell_client.Client, w Writer) error {
	params := cromwell_client.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	resp, err := c.Metadata(operation, &params)
	if err != nil {
		return err
	}
	var mtr = MetadataTableResponse{Metadata: resp}
	w.Table(mtr)
	if len(resp.Failures) > 0 {
		w.Error(hasFailureMsg(resp.Failures))
		recursiveFailureParse(resp.Failures, w)
	}

	items, err := showCustomOptions(resp.SubmittedFiles)
	if err != nil {
		return err
	}

	if len(items) > 0 {
		w.Accent("🔧 Custom options")
	}
	// iterate over items strings
	for _, v := range items {
		w.Primary(v)
	}
	return err
}

func showCustomOptions(s cromwell_client.SubmittedFiles) ([]string, error) {
	items := make([]string, 0)

	var options map[string]interface{}
	err := json.Unmarshal([]byte(s.Options), &options)
	if err != nil {
		return items, err
	}

	keys := sortOptionsKeys(options)

	if len(keys) > 0 {
		items = writeOptions(keys, options)
	}

	return items, nil
}

func writeOptions(keys []string, o map[string]interface{}) []string {
	items := make([]string, 0)
	for _, v := range keys {
		if o[v] != "" {
			items = append(items, fmt.Sprintf("- %s: %v", v, o[v]))
		}
	}
	return items
}

func sortOptionsKeys(f map[string]interface{}) []string {
	keys := make([]string, 0)
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func hasFailureMsg(fails []cromwell_client.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}

func recursiveFailureParse(f []cromwell_client.Failure, w Writer) {
	for idx := range f {
		w.Primary(" - " + f[idx].Message)
		recursiveFailureParse(f[idx].CausedBy, w)
	}
}

type rowSlice [][]string

func (c rowSlice) Len() int           { return len(c) }
func (c rowSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c rowSlice) Less(i, j int) bool { return c[i][0] < c[j][0] }
