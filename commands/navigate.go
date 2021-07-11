package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/manifoldco/promptui"
)

func (c *Commands) Navigate(operation string) error {
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
		task, err := selectDesiredTask(resp)
		if err != nil {
			return err
		}
		item, err = selectDesiredShard(task)
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

func selectDesiredTask(m cromwell.MetadataResponse) ([]cromwell.CallItem, error) {
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
	color.Green("%s: %s\n", cat, m.WorkflowName)
	prompt := promptui.Select{
		Label: "Select a task",
		Items: taskOptions,
	}
	_, taskName, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return []cromwell.CallItem{}, err
	}
	return calls[taskName], nil
}

func selectDesiredShard(shards []cromwell.CallItem) (cromwell.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "✔ {{ .ShardIndex  | green }} ({{ .ExecutionStatus | red }}) CallCaching: {{ .CallCaching.Hit}}",
		Inactive: "  {{ .ShardIndex | faint }} ({{ .ExecutionStatus | red }})",
		Selected: "✔ {{ .ShardIndex | green }}",
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	prompt := promptui.Select{
		Label:     "Which shard?",
		Items:     shards,
		Templates: templates,
		Size:      6,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()

	if err != nil {
		return cromwell.CallItem{}, err
	}

	return shards[i], err
}
