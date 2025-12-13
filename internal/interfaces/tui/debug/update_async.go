package debug

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
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

		metadata, err := debuginfo.ParseMetadata(data)
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

		if strings.HasPrefix(path, "gs://") {
			// Read from Google Cloud Storage
			content, err := readGCSFile(path)
			if err != nil {
				return logErrorMsg{err: err}
			}
			return logLoadedMsg{content: content, title: title}
		}

		// Read local file
		content, err := readLocalFile(path)
		if err != nil {
			return logErrorMsg{err: err}
		}
		return logLoadedMsg{content: content, title: title}
	}
}
