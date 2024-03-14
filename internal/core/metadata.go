package core

import (
	"encoding/json"
	"fmt"
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
	"sort"
)

type MetadataTable struct {
	c ports.Cromwell
	w ports.Writer
}

func NewMetadata(c ports.Cromwell, w ports.Writer) *MetadataTable {
	return &MetadataTable{c: c, w: w}
}

func (m *MetadataTable) Metadata(o string) error {
	params := types.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	resp, err := m.c.Metadata(o, &params)
	if err != nil {
		return err
	}
	var mtr = types.MetadataTableResponse{Metadata: resp}
	m.w.Table(mtr)
	if len(resp.Failures) > 0 {
		m.w.Error(hasFailureMsg(resp.Failures))
		recursiveFailureParse(resp.Failures, m.w)
	}

	items, err := showCustomOptions(resp.SubmittedFiles)
	if err != nil {
		return err
	}

	if len(items) > 0 {
		m.w.Accent("🔧 Custom options")
	}
	// iterate over items strings
	for _, v := range items {
		m.w.Primary(v)
	}
	return err
}

func showCustomOptions(s types.SubmittedFiles) ([]string, error) {
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

func recursiveFailureParse(f []types.Failure, w ports.Writer) {
	for idx := range f {
		w.Primary(" - " + f[idx].Message)
		recursiveFailureParse(f[idx].CausedBy, w)
	}
}

func hasFailureMsg(fails []types.Failure) string {
	m := "issue"
	if len(fails) > 1 {
		m = "issues"
	}
	msg := fmt.Sprintf("❗You have %d %s:\n", len(fails), m)
	return msg
}