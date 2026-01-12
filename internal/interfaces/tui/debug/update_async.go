package debug

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
)

// fetchSubWorkflowMetadata returns a command to fetch subworkflow metadata
func (m Model) fetchSubWorkflowMetadata(node *TreeNode) tea.Cmd {
	if m.fetcher == nil || node.SubWorkflowID == "" {
		return nil
	}

	workflowID := node.SubWorkflowID
	nodeID := node.ID

	return func() tea.Msg {
		ctx := context.Background()
		data, err := m.fetcher.GetRawMetadataWithOptions(ctx, workflowID, false)
		if err != nil {
			return subWorkflowErrorMsg{nodeID: nodeID, err: err}
		}

		metadata, err := cromwell.ParseDetailedMetadata(data)
		if err != nil {
			return subWorkflowErrorMsg{nodeID: nodeID, err: err}
		}

		return subWorkflowLoadedMsg{nodeID: nodeID, metadata: metadata}
	}
}

// openLogFile returns a command to load a log file asynchronously
func (m Model) openLogFile(path string) tea.Cmd {
	return func() tea.Msg {
		var title string
		switch m.logCursor {
		case 0:
			title = "stdout"
		case 1:
			title = "stderr"
		case 2:
			title = "monitoring"
		}

		if m.fileProvider == nil {
			return logErrorMsg{err: fmt.Errorf("file provider not initialized")}
		}

		content, err := m.fileProvider.Read(context.Background(), path)
		if err != nil {
			return logErrorMsg{err: err}
		}
		return logLoadedMsg{content: content, title: title, path: path}
	}
}

// openWorkflowLog returns a command to load a workflow log file asynchronously
func (m Model) openWorkflowLog(path string) tea.Cmd {
	return func() tea.Msg {
		title := "Workflow Log"

		if m.fileProvider == nil {
			return logErrorMsg{err: fmt.Errorf("file provider not initialized")}
		}

		content, err := m.fileProvider.Read(context.Background(), path)
		if err != nil {
			return logErrorMsg{err: err}
		}
		return logLoadedMsg{content: content, title: title, path: path}
	}
}

// loadBatchLogs returns a command to load Google Batch logs asynchronously.
// startTime and endTime should be the task's VM execution times (VMStartTime, VMEndTime).
// A margin of ±2h is added for safety to capture initialization and cleanup logs.
func (m Model) loadBatchLogs(jobName string, startTime, endTime time.Time) tea.Cmd {
	if m.batchLogsUC == nil {
		return func() tea.Msg {
			return batchLogsErrorMsg{err: fmt.Errorf("batch logs use case not initialized")}
		}
	}

	return func() tea.Msg {
		ctx := context.Background()

		// Add ±2h margin for safety to capture VM initialization and cleanup logs
		var adjustedStart, adjustedEnd time.Time
		if !startTime.IsZero() {
			adjustedStart = startTime.Add(-2 * time.Hour)
		}
		if !endTime.IsZero() {
			adjustedEnd = endTime.Add(2 * time.Hour)
		}

		input := workflowapp.GetBatchLogsInput{
			JobName:   jobName,
			Limit:     300,
			StartTime: adjustedStart,
			EndTime:   adjustedEnd,
		}

		output, err := m.batchLogsUC.Execute(ctx, input)
		if err != nil {
			return batchLogsErrorMsg{err: err}
		}

		return batchLogsLoadedMsg{entries: output.Entries, jobID: output.JobID}
	}
}
