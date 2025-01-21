package usecase

import (
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowQueryInputDTO - Input
type WorkflowQueryInputDTO struct {
	Name string
	Days time.Duration
}

// Workflow - Workflow minimal information
type Workflow struct {
	ID                    string
	Name                  string
	Status                string
	Submission            string
	Start                 time.Time
	End                   time.Time
	MetadataArchiveStatus string
}

// WorkflowQueryOutputDTO - Output
type WorkflowQueryOutputDTO struct {
	Workflows []Workflow
}

// WorkflowQueryUseCase is a usecase to query workflows from Cromwell
type WorkflowQueryUseCase struct {
	CromwellClient entities.CromwellServer
}

// NewWorkflowQuery creates a new WorkflowQuery usecase
func NewWorkflowQuery(c entities.CromwellServer) *WorkflowQueryUseCase {
	return &WorkflowQueryUseCase{CromwellClient: c}
}

// Execute queries workflows from Cromwell
func (w *WorkflowQueryUseCase) Execute(i *WorkflowQueryInputDTO) (*WorkflowQueryOutputDTO, error) {
	var submission time.Time
	if i.Days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * i.Days)
	}
	params := entities.ParamsQueryGet{
		Submission: submission,
		Name:       i.Name,
	}
	result, err := w.CromwellClient.Query(&params)
	if err != nil {
		return nil, err
	}
	output := &WorkflowQueryOutputDTO{}
	for _, r := range result.Results {
		output.Workflows = append(output.Workflows, Workflow{
			ID:                    r.ID,
			Name:                  r.Name,
			Status:                r.Status,
			Submission:            r.Submission,
			Start:                 r.Start,
			End:                   r.End,
			MetadataArchiveStatus: r.MetadataArchiveStatus,
		})
	}
	return output, nil
}
