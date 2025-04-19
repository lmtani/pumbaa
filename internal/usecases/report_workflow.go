package usecases

import (
	"errors"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/interfaces"
)

type ReportWorkflow struct {
	WorkflowProvider interfaces.WorkflowProvider
}

type ReportDTOOutput struct {
	WorkflowID string `json:"workflow_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Start      string `json:"start"`
	End        string `json:"end"`
}

func (w *ReportWorkflow) Execute(uuid string) (*entities.Workflow, error) {
	workflow, err := w.WorkflowProvider.Get(uuid)
	if err != nil {
		return nil, err
	}

	if workflow.ID == "" {
		return nil, errors.New("workflow not found")
	}
	return &workflow, nil
}
