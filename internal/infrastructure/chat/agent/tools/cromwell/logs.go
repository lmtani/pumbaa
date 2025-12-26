package cromwell

import (
	"context"

	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/types"
)

// LogsHandler handles the "logs" action to get workflow log file paths.
type LogsHandler struct {
	repo Repository
}

// NewLogsHandler creates a new LogsHandler.
func NewLogsHandler(repo Repository) *LogsHandler {
	return &LogsHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *LogsHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	if input.WorkflowID == "" {
		return types.NewErrorOutput("logs", "workflow_id is required"), nil
	}

	logs, err := h.repo.GetLogs(ctx, input.WorkflowID)
	if err != nil {
		return types.NewErrorOutput("logs", err.Error()), nil
	}

	callLogs := make(map[string]interface{})
	for callName, logEntries := range logs {
		entries := make([]map[string]interface{}, 0, len(logEntries))
		for _, entry := range logEntries {
			entries = append(entries, map[string]interface{}{
				"stdout":      entry.Stdout,
				"stderr":      entry.Stderr,
				"attempt":     entry.Attempt,
				"shard_index": entry.ShardIndex,
			})
		}
		callLogs[callName] = entries
	}

	return types.NewSuccessOutput("logs", map[string]interface{}{
		"id":    input.WorkflowID,
		"calls": callLogs,
	}), nil
}
