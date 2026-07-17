package cromwell

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/gcs"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

const (
	defaultLogLines = 100
	maxLogLines     = 500
	// maxLocalLogSize caps local log reads, mirroring the GCS limit.
	maxLocalLogSize = 5 * 1024 * 1024
)

// ReadLogHandler handles the "read_log" action: the tail of a task's stderr
// or stdout in one call — either from an explicit log path or resolved from
// workflow_id + task.
type ReadLogHandler struct {
	repo ports.WorkflowReader
}

// NewReadLogHandler creates a new ReadLogHandler.
func NewReadLogHandler(repo ports.WorkflowReader) *ReadLogHandler {
	return &ReadLogHandler{repo: repo}
}

// Handle implements types.Handler.
func (h *ReadLogHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "read_log"

	lines := input.Lines
	if lines <= 0 {
		lines = defaultLogLines
	}
	if lines > maxLogLines {
		lines = maxLogLines
	}

	path := input.Path
	if path == "" {
		resolved, err := h.resolveLogPath(ctx, input)
		if err != nil {
			return types.NewErrorOutput(action, err.Error()), nil
		}
		path = resolved
	}

	content, err := fetchLog(ctx, path)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	tail, total := tailLines(content, lines)
	return types.NewSuccessOutput(action, map[string]any{
		"path":        path,
		"total_lines": total,
		"shown_lines": min(lines, total),
		"truncated":   total > lines,
		"content":     tail,
	}), nil
}

// resolveLogPath finds the log path for workflow_id + task (+ optional
// shard/stream) using the workflow's log listing. The latest attempt wins:
// earlier attempts were preempted or retried.
func (h *ReadLogHandler) resolveLogPath(ctx context.Context, input types.Input) (string, error) {
	if input.WorkflowID == "" || input.Task == "" {
		return "", fmt.Errorf("either path, or workflow_id and task, are required")
	}

	logs, err := h.repo.GetLogs(ctx, input.WorkflowID)
	if err != nil {
		return "", fmt.Errorf("failed to list logs: %v", err)
	}

	entries, callName := matchTaskLogs(logs, input.Task)
	if len(entries) == 0 {
		available := make([]string, 0, len(logs))
		for name := range logs {
			available = append(available, name)
		}
		sort.Strings(available)
		return "", fmt.Errorf("task %q not found in workflow logs; available: %s (for tasks inside subworkflows, call read_log with the subworkflow's workflow_id or pass the stderr path directly)",
			input.Task, strings.Join(available, ", "))
	}

	shard := -1
	if input.Shard != nil {
		shard = *input.Shard
	}
	var best *workflow.CallLog
	for i := range entries {
		e := &entries[i]
		if e.ShardIndex != shard {
			continue
		}
		if best == nil || e.Attempt > best.Attempt {
			best = e
		}
	}
	if best == nil {
		return "", fmt.Errorf("no shard %d for task %q (call name %s)", shard, input.Task, callName)
	}

	if strings.EqualFold(input.Stream, "stdout") {
		return best.Stdout, nil
	}
	return best.Stderr, nil
}

// matchTaskLogs finds the log entries for a task by exact call name or by
// its short name (the part after the workflow prefix).
func matchTaskLogs(logs map[string][]workflow.CallLog, task string) ([]workflow.CallLog, string) {
	if entries, ok := logs[task]; ok {
		return entries, task
	}
	for callName, entries := range logs {
		if strings.EqualFold(callName[strings.LastIndex(callName, ".")+1:], task) {
			return entries, callName
		}
	}
	return nil, ""
}

// fetchLog reads a log from GCS (gs:// paths) or the local filesystem
// (local Cromwell backends).
func fetchLog(ctx context.Context, path string) (string, error) {
	if strings.HasPrefix(path, "gs://") {
		return gcs.Fetch(ctx, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("log not found: %v", err)
	}
	if info.Size() > maxLocalLogSize {
		return "", fmt.Errorf("log too large: %d bytes (max 5MB); read it outside the chat", info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read log: %v", err)
	}
	return string(data), nil
}

// tailLines returns the last n lines of content and the total line count.
func tailLines(content string, n int) (string, int) {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return "", 0
	}
	all := strings.Split(trimmed, "\n")
	if len(all) <= n {
		return trimmed, len(all)
	}
	return strings.Join(all[len(all)-n:], "\n"), len(all)
}
