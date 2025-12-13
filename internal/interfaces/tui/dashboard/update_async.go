package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
