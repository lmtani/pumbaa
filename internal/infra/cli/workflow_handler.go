package cli

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
	"github.com/lmtani/pumbaa/internal/usecase"
	urfaveCli "github.com/urfave/cli/v2"
)

// WorkflowHandler is a handler for workflows
type WorkflowHandler struct {
	CromwellClient ports.CromwellServer
	Writer         ports.Writer
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(c ports.CromwellServer, w ports.Writer) *WorkflowHandler {
	return &WorkflowHandler{CromwellClient: c, Writer: w}
}

// Query queries workflows from Cromwell
func (w *WorkflowHandler) Query(c *urfaveCli.Context) error {
	queryUseCase := usecase.NewWorkflowQuery(w.CromwellClient)
	input := &usecase.WorkflowQueryInputDTO{
		Name: c.String("name"),
		Days: time.Duration(c.Int64("days")),
	}

	output, err := queryUseCase.Execute(input)
	if err != nil {
		return err
	}

	workflows := []types.QueryResponseWorkflow{}
	for _, w := range output.Workflows {
		workflows = append(workflows, types.QueryResponseWorkflow{
			ID:                    w.ID,
			Name:                  w.Name,
			Status:                w.Status,
			Submission:            w.Submission,
			Start:                 w.Start,
			End:                   w.End,
			MetadataArchiveStatus: w.MetadataArchiveStatus,
		})
	}
	table := types.QueryResponse{
		Results:           workflows,
		TotalResultsCount: len(workflows),
	}

	w.Writer.QueryTable(table)
	w.Writer.Accent(fmt.Sprintf("Total Results: %d", len(workflows)))
	return nil
}

// Submit submits a workflow to Cromwell
func (w *WorkflowHandler) Submit(c *urfaveCli.Context) error {
	submitUseCase := usecase.NewWorkflowSubmit(w.CromwellClient)
	input := &usecase.WorkflowSubmitInputDTO{
		Wdl:          c.String("wdl"),
		Inputs:       c.String("inputs"),
		Dependencies: c.String("dependencies"),
		Options:      c.String("options"),
	}
	output, err := submitUseCase.Execute(input)
	if err != nil {
		return err
	}

	w.Writer.Json(output)
	return nil
}

// Kill kills a running job
func (w *WorkflowHandler) Kill(c *urfaveCli.Context) error {
	killUseCase := usecase.NewWorkflowKill(w.CromwellClient)
	input := &usecase.WorkflowKillInputDTO{
		WorkflowID: c.String("operation"),
	}
	output, err := killUseCase.Execute(input)
	if err != nil {
		return err
	}

	w.Writer.Json(output)
	return nil
}

// Metadata retrieves metadata from a workflow
func (w *WorkflowHandler) Metadata(c *urfaveCli.Context) error {
	metadataUseCase := usecase.NewWorkflowMetadata(w.CromwellClient)
	input := &usecase.WorkflowMetadataInputDTO{
		WorkflowID: c.String("operation"),
	}
	output, err := metadataUseCase.Execute(input)
	if err != nil {
		return err
	}

	// Cast to types.MetadataResponse
	metadata := types.MetadataResponse{
		WorkflowName:   output.Metadata.WorkflowName,
		SubmittedFiles: output.Metadata.SubmittedFiles,
		RootWorkflowID: output.Metadata.RootWorkflowID,
		Calls:          output.Metadata.Calls,
		Inputs:         output.Metadata.Inputs,
		Outputs:        output.Metadata.Outputs,
		Start:          output.Metadata.Start,
		End:            output.Metadata.End,
		Status:         output.Metadata.Status,
		Failures:       output.Metadata.Failures,
	}
	w.Writer.MetadataTable(metadata)
	w.Writer.Accent(fmt.Sprintf("Workflow ID: %s", output.WorkflowID))

	return nil
}

// Outputs retrieves outputs from a workflow
func (w *WorkflowHandler) Outputs(c *urfaveCli.Context) error {
	outputsUseCase := usecase.NewWorkflowOutputs(w.CromwellClient)
	input := &usecase.WorkflowOutputsInputDTO{
		WorkflowID: c.String("operation"),
	}
	output, err := outputsUseCase.Execute(input)
	if err != nil {
		return err
	}
	w.Writer.Json(output.Outputs)
	return nil
}

// Inputs retrieves inputs from a workflow
func (w *WorkflowHandler) Inputs(c *urfaveCli.Context) error {
	inputsUseCase := usecase.NewWorkflowInputs(w.CromwellClient)
	input := &usecase.WorkflowInputsInputDTO{
		WorkflowID: c.String("operation"),
	}
	output, err := inputsUseCase.Execute(input)
	if err != nil {
		return err
	}
	w.Writer.Json(output.Inputs)
	return nil
}

// Wait waits for operation until it is complete or fail
func (w *WorkflowHandler) Wait(c *urfaveCli.Context) error {
	waitUseCase := usecase.NewWorkflowWait(w.CromwellClient)
	input := &usecase.WorkflowWaitInputDTO{
		Operation: c.String("operation"),
		Sleep:     time.Duration(c.Int64("sleep")),
	}
	err := waitUseCase.Execute(input)
	if err != nil {
		return err
	}
	return nil
}
