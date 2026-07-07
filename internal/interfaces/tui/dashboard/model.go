// Package dashboard provides the dashboard screen for the TUI.
package dashboard

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/version"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// VersionCheckMsg is sent when version check completes.
type VersionCheckMsg struct {
	Info *version.VersionInfo
}

// Model represents the dashboard screen state.
type Model struct {
	width                int
	height               int
	workflows            []workflow.Workflow // currently visible (possibly narrowed by the inline filter)
	allWorkflows         []workflow.Workflow // full list from the last fetch
	totalCount           int
	cursor               int
	scrollY              int
	keys                 KeyMap
	globalKeys           common.GlobalKeys
	querier              ports.WorkflowQuerier
	aborter              ports.WorkflowAborter
	loading              bool
	autoRefresh          bool // refresh the list on every periodic tick
	spinner              spinner.Model
	error                string
	statusMsg            string
	statusMessageExpires time.Time
	lastRefresh          time.Time

	// Version check
	updateInfo     *version.VersionInfo
	currentVersion string

	// Filtering
	filterInput   textinput.Model
	showFilter    bool
	filterType    string // "name" or "label"
	activeFilters FilterState

	// Confirmation modal
	showConfirm   bool
	confirmAction string
	confirmID     string

	// Help overlay
	showHelp bool

	// Error detail modal (full text of the last error)
	showError bool

	// Debug transition state
	loadingDebug    bool
	loadingDebugID  string
	metadataFetcher ports.WorkflowMetadataFetcher

	// Health status
	healthChecker ports.HealthChecker
	healthStatus  *workflow.HealthStatus

	// Labels modal state
	labelManager       ports.LabelManager
	showLabelsModal    bool
	labelsWorkflowID   string
	labelsWorkflowName string
	labelsData         map[string]string
	labelsCursor       int
	labelsLoading      bool
	labelsUpdating     bool // true when PATCH is in progress
	labelsEditing      bool
	labelsEditKey      string
	labelsEditValue    string
	labelsInput        textinput.Model
	labelsMessage      string // In-modal feedback message

	// LastError keeps the most recent error for telemetry and the error modal.
	LastError error
}

// FilterState holds the current filter configuration
// Types and key bindings are now in types.go

// NewModel creates a new dashboard model.
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.PrimaryColor)

	ti := textinput.New()
	ti.Placeholder = "Filter by workflow name..."
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		keys:        DefaultKeyMap(),
		globalKeys:  common.DefaultGlobalKeys(),
		spinner:     s,
		filterInput: ti,
		activeFilters: FilterState{
			Status: []workflow.Status{}, // Empty means all
		},
	}
}

// NewModelWithRepository creates a new dashboard model with all repository capabilities.
// The repository satisfies WorkflowQuerier, WorkflowAborter, WorkflowMetadataFetcher,
// HealthChecker, and LabelManager through interface composition.
func NewModelWithRepository(repo ports.WorkflowRepository, version string) Model {
	m := NewModel()
	m.querier = repo
	m.aborter = repo
	m.metadataFetcher = repo
	m.healthChecker = repo
	m.labelManager = repo
	m.currentVersion = version
	m.loading = true
	return m
}

