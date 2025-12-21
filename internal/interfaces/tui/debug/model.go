package debug

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
	monitoringuc "github.com/lmtani/pumbaa/internal/application/workflow/monitoring"
	"github.com/lmtani/pumbaa/internal/domain/workflow/monitoring"
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
	showResourceModal bool
	resourceReport    *monitoring.EfficiencyReport
	resourceError     string
	resourceViewport  viewport.Model

	// Components
	keys           KeyMap
	help           help.Model
	detailViewport viewport.Model

	// Status message
	statusMessage        string
	statusMessageExpires time.Time // When the status message should disappear

	// Infrastructure
	monitoringUC monitoringuc.Usecase
	fileProvider monitoring.FileProvider

	// Pre-computed preemption summary when using a DebugInfo-based model
	preemption *debuginfo.WorkflowPreemptionSummary
}

// NewModel creates a new debug TUI model.
func NewModel(metadata *WorkflowMetadata) Model {
	return NewModelWithFetcher(metadata, nil)
}

// NewModelWithFetcher creates a new debug TUI model with a metadata fetcher.
func NewModelWithFetcher(metadata *WorkflowMetadata, fetcher MetadataFetcher) Model {
	return NewModelWithAllDependencies(metadata, fetcher, nil, nil)
}

// NewModelWithAllDependencies creates a model with metadata and all optional dependencies.
func NewModelWithAllDependencies(metadata *WorkflowMetadata, fetcher MetadataFetcher, muc monitoringuc.Usecase, fp monitoring.FileProvider) Model {
	tree := debuginfo.BuildTree(metadata)
	nodes := debuginfo.GetVisibleNodes(tree)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return Model{
		metadata:       metadata,
		tree:           tree,
		nodes:          nodes,
		fetcher:        fetcher,
		monitoringUC:   muc,
		fileProvider:   fp,
		cursor:         0,
		focus:          FocusTree,
		viewMode:       ViewModeTree,
		keys:           DefaultKeyMap(),
		help:           help.New(),
		detailViewport: viewport.New(80, 20),
		loadingSpinner: s,
	}
}

// NewModelWithDebugInfo creates a model from a precomputed DebugInfo.
func NewModelWithDebugInfo(di *debuginfo.DebugInfo, fetcher MetadataFetcher) Model {
	return NewModelWithDebugInfoAndMonitoring(di, fetcher, nil, nil)
}

// NewModelWithDebugInfoAndMonitoring creates a model with all dependencies.
func NewModelWithDebugInfoAndMonitoring(di *debuginfo.DebugInfo, fetcher MetadataFetcher, muc monitoringuc.Usecase, fp monitoring.FileProvider) Model {
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
