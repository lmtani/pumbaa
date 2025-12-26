// Package dashboard provides the dashboard screen for the TUI.
package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// WorkflowFetcher interface for fetching workflows
type WorkflowFetcher interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	Abort(ctx context.Context, workflowID string) error
}

// MetadataFetcher interface for fetching workflow metadata (for debug transition)
type MetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
}

// HealthChecker provides health status checking for the workflow server.
type HealthChecker interface {
	GetHealthStatus(ctx context.Context) (*workflow.HealthStatus, error)
}

// LabelManager provides label management for workflows.
type LabelManager interface {
	GetLabels(ctx context.Context, workflowID string) (map[string]string, error)
	UpdateLabels(ctx context.Context, workflowID string, labels map[string]string) error
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
	fetcher     WorkflowFetcher
	loading     bool
	spinner     spinner.Model
	error       string
	statusMsg   string
	lastRefresh time.Time

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
	metadataFetcher    MetadataFetcher
	DebugMetadataReady []byte // Metadata ready for debug view

	// Health status
	healthChecker HealthChecker
	healthStatus  *workflow.HealthStatus

	// Labels modal state
	labelManager       LabelManager
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

	// Navigation state (for external handlers to check)
	NavigateToDebugID string
	ShouldQuit        bool
	LastError         error // Last error for telemetry capture
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
func NewModelWithFetcher(fetcher WorkflowFetcher) Model {
	m := NewModel()
	m.fetcher = fetcher
	m.loading = true
	return m
}

// SetMetadataFetcher sets the metadata fetcher for debug transitions
func (m *Model) SetMetadataFetcher(fetcher MetadataFetcher) {
	m.metadataFetcher = fetcher
}

// SetHealthChecker sets the health checker for server status monitoring
func (m *Model) SetHealthChecker(checker HealthChecker) {
	m.healthChecker = checker
}

// SetLabelManager sets the label manager for workflow labels
func (m *Model) SetLabelManager(manager LabelManager) {
	m.labelManager = manager
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
		m.DebugMetadataReady = msg.metadata
		m.NavigateToDebugID = msg.workflowID
		return m, tea.Quit

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
