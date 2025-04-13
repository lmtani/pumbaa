package usecase

import (
	"fmt"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"time"
)

// WorkflowWaitInputDTO - Input
type WorkflowWaitInputDTO struct {
	Operation string
	Sleep     time.Duration
}

// WorkflowWaitUseCase is a usecase to wait for a workflow operation to complete
type WorkflowWaitUseCase struct {
	CromwellClient interfaces.CromwellServer
}

// NewWorkflowWait creates a new WorkflowWait usecase
func NewWorkflowWait(c interfaces.CromwellServer) *WorkflowWaitUseCase {
	return &WorkflowWaitUseCase{CromwellClient: c}
}

// Execute waits for a workflow operation to complete
func (w *WorkflowWaitUseCase) Execute(i *WorkflowWaitInputDTO) error {
	resp, err := w.CromwellClient.Status(i.Operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", i.Sleep)
	fmt.Printf("Status=%s\n", resp.Status)
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(i.Sleep) * time.Second)
		resp, err := w.CromwellClient.Status(i.Operation)
		if err != nil {
			return err
		}
		fmt.Printf("Status=%s\n", resp.Status)
		status = resp.Status
	}
	return nil
}
