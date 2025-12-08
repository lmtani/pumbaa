package debug

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MetadataFetcher is an interface for fetching workflow metadata.
type MetadataFetcher interface {
	GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error)
}

// Model is the main model for the debug TUI.
type Model struct {
	// Data
	metadata *WorkflowMetadata
	tree     *TreeNode
	nodes    []*TreeNode

	// Metadata fetcher for on-demand subworkflow loading
	fetcher MetadataFetcher

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
	showLogModal     bool
	logModalContent  string
	logModalTitle    string
	logModalError    string
	logModalLoading  bool
	logModalViewport viewport.Model
	logCursor        int // 0 = stdout, 1 = stderr

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

	// Copy feedback message (shown in modal footer)
	copyMessage string

	// Components
	keys           KeyMap
	help           help.Model
	detailViewport viewport.Model

	// Status message
	statusMessage string
}

// NewModel creates a new debug TUI model.
func NewModel(metadata *WorkflowMetadata) Model {
	return NewModelWithFetcher(metadata, nil)
}

// NewModelWithFetcher creates a new debug TUI model with a metadata fetcher.
func NewModelWithFetcher(metadata *WorkflowMetadata, fetcher MetadataFetcher) Model {
	tree := BuildTree(metadata)
	nodes := GetVisibleNodes(tree)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return Model{
		metadata:       metadata,
		tree:           tree,
		nodes:          nodes,
		fetcher:        fetcher,
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
	return m.loadingSpinner.Tick
}
