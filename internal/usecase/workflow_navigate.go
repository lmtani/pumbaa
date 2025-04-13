package usecase

import (
	"fmt"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowNavigateInputDTO - Input
type WorkflowNavigateInputDTO struct {
	WorkflowID string
}

// WorkflowNavigate - UseCase
type WorkflowNavigate struct {
	c interfaces.CromwellServer
	w interfaces.Writer
	p interfaces.Prompt
}

// NewWorkflowNavigate - Constructor
func NewWorkflowNavigate(c interfaces.CromwellServer, w interfaces.Writer, p interfaces.Prompt) *WorkflowNavigate {
	return &WorkflowNavigate{c: c, w: w, p: p}
}

// Execute - UseCase
func (wo *WorkflowNavigate) Execute(input *WorkflowNavigateInputDTO) error {
	params := entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}
	resp, err := wo.c.Metadata(input.WorkflowID, &params)
	if err != nil {
		return err
	}
	var item entities.CallItem
	for {
		task, err := wo.selectDesiredTask(&resp)
		if err != nil {
			return err
		}
		item, err = wo.selectDesiredShard(task)
		if err != nil {
			return err
		}
		if item.SubWorkflowID == "" {
			break
		}
		resp, err = wo.c.Metadata(item.SubWorkflowID, &params)
		if err != nil {
			return err
		}
	}

	wo.w.Accent(item.ExecutionStatus)
	if item.ExecutionStatus == "QueuedInCromwell" {
		return nil
	}
	if item.CallCaching.Hit {
		wo.w.Accent(item.CallCaching.Result)
	} else {
		wo.w.Accent(item.CommandLine)
	}

	return nil
}

func (wo *WorkflowNavigate) selectDesiredTask(m *entities.MetadataResponse) ([]entities.CallItem, error) {
	var taskOptions []string
	calls := make(map[string][]entities.CallItem)
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
	wo.w.Accent(fmt.Sprintf("%s: %s\n", cat, m.WorkflowName))

	taskName, err := wo.p.SelectByKey(taskOptions)
	if err != nil {
		return []entities.CallItem{}, err
	}
	return calls[taskName], nil
}

func (wo *WorkflowNavigate) selectDesiredShard(shards []entities.CallItem) (entities.CallItem, error) {
	if len(shards) == 1 {
		return shards[0], nil
	}

	searcher := func(input string, index int) bool {
		shard := shards[index]
		name := strconv.Itoa(shard.ShardIndex)
		return name == input
	}

	i, err := wo.p.SelectByIndex(searcher, shards)
	if err != nil {
		return entities.CallItem{}, err
	}

	return shards[i], err
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
