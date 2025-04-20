package cromwell

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/entities"
)

type CromwellWorkflowProvider struct {
	c *Client
}

func NewCromwellWorkflowProvider(host string) *CromwellWorkflowProvider {
	fmt.Println(host)
	return &CromwellWorkflowProvider{
		c: NewCromwellClient(host),
	}
}

func (c *CromwellWorkflowProvider) Get(uuid string) (entities.Workflow, error) {
	urlParams := map[string]string{
		"expandSubworkflows": fmt.Sprintf("%t", false),
	}
	metadata, err := c.c.Metadata(uuid, urlParams)
	if err != nil {
		return entities.Workflow{}, err
	}

	// transform metadata.Calls into map[string][]entities.Step
	steps := make(map[string][]entities.Step)
	for callName, callList := range metadata.Calls {
		for _, call := range callList {
			step := entities.Step{
				Name:    callName,
				Status:  call.ExecutionStatus,
				Start:   call.Start.String(),
				End:     call.End.String(),
				Spot:    call.CallCaching.Hit,
				Command: call.CommandLine,
			}
			steps[callName] = append(steps[callName], step)
		}
	}

	workflow := entities.Workflow{
		ID:     metadata.ID,
		Name:   metadata.WorkflowName,
		Status: metadata.Status,
		Start:  metadata.Start,
		End:    metadata.End,
		Calls:  steps,
	}
	return workflow, nil
}

func (c *CromwellWorkflowProvider) Query() ([]entities.Workflow, error) {
	urlParams := map[string]string{}
	workflows, err := c.c.Query(urlParams)
	if err != nil {
		return nil, err
	}
	var result []entities.Workflow
	for _, workflow := range workflows.Results {
		result = append(result, entities.Workflow{
			ID:     workflow.ID,
			Name:   workflow.Name,
			Status: workflow.Status,
			Start:  workflow.Start,
			End:    workflow.End,
		})
	}
	return result, nil
}