// HasActiveModal returns true if there's an active modal being displayed.
func (m *Model) HasActiveModal() bool {
	return m.showFilter || m.showConfirm || m.showLabelsModal || m.showHelp || m.showError
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.querier != nil {
		cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
	}

	// Start health check if configured
	if m.healthChecker != nil {
		cmds = append(cmds, m.fetchHealthStatus(), tickHealthCheck())
	}

	// Start async version check
	if m.currentVersion != "" && m.currentVersion != "dev" {
		cmds = append(cmds, m.checkVersion())
	}

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case clearStatusMsg:
		m.statusMsg = ""
		m.statusMessageExpires = time.Time{}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filterInput.Width = minInt(40, m.width-20)

	case spinner.TickMsg:
		if m.loading || m.loadingDebug || m.labelsLoading || m.labelsUpdating || m.statusMsg != "" {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case workflowsLoadedMsg:
		m.loading = false
		m.allWorkflows = msg.workflows
		if m.filterBarVisible() {
			// Keep the live narrowing consistent with what the user is typing
			m.applyLocalFilter()
		} else {
			m.workflows = msg.workflows
		}
		m.totalCount = msg.totalCount
		m.lastRefresh = time.Now()
		m.error = ""
		// Reset cursor if out of bounds
		if m.cursor >= len(m.workflows) {
			m.cursor = maxInt(0, len(m.workflows)-1)
		}

	case workflowsErrorMsg:
		m.loading = false
		m.error = msg.err.Error()
		m.LastError = msg.err

	case abortResultMsg:
		m.showConfirm = false
		if msg.success {
			m.setStatusMessage("✓ Workflow " + truncateID(msg.id) + " abort requested")
			// Refresh the list
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows(), getClearStatusCmd())
		} else {
			m.setStatusMessage("✗ Failed to abort: " + msg.err.Error())
			cmds = append(cmds, getClearStatusCmd())
		}

	case debugMetadataLoadedMsg:
		m.loadingDebug = false
		m.loadingDebugID = ""
		// Parse metadata and hand navigation to the parent AppModel
		wf, err := m.metadataFetcher.ParseMetadata(msg.metadata)
		if err != nil {
			m.setStatusMessage("✗ Failed to parse metadata")
			m.LastError = err
			return m, getClearStatusCmd()
		}
		return m, common.NavigateCmd(common.NavigateToDebugMsg{Workflow: wf})

	case debugMetadataErrorMsg:
		m.loadingDebug = false
		m.loadingDebugID = ""
		m.LastError = msg.err
		errorMsg := friendlyError(msg.err)
		m.setStatusMessage("✗ " + errorMsg)
		return m, getClearStatusCmd()

	case healthStatusLoadedMsg:
		m.healthStatus = msg.status

	case healthStatusErrorMsg:
		// Silent fail - just don't update health status
		m.healthStatus = nil

	case VersionCheckMsg:
		// Store version info if update is available
		if msg.Info != nil && msg.Info.UpdateAvailable {
			m.updateInfo = msg.Info
		}

	case tickMsg:
		// Periodic health check
		if m.healthChecker != nil {
			cmds = append(cmds, m.fetchHealthStatus(), tickHealthCheck())
		}
		// Auto-refresh the workflow list, unless the user is mid-interaction
		if m.autoRefresh && m.querier != nil && !m.loading && !m.HasActiveModal() {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		}

	case labelsLoadedMsg:
		m.labelsLoading = false
		m.labelsData = msg.labels

	case labelsErrorMsg:
		m.labelsLoading = false
		m.showLabelsModal = false
		m.setStatusMessage(fmt.Sprintf("✗ Failed to load labels: %v", msg.err))
		m.LastError = msg.err
		cmds = append(cmds, getClearStatusCmd())

	case labelsUpdatedMsg:
		m.labelsUpdating = false
		if msg.success {
			m.labelsMessage = "✓ Label updated"
			// No re-fetch needed - we already updated labelsData optimistically
		} else {
			m.labelsMessage = fmt.Sprintf("✗ Failed: %v", msg.err)
			// On failure, refresh to get actual state from API
			m.labelsLoading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchLabels(m.labelsWorkflowID))
		}

	case tea.KeyMsg:
		// Ignore keys while loading debug
		if m.loadingDebug {
			return m, nil
		}

		// Help overlay: any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Error detail modal
		if m.showError {
			return m.handleErrorModalKeys(msg)
		}

		// Handle confirmation modal first
		if m.showConfirm {
			return m.handleConfirmKeys(msg)
		}

		// Handle labels modal
		if m.showLabelsModal {
			return m.handleLabelsModalKeys(msg)
		}

		// Handle filter input
		if m.showFilter {
			return m.handleFilterKeys(msg)
		}

		return m.handleMainKeys(msg)
	}

	return m, tea.Batch(cmds...)
}

// View and rendering methods are in separate files:
// - view.go: View() implementation
// - view_header.go: renderHeader(), renderDebugLoadingScreen()
// - view_content.go: renderContent(), renderFilterInput(), renderConfirmModal()
// - view_table.go: renderTable(), renderWorkflowRow(), getColumnWidths()
// - view_footer.go: renderFooter()
// Helper functions are in helpers.go

// setStatusMessage sets a temporary status message that auto-clears after 3 seconds.
func (m *Model) setStatusMessage(message string) {
	m.statusMsg = message
	m.statusMessageExpires = time.Now().Add(statusDuration)
}

// getClearStatusCmd returns a command to clear the status message after the duration.
func getClearStatusCmd() tea.Cmd {
	return tea.Tick(statusDuration, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// friendlyError translates infrastructure errors into user-facing messages.
func friendlyError(err error) string {
	if errors.Is(err, workflow.ErrWorkflowNotFound) {
		return "Workflow not found"
	}
	if errors.Is(err, workflow.ErrConnectionFailed) {
		return "Cannot connect to server"
	}
	var apiErr workflow.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 400 {
			return "Invalid workflow ID"
		}
		return fmt.Sprintf("Server error (HTTP %d)", apiErr.StatusCode)
	}
	msg := err.Error()
	if len(msg) > 80 {
		return common.Truncate(msg, 80)
	}
	return msg
}
