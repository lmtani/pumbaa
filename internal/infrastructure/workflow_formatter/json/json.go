package json

import (
	"encoding/json"
	"io"
	"os"

	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowJsonFormatter formats workflows as JSON and writes to the specified output
type WorkflowJsonFormatter struct {
	Output io.Writer
}

// NewWorkflowJsonFormatter creates a new formatter with the specified output
// If output is nil, stdout is used as the default
func NewWorkflowJsonFormatter(output io.Writer) *WorkflowJsonFormatter {
	if output == nil {
		output = os.Stdout
	}
	return &WorkflowJsonFormatter{
		Output: output,
	}
}

// Query formats and writes multiple workflows as JSON
func (f *WorkflowJsonFormatter) Query(workflows []entities.Workflow) error {
	encoder := json.NewEncoder(f.Output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(workflows)
}

// Report formats and writes a single workflow as JSON
func (f *WorkflowJsonFormatter) Report(workflow *entities.Workflow) error {
	encoder := json.NewEncoder(f.Output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(workflow)
}
