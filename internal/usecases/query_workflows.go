package usecases

import (
	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/interfaces"
)

type QueryWorkflows struct {
	WorkflowProvider interfaces.WorkflowProvider
}

type QueryDTOOutput struct {
	Workflows []entities.Workflow `json:"workflows"`
}

func (w *QueryWorkflows) Execute() (*QueryDTOOutput, error) {
	workflows, err := w.WorkflowProvider.Query()
	if err != nil {
		return nil, err
	}
	if len(workflows) == 0 {
		return &QueryDTOOutput{}, nil
	}

	var output QueryDTOOutput
	output.Workflows = make([]entities.Workflow, len(workflows))
	copy(output.Workflows, workflows)
	return &output, nil
}
