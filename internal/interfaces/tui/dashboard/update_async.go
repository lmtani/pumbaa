package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// fetchWorkflows fetches workflows with applied filters and returns a message.
func (m Model) fetchWorkflows() tea.Cmd {
	return func() tea.Msg {
		if m.fetcher == nil {
			return workflowsErrorMsg{err: fmt.Errorf("no fetcher configured")}
		}

		filter := workflow.QueryFilter{
			Status:   m.activeFilters.Status,
			Name:     m.activeFilters.Name,
			PageSize: 100,
		}

		// Parse label filter (format: key:value)
		if m.activeFilters.Label != "" {
			parts := strings.SplitN(m.activeFilters.Label, ":", 2)
			if len(parts) == 2 {
				filter.Labels = map[string]string{parts[0]: parts[1]}
			} else {
				// If no colon, search by key only (empty value)
				filter.Labels = map[string]string{m.activeFilters.Label: ""}
			}
		}

		result, err := m.fetcher.Query(context.Background(), filter)
		if err != nil {
			return workflowsErrorMsg{err: err}
		}

		// Sort by submission time (newest first)
		sort.Slice(result.Workflows, func(i, j int) bool {
			return result.Workflows[i].SubmittedAt.After(result.Workflows[j].SubmittedAt)
		})

		return workflowsLoadedMsg{
			workflows:  result.Workflows,
			totalCount: result.TotalCount,
		}
	}
}

// abortWorkflow aborts a workflow and returns a message.
func (m Model) abortWorkflow(id string) tea.Cmd {
	return func() tea.Msg {
		if m.fetcher == nil {
			return abortResultMsg{success: false, id: id, err: fmt.Errorf("no fetcher configured")}
		}

		err := m.fetcher.Abort(context.Background(), id)
		if err != nil {
			return abortResultMsg{success: false, id: id, err: err}
		}

		return abortResultMsg{success: true, id: id}
	}
}

// fetchDebugMetadata fetches debug metadata for a workflow.
func (m Model) fetchDebugMetadata(workflowID string) tea.Cmd {
	return func() tea.Msg {
		if m.metadataFetcher == nil {
			return debugMetadataErrorMsg{err: fmt.Errorf("no metadata fetcher configured")}
		}

		metadata, err := m.metadataFetcher.GetRawMetadataWithOptions(context.Background(), workflowID, false)
		if err != nil {
			return debugMetadataErrorMsg{err: err}
		}

		return debugMetadataLoadedMsg{
			workflowID: workflowID,
			metadata:   metadata,
		}
	}
}

// fetchHealthStatus fetches the Cromwell server health status.
func (m Model) fetchHealthStatus() tea.Cmd {
	return func() tea.Msg {
		if m.healthChecker == nil {
			return healthStatusErrorMsg{err: fmt.Errorf("no health checker configured")}
		}

		status, err := m.healthChecker.GetHealthStatus(context.Background())
		if err != nil {
			return healthStatusErrorMsg{err: err}
		}

		// Domain type returned directly from client
		return healthStatusLoadedMsg{status: status}
	}
}

// fetchLabels fetches labels for a workflow.
func (m Model) fetchLabels(workflowID string) tea.Cmd {
	return func() tea.Msg {
		if m.labelManager == nil {
			return labelsErrorMsg{err: fmt.Errorf("no label manager configured")}
		}

		labels, err := m.labelManager.GetLabels(context.Background(), workflowID)
		if err != nil {
			return labelsErrorMsg{err: err}
		}

		return labelsLoadedMsg{labels: labels}
	}
}

// updateLabels updates labels for a workflow.
func (m Model) updateLabels(workflowID string, labels map[string]string) tea.Cmd {
	return func() tea.Msg {
		if m.labelManager == nil {
			return labelsUpdatedMsg{success: false, err: fmt.Errorf("no label manager configured")}
		}

		err := m.labelManager.UpdateLabels(context.Background(), workflowID, labels)
		if err != nil {
			return labelsUpdatedMsg{success: false, err: err}
		}

		return labelsUpdatedMsg{success: true}
	}
}

// tickHealthCheck returns a ticker command for periodic health checks.
func tickHealthCheck() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
