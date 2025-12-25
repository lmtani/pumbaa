package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// handleMainKeys processes keyboard input during normal navigation.
func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.globalKeys.Quit):
		m.ShouldQuit = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.workflows)-1 {
			m.cursor++
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Home):
		m.cursor = 0
		m.scrollY = 0

	case key.Matches(msg, m.keys.End):
		m.cursor = maxInt(0, len(m.workflows)-1)
		m.ensureVisible()

	case key.Matches(msg, m.keys.PageUp):
		m.cursor = maxInt(0, m.cursor-10)
		m.ensureVisible()

	case key.Matches(msg, m.keys.PageDown):
		m.cursor = minInt(len(m.workflows)-1, m.cursor+10)
		m.ensureVisible()

	case key.Matches(msg, m.keys.Refresh):
		if m.fetcher != nil && !m.loading {
			m.loading = true
			m.statusMsg = ""
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case key.Matches(msg, m.keys.Open):
		if len(m.workflows) > 0 && m.cursor < len(m.workflows) {
			wf := m.workflows[m.cursor]
			// If we have a metadata fetcher, load metadata with loading screen
			if m.metadataFetcher != nil {
				m.loadingDebug = true
				m.loadingDebugID = wf.ID
				cmds = append(cmds, m.spinner.Tick, m.fetchDebugMetadata(wf.ID))
			} else {
				// Fallback to old behavior if no metadata fetcher
				m.NavigateToDebugID = wf.ID
				return m, tea.Quit
			}
		}

	case key.Matches(msg, m.keys.Abort):
		if len(m.workflows) > 0 && m.cursor < len(m.workflows) {
			wf := m.workflows[m.cursor]
			// Only allow aborting running/submitted workflows
			if wf.Status == workflow.StatusRunning || wf.Status == workflow.StatusSubmitted {
				m.showConfirm = true
				m.confirmAction = "abort"
				m.confirmID = wf.ID
			} else {
				m.statusMsg = "Can only abort Running or Submitted workflows"
			}
		}

	case key.Matches(msg, m.keys.Filter):
		m.showFilter = true
		m.filterType = "name"
		m.filterInput.Placeholder = "Filter by workflow name..."
		m.filterInput.SetValue(m.activeFilters.Name)
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.LabelFilter):
		m.showFilter = true
		m.filterType = "label"
		m.filterInput.Placeholder = "Filter by label (key:value)..."
		m.filterInput.SetValue(m.activeFilters.Label)
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.ClearFilter):
		m.activeFilters = FilterState{Status: []workflow.Status{}}
		m.filterInput.SetValue("")
		m.filterType = ""
		if m.fetcher != nil {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case key.Matches(msg, m.keys.StatusFilter):
		m.cycleStatusFilter()
		if m.fetcher != nil {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case key.Matches(msg, m.keys.LabelsManager):
		if len(m.workflows) > 0 && m.cursor < len(m.workflows) && m.labelManager != nil {
			wf := m.workflows[m.cursor]
			m.showLabelsModal = true
			m.labelsWorkflowID = wf.ID
			m.labelsWorkflowName = wf.Name
			m.labelsLoading = true
			m.labelsData = nil
			m.labelsCursor = 0
			cmds = append(cmds, m.spinner.Tick, m.fetchLabels(wf.ID))
		}
	}

	return m, tea.Batch(cmds...)
}

// handleFilterKeys processes keyboard input when filter input is active.
func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyEsc:
		m.showFilter = false
		m.filterInput.Blur()
		return m, nil

	case tea.KeyEnter:
		m.showFilter = false
		m.filterInput.Blur()
		if m.filterType == "label" {
			m.activeFilters.Label = m.filterInput.Value()
		} else {
			m.activeFilters.Name = m.filterInput.Value()
		}
		if m.fetcher != nil {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchWorkflows())
		}
		return m, nil
	}

	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

// handleConfirmKeys processes keyboard input in the confirmation modal.
func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.confirmAction == "abort" && m.fetcher != nil {
			return m, m.abortWorkflow(m.confirmID)
		}
		m.showConfirm = false

	case "n", "N", "esc":
		m.showConfirm = false
	}

	return m, nil
}

// cycleStatusFilter cycles through status filter options.
func (m *Model) cycleStatusFilter() {
	// Cycle through: All -> Running -> Failed -> Succeeded -> All
	if len(m.activeFilters.Status) == 0 {
		m.activeFilters.Status = []workflow.Status{workflow.StatusRunning, workflow.StatusSubmitted}
	} else if containsStatus(m.activeFilters.Status, workflow.StatusRunning) {
		m.activeFilters.Status = []workflow.Status{workflow.StatusFailed}
	} else if containsStatus(m.activeFilters.Status, workflow.StatusFailed) {
		m.activeFilters.Status = []workflow.Status{workflow.StatusSucceeded}
	} else {
		m.activeFilters.Status = []workflow.Status{}
	}
}
