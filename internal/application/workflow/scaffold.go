// scaffold.go generates an inputs template from a WDL, so users start from
// the workflow's own declarations instead of a blank file.
package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// ScaffoldInputsUseCase builds an inputs JSON template for a workflow.
type ScaffoldInputsUseCase struct {
	fileProvider ports.FileProvider
}

// NewScaffoldInputsUseCase creates a new scaffold use case.
func NewScaffoldInputsUseCase(fp ports.FileProvider) *ScaffoldInputsUseCase {
	return &ScaffoldInputsUseCase{fileProvider: fp}
}

// ScaffoldInputsInput is the input for scaffolding.
type ScaffoldInputsInput struct {
	WorkflowFile string
	// IncludeOptional adds the optional inputs, rendered with their defaults.
	IncludeOptional bool
}

// ScaffoldInputsOutput carries the template and the declarations behind it.
type ScaffoldInputsOutput struct {
	WorkflowName string
	Template     []byte
	Inputs       []wdl.InputSpec
}

// Execute reads the WDL and renders its inputs template.
func (uc *ScaffoldInputsUseCase) Execute(ctx context.Context, input ScaffoldInputsInput) (*ScaffoldInputsOutput, error) {
	if input.WorkflowFile == "" {
		return nil, application.NewInputValidationError("workflowFile", "is required")
	}

	source, err := uc.fileProvider.ReadBytes(ctx, input.WorkflowFile)
	if err != nil {
		return nil, application.NewUseCaseError("scaffold_inputs", "failed to read workflow file", err)
	}

	scaffold, err := wdl.ScaffoldInputs(source, wdl.ScaffoldOptions{IncludeOptional: input.IncludeOptional})
	if err != nil {
		return nil, application.NewUseCaseError("scaffold_inputs", "failed to read the workflow's inputs", err)
	}

	return &ScaffoldInputsOutput{
		WorkflowName: scaffold.WorkflowName,
		Template:     scaffold.Template,
		Inputs:       scaffold.Inputs,
	}, nil
}
