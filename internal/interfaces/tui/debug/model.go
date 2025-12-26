package debug

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// MetadataFetcher is an interface for fetching workflow metadata.
type MetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
	GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error)
}

// Model is the main model for the debug TUI.
type Model struct {
	// Data
	metadata *WorkflowMetadata
	tree     *TreeNode
	nodes    []*TreeNode

	// Metadata fetcher for on-demand subworkflow loading
	fetcher MetadataFetcher

	totalCost float64 // Cached total cost from API

	// View state persistence
	nodeStates map[string]NodeViewState

	// UI state
	cursor       int
	focus        PanelFocus
	viewMode     ViewMode
	showHelp     bool
	width        int
	height       int
	treeWidth    int
	detailsWidth int

	// Loading state
	isLoading      bool
	loadingMessage string
	loadingSpinner spinner.Model

	// Log modal state
	showLogModal          bool
	logModalContent       string // Highlighted content for display
	logModalRawContent    string // Raw content for clipboard
	logModalTitle         string
	logModalError         string
	logModalLoading       bool
	logModalViewport      viewport.Model
	logModalHScrollOffset int // Horizontal scroll offset for long lines
	logCursor             int // 0 = stdout, 1 = stderr, 2 = monitoring

	// Inputs/Outputs modal state
	showInputsModal      bool
	showOutputsModal     bool
	showOptionsModal     bool
	inputsModalViewport  viewport.Model
	outputsModalViewport viewport.Model
	optionsModalViewport viewport.Model

	// Call-level modal state
	showCallInputsModal  bool
	showCallOutputsModal bool
	showCallCommandModal bool
	callInputsViewport   viewport.Model
	callOutputsViewport  viewport.Model
	callCommandViewport  viewport.Model

	// Global timeline modal state (shows all tasks with duration)
	showGlobalTimelineModal bool
	globalTimelineViewport  viewport.Model
	globalTimelineTitle     string

	// Resource analysis modal state
	resourceReport *workflowDomain.EfficiencyReport
	resourceError  string

	// Components
	keys           KeyMap
	help           help.Model
	detailViewport viewport.Model

	// Status message
	statusMessage        string
	statusMessageExpires time.Time // When the status message should disappear

	// Infrastructure
	monitoringUC *workflow.MonitoringUseCase
	fileProvider ports.FileProvider

	// Pre-computed preemption summary when using a DebugInfo-based model
	preemption *workflowDomain.PreemptionSummary
}

// NewModelWithDebugInfoAndMonitoring creates a model with all dependencies.
func NewModelWithDebugInfoAndMonitoring(di *debuginfo.DebugInfo, fetcher MetadataFetcher, muc *workflow.MonitoringUseCase, fp ports.FileProvider) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return Model{
		metadata:       di.Metadata,
		tree:           di.Root,
		nodes:          di.Visible,
		preemption:     di.Preemption,
		fetcher:        fetcher,
		monitoringUC:   muc,
		fileProvider:   fp,
		cursor:         0,
		focus:          FocusTree,
		viewMode:       ViewModeTree,
		nodeStates:     make(map[string]NodeViewState),
		keys:           DefaultKeyMap(),
		help:           help.New(),
		detailViewport: viewport.New(80, 20),
		loadingSpinner: s,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadingSpinner.Tick,
		m.fetchTotalCost(),
	)
}
