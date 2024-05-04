package interactive

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Navigate struct {
	c ports.CromwellServer
	w ports.Writer
	p ports.Prompt
}

func NewNavigate(c ports.CromwellServer, w ports.Writer, p ports.Prompt) *Navigate {
	return &Navigate{c: c, w: w, p: p}
}

func (n *Navigate) Navigate(operation string) error {
	params := types.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}
	resp, err := n.c.Metadata(operation, &params)
	if err != nil {
		return err
	}
	var item types.CallItem
	for {
		task, err := n.selectDesiredTask(&resp)
		if err != nil {
			return err
		}
		item, err = n.selectDesiredShard(task)
		if err != nil {
			return err
		}
		if item.SubWorkflowID == "" {
			break
		}
		resp, err = n.c.Metadata(item.SubWorkflowID, &params)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Command status: %s\n", item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		n.w.Accent(item.CallCaching.Result)
	} else {
		n.w.Accent(item.CommandLine)
	}

	fmt.Printf("Logs:\n")
	n.w.Accent(fmt.Sprintf("%s\n%s\n", item.Stderr, item.Stdout))
	if item.MonitoringLog != "" {
		n.w.Accent(fmt.Sprintf("%s\n", item.MonitoringLog))
	}
	if item.BackendLogs.Log != "" {
		n.w.Accent(fmt.Sprintf("%s\n", item.BackendLogs.Log))
	}

	fmt.Printf("🐋 Docker image:\n")
	n.w.Accent(fmt.Sprintf("%s\n", item.RuntimeAttributes.Docker))
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

func (n *Navigate) selectDesiredTask(m *types.MetadataResponse) ([]types.CallItem, error) {
	var taskOptions []string
	calls := make(map[string][]types.CallItem)
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
	n.w.Accent(fmt.Sprintf("%s: %s\n", cat, m.WorkflowName))

	taskName, err := n.p.SelectByKey(taskOptions)
	if err != nil {
		fmt.Printf("Ui failed %v\n", err)
		return []types.CallItem{}, err
	}
	return calls[taskName], nil
}

func (n *Navigate) selectDesiredShard(shards []types.CallItem) (types.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	i, err := n.p.SelectByIndex(searcher, shards)
	if err != nil {
		return types.CallItem{}, err
	}

	return shards[i], err
}
