package commands

import (
	"errors"
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

func selectDesiredTask(c map[string][]CallItem) (string, error) {
	taskOptions := []string{}
	for key := range c {
		taskName := strings.Split(key, ".")[1]
		if !contains(taskOptions, taskName) {
			taskOptions = append(taskOptions, taskName)
		}
	}
	prompt := promptui.Select{
		Label: "Select a task",
		Items: taskOptions,
	}
	_, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", err
	}
	return result, nil
}

func selectDesiredShard(s []CallItem) (CallItem, error) {
	maxValue := len(s)
	if maxValue == 1 {
		return s[0], nil
	}
	validate := func(input string) error {
		v, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return errors.New("Invalid number")
		}
		if int(v) > maxValue {
			return errors.New("You do not have shard with this value")
		}
		return nil
	}
	label := "Select the desired shard number (max: %d)\n"
	for idx, e := range s {
		label += fmt.Sprintf("[%v] Attempt: %v, Status: %v\n", e.ShardIndex, e.Attempt, e.ExecutionStatus)
		if idx > 20 {
			label += "More than 20 shards, omitting remaining ones."
		}
	}
	// fmt.Println(label)
	prompt2 := promptui.Prompt{
		Label:    label,
		Validate: validate,
	}
	result, err := prompt2.Run()

	if err != nil {
		return CallItem{}, err
	}
	v, _ := strconv.Atoi(result)
	return s[v], nil
}

func Navigate(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	task, err := selectDesiredTask(resp.Calls)
	if err != nil {
		return err
	}
	selectedTask := resp.Calls[fmt.Sprintf("%s.%s", resp.WorkflowName, task)]
	item, err := selectDesiredShard(selectedTask)
	if err != nil {
		return err
	}

	fmt.Printf("🐖 Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		color.Cyan(item.CallCaching.Result)
	} else {
		color.Cyan(item.CommandLine)
	}

	fmt.Printf("🐖 Logs:\n")
	color.Cyan("%s\n%s\n", item.Stderr, item.Stdout)
	if item.MonitoringLog != "" {
		color.Cyan("%s\n", item.MonitoringLog)
	}

	fmt.Printf("🐋 Docker image:\n")
	color.Cyan("%s\n", item.RuntimeAttributes.Docker)
	return nil
}
