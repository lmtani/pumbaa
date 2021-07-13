package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/lmtani/cromwell-cli/internal/prompt"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) Navigate(p prompt.IPrompt, operation string) error {
	params := url.Values{}
	params.Add("excludeKey", "executionEvents")
	params.Add("excludeKey", "submittedFiles")
	params.Add("excludeKey", "jes")
	params.Add("excludeKey", "inputs")
	resp, err := c.CromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	var item cromwell.CallItem
	for {
		task, err := c.selectDesiredTask(p, resp)
		if err != nil {
			return err
		}
		item, err = c.selectDesiredShard(p, task)
		if err != nil {
			return err
		}
		if item.SubWorkflowID == "" {
			break
		}
		resp, err = c.CromwellClient.Metadata(item.SubWorkflowID, params)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		c.writer.Accent(item.CallCaching.Result)
	} else {
		c.writer.Accent(item.CommandLine)
	}

	fmt.Printf("Logs:\n")
	c.writer.Accent(fmt.Sprintf("%s\n%s\n", item.Stderr, item.Stdout))
	if item.MonitoringLog != "" {
		c.writer.Accent(fmt.Sprintf("%s\n", item.MonitoringLog))
	}
	if item.BackendLogs.Log != "" {
		c.writer.Accent(fmt.Sprintf("%s\n", item.BackendLogs.Log))
	}

	fmt.Printf("🐋 Docker image:\n")
	c.writer.Accent(fmt.Sprintf("%s\n", item.RuntimeAttributes.Docker))
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (c *Commands) selectDesiredTask(p prompt.IPrompt, m cromwell.MetadataResponse) ([]cromwell.CallItem, error) {
	taskOptions := []string{}
	calls := map[string][]cromwell.CallItem{}
	for key := range m.Calls {
		sliceName := strings.Split(key, ".")
		taskName := sliceName[len(sliceName)-1]
		calls[taskName] = m.Calls[key]
		if !contains(taskOptions, taskName) {
			taskOptions = append(taskOptions, taskName)
		}
	}
	cat := "Workflow"
	if m.RootWorkflowID != "" {
		cat = "SubWorkflow"
	}
	c.writer.Accent(fmt.Sprintf("%s: %s\n", cat, m.WorkflowName))

	taskName, err := p.SelectByKey(taskOptions)
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return []cromwell.CallItem{}, err
	}
	return calls[taskName], nil
}

func (c *Commands) selectDesiredShard(p prompt.IPrompt, shards []cromwell.CallItem) (cromwell.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}

	template := prompt.TemplateOptions{
		Label:    "{{ . }}?",
		Active:   "✔ {{ .ShardIndex  | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
		Inactive: "  {{ .ShardIndex | faint }} ({{ .ExecutionStatus | red }}) Attempt: {{ .Attempt | faint }} CallCaching: {{ .CallCaching.Hit | faint}}",
		Selected: "✔ {{ .ShardIndex | green }} ({{ .ExecutionStatus | green }}) Attempt: {{ .Attempt | green }} CallCaching: {{ .CallCaching.Hit | green}}",
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	i, err := p.SelectByIndex(template, searcher, shards)
	if err != nil {
		return cromwell.CallItem{}, err
	}

	return shards[i], err
}
