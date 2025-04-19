package workflowformatter

import (
	"io"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/lmtani/pumbaa/internal/infrastructure/workflow_formatter/json"
	"github.com/lmtani/pumbaa/internal/infrastructure/workflow_formatter/table"
	"github.com/lmtani/pumbaa/internal/interfaces"
)

func GetFormatter(format entities.FormatType, output io.Writer) interfaces.WorkflowFormatter {
	switch format {
	case entities.JSONFormat:
		return json.NewWorkflowJsonFormatter(output)
	case entities.TableFormat, "":
		return table.NewWorkflowTableFormatter()
	default:
		// Default to table formatter for unsupported formats
		return table.NewWorkflowTableFormatter()
	}
}
