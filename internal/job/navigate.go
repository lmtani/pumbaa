package job

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func Navigate(operation string, c *cromwell_client.Client, w Writer, p Prompt) error {
	params := cromwell_client.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}
	resp, err := c.Metadata(operation, &params)
	if err != nil {
		return err
	}
	var item cromwell_client.CallItem
	for {
		task, err := selectDesiredTask(&resp, p, w)
		if err != nil {
			return err
		}
		item, err = selectDesiredShard(task, p)
		if err != nil {
			return err
		}
		if item.SubWorkflowID == "" {
			break
		}
		resp, err = c.Metadata(item.SubWorkflowID, &params)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		w.Accent(item.CallCaching.Result)
	} else {
		w.Accent(item.CommandLine)
	}

	fmt.Printf("Logs:\n")
	w.Accent(fmt.Sprintf("%s\n%s\n", item.Stderr, item.Stdout))
	if item.MonitoringLog != "" {
		w.Accent(fmt.Sprintf("%s\n", item.MonitoringLog))
	}
	if item.BackendLogs.Log != "" {
		w.Accent(fmt.Sprintf("%s\n", item.BackendLogs.Log))
	}

	fmt.Printf("🐋 Docker image:\n")
	w.Accent(fmt.Sprintf("%s\n", item.RuntimeAttributes.Docker))
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

func selectDesiredTask(m *cromwell_client.MetadataResponse, p Prompt, w Writer) ([]cromwell_client.CallItem, error) {
	var taskOptions []string
	calls := make(map[string][]cromwell_client.CallItem)
	for key, value := range m.Calls {
		sliceName := strings.Split(key, ".")
		taskName := sliceName[len(sliceName)-1]
		calls[taskName] = value
		if !contains(taskOptions, taskName) {
			taskOptions = append(taskOptions, taskName)
		}
	}
	cat := "Workflow"
	if m.RootWorkflowID != "" {
		cat = "SubWorkflow"
	}
	w.Accent(fmt.Sprintf("%s: %s\n", cat, m.WorkflowName))

	taskName, err := p.SelectByKey(taskOptions)
	if err != nil {
		fmt.Printf("Ui failed %v\n", err)
		return []cromwell_client.CallItem{}, err
	}
	return calls[taskName], nil
}

func selectDesiredShard(shards []cromwell_client.CallItem, p Prompt) (cromwell_client.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	i, err := p.SelectByIndex(searcher, shards)
	if err != nil {
		return cromwell_client.CallItem{}, err
	}

	return shards[i], err
}
