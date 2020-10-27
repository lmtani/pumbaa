package commands

import (
	"errors"
	"fmt"
	"strconv"

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
	for _, elements := range c {
		if !contains(taskOptions, elements[0].Labels.WdlTaskName) {
			taskOptions = append(taskOptions, elements[0].Labels.WdlTaskName)
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
	fmt.Printf("You choose %q\n", result)

	return result, nil
}

func selectDesiredShard(s []CallItem) (CallItem, error) {
	maxValue := len(s)
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

	prompt2 := promptui.Prompt{
		Label:    fmt.Sprintf("Select the desired shard number (max: %d)", maxValue-1),
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
	color.Cyan(item.CommandLine)
	fmt.Printf("🐖 Logs:\n")
	color.Cyan("%s\n%s\n%s\n", item.Stderr, item.Stdout, item.MonitoringLog)
	fmt.Printf("🐋 Docker image:\n")
	color.Cyan("%s\n", item.RuntimeAttributes.Docker)
	return nil
}
