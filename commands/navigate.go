package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func selectDesiredTask(m MetadataResponse) ([]CallItem, error) {
	taskOptions := []string{}
	for key := range m.Calls {
		taskName := strings.Split(key, ".")[1]
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
		return []CallItem{}, err
	}

	return m.Calls[fmt.Sprintf("%s.%s", m.WorkflowName, taskName)], nil
}

func selectDesiredShard(shards []CallItem) (CallItem, error) {
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
		return CallItem{}, err
	}

	return shards[i], err
}

func Navigate(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	var item CallItem
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
		resp, err = cromwellClient.Metadata(item.SubWorkflowID)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		color.Cyan(item.CallCaching.Result)
	} else {
		color.Cyan(item.CommandLine)
	}

	fmt.Printf("Logs:\n")
	color.Cyan("%s\n%s\n", item.Stderr, item.Stdout)
	if item.MonitoringLog != "" {
		color.Cyan("%s\n", item.MonitoringLog)
	}

	fmt.Printf("🐋 Docker image:\n")
	color.Cyan("%s\n", item.RuntimeAttributes.Docker)
	return nil
}
