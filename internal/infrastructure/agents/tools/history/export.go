package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/localfs"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// exportAction is the tool action the export handler answers.
const exportAction = "export"

// Exporter is the slice of the export use case the action needs. It is an
// interface so the tool can be built without the whole application wired.
type Exporter interface {
	Execute(ctx context.Context, input workflow.ExportRunInput) (*workflow.ExportRunOutput, error)
}

// ExportHandler answers the "export" action: a run's submitted files written
// into a directory under the working directory, ready to submit again. Like
// write_file, it never writes outside the directory pumbaa was launched from.
type ExportHandler struct {
	exporter Exporter
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(exporter Exporter) *ExportHandler {
	return &ExportHandler{exporter: exporter}
}

// Handle implements types.Handler.
func (h *ExportHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	if input.WorkflowID == "" {
		return types.NewErrorOutput(exportAction, "workflow_id is required"), nil
	}
	dir := input.Path
	if dir == "" {
		dir = input.WorkflowID
	}
	fullPath, err := localfs.ResolveWorkingDirPath(dir)
	if err != nil {
		return types.NewErrorOutput(exportAction, err.Error()), nil
	}

	out, err := h.exporter.Execute(ctx, workflow.ExportRunInput{
		WorkflowID: input.WorkflowID,
		Dir:        fullPath,
		Overwrite:  input.Overwrite,
	})
	if errors.Is(err, workflow.ErrExportWouldOverwrite) {
		return types.NewErrorOutput(exportAction, fmt.Sprintf(
			"%v. Pass overwrite=true to replace them, or another path.", err)), nil
	}
	if err != nil {
		return types.NewErrorOutput(exportAction, err.Error()), nil
	}

	files := make([]map[string]any, 0, len(out.Files))
	for _, f := range out.Files {
		entry := map[string]any{"file": f.Name, "source": string(f.Source), "size": f.Size}
		if f.Warning != "" {
			entry["note"] = f.Warning
		}
		files = append(files, entry)
	}
	data := map[string]any{
		"dir":      out.Dir,
		"files":    files,
		"complete": out.Complete(),
		"submit":   "cd " + out.Dir + " && " + out.SubmitCommand,
	}
	if len(out.Missing) > 0 {
		missing := make([]map[string]any, 0, len(out.Missing))
		for _, m := range out.Missing {
			missing = append(missing, map[string]any{"kind": string(m.Kind), "reason": m.Reason})
		}
		data["missing"] = missing
	}
	if len(out.Secrets) > 0 {
		data["options_secrets"] = out.Secrets
		data["note"] = "options.json carries credentials; `pumbaa history export --strip-secrets` drops them before the directory is shared"
	}
	return types.NewSuccessOutput(exportAction, data), nil
}
