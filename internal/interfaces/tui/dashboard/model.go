// Package dashboard provides the dashboard screen for the TUI.
package dashboard

import (
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

// NavigateToDebugMsg is sent when user wants to navigate to debug screen.
// This message is handled by the parent AppModel.
type NavigateToDebugMsg struct {
	Workflow *workflow.Workflow
}

// VersionCheckMsg is sent when version check completes.
type VersionCheckMsg struct {
	Info *version.VersionInfo
}

// Model represents the dashboard screen state.
type Model struct {
	width       int
	height      int
	workflows   []workflow.Workflow
	totalCount  int
	cursor      int
	scrollY     int
	keys        KeyMap
	globalKeys  common.GlobalKeys
	fetcher     ports.WorkflowQuerier
	loading     bool
	spinner     spinner.Model
	error       string
	statusMsg   string
	lastRefresh time.Time

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

	// Debug transition state
	loadingDebug       bool
	loadingDebugID     string
	metadataFetcher    ports.WorkflowMetadataFetcher
	metadataParser     ports.MetadataParser
	DebugMetadataReady []byte // Deprecated: metadata ready for debug view

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

	// Navigation state
	NavigateToDebugID string // Deprecated: use pendingNavigation
	ShouldQuit        bool
	LastError         error              // Last error for telemetry capture
	pendingNavigation tea.Cmd            // Pending navigation command for parent
	pendingWorkflow   *workflow.Workflow // Workflow to navigate to
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

// NewModelWithFetcher creates a new dashboard model with a workflow fetcher.
func NewModelWithFetcher(fetcher ports.WorkflowQuerier) Model {
	m := NewModel()
	m.fetcher = fetcher
	m.loading = true
	return m
}

// SetMetadataFetcher sets the metadata fetcher for debug transitions
func (m *Model) SetMetadataFetcher(fetcher ports.WorkflowMetadataFetcher) {
	m.metadataFetcher = fetcher
}

// SetHealthChecker sets the health checker for server status monitoring
func (m *Model) SetHealthChecker(checker ports.HealthChecker) {
	m.healthChecker = checker
}

// SetLabelManager sets the label manager for workflow labels
func (m *Model) SetLabelManager(manager ports.LabelManager) {
	m.labelManager = manager
}

// SetMetadataParser sets the metadata parser for debug transitions
func (m *Model) SetMetadataParser(parser ports.MetadataParser) {
	m.metadataParser = parser
}

// SetCurrentVersion sets the current version for version checking
func (m *Model) SetCurrentVersion(v string) {
	m.currentVersion = v
}

// HasActiveModal returns true if there's an active modal being displayed.
func (m *Model) HasActiveModal() bool {
	return m.showFilter || m.showConfirm || m.showLabelsModal
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.fetcher != nil {
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filterInput.Width = minInt(40, m.width-20)

	case spinner.TickMsg:
		if m.loading || m.loadingDebug || m.labelsLoading || m.labelsUpdating {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case workflowsLoadedMsg:
		m.loading = false
		m.workflows = msg.workflows
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
			m.statusMsg = fmt.Sprintf("✓ Workflow %s abort requested", truncateID(msg.id))
			// Refresh the list
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchWorkflows())
		} else {
			m.statusMsg = fmt.Sprintf("✗ Failed to abort: %v", msg.err)
		}

	case debugMetadataLoadedMsg:
		m.loadingDebug = false
		m.loadingDebugID = ""
		// Parse metadata and set up navigation
		if m.metadataParser != nil {
			wf, err := m.metadataParser.ParseMetadata(msg.metadata)
			if err != nil {
				m.statusMsg = fmt.Sprintf("✗ Failed to parse metadata: %v", err)
				m.LastError = err
				return m, nil
			}
			m.SetPendingNavigation(wf)
		} else {
			// Fallback for backward compatibility
			m.DebugMetadataReady = msg.metadata
			m.NavigateToDebugID = msg.workflowID
		}
		return m, nil

	case debugMetadataErrorMsg:
		m.loadingDebug = false
		m.loadingDebugID = ""
		m.statusMsg = fmt.Sprintf("✗ Failed to load metadata: %v", msg.err)
		m.LastError = msg.err

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

	case labelsLoadedMsg:
		m.labelsLoading = false
		m.labelsData = msg.labels

	case labelsErrorMsg:
		m.labelsLoading = false
		m.showLabelsModal = false
		m.statusMsg = fmt.Sprintf("✗ Failed to load labels: %v", msg.err)
		m.LastError = msg.err

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

// GetNavigationCmd returns a pending navigation command, if any.
// This is used by the parent AppModel to check if the dashboard wants to navigate.
func (m *Model) GetNavigationCmd() tea.Cmd {
	return m.pendingNavigation
}

// ClearNavigation clears the pending navigation state.
func (m *Model) ClearNavigation() {
	m.pendingNavigation = nil
	m.pendingWorkflow = nil
}

// SetPendingNavigation sets a navigation command to be executed by the parent.
func (m *Model) SetPendingNavigation(wf *workflow.Workflow) {
	m.pendingWorkflow = wf
	m.pendingNavigation = func() tea.Msg {
		return NavigateToDebugMsg{Workflow: wf}
	}
}
