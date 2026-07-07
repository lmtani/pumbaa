package debug

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	adkmodel "google.golang.org/adk/model"
	adksession "google.golang.org/adk/session"
	"google.golang.org/adk/tool"

	"github.com/lmtani/pumbaa/internal/application/ports"
	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// ChatDataSelection holds user selections for what data to include in chat context.
type ChatDataSelection struct {
	Metadata      bool
	Stdout        bool
	Stderr        bool
	MonitoringLog bool
	BatchLogs     bool
}

// DefaultChatDataSelection returns the default selection (metadata and stderr enabled).
func DefaultChatDataSelection() ChatDataSelection {
	return ChatDataSelection{
		Metadata:      true,
		Stdout:        false,
		Stderr:        true,
		MonitoringLog: false,
		BatchLogs:     false,
	}
}

// Model is the main model for the debug TUI.
type Model struct {
	// Data
	metadata *WorkflowMetadata
	tree     *TreeNode
	nodes    []*TreeNode

	// Metadata fetcher for on-demand subworkflow loading and parsing
	fetcher ports.WorkflowMetadataFetcher

	totalCost float64 // Cached total cost from API

	// View state persistence
	nodeStates map[string]NodeViewState

	// UI state
	cursor           int
	focus            PanelFocus
	viewMode         ViewMode
	activeModal      ModalKind
	width            int
	height           int
	treeWidth        int
	detailsWidth     int
	treeWidthPercent int // tree panel share of the width, adjustable with < and >

	// Tree search/filter state
	searchActive         bool
	searchQuery          string
	searchMatches        []int
	searchMatchCursor    int
	searchForcedExpanded map[*TreeNode]bool

	// Failure expansion state (f key): chained fetches of failed subworkflows
	failureExpandActive   bool
	failureExpandPending  int             // in-flight subworkflow fetches
	failureFetchRequested map[string]bool // node IDs already requested

	// Watch mode state (w key): periodic metadata refresh while running
	watchActive          bool
	watchRefreshing      bool            // a refresh fetch is in flight
	watchRestoreExpanded map[string]bool // expansion snapshot to reapply after subworkflow reloads
	watchReloadSubs      map[string]bool // subworkflow node IDs still to re-fetch after a refresh

	// Loading state
	isLoading        bool
	loadingMessage   string
	loadingSpinner   spinner.Model
	loadingStartTime time.Time // When loading started, for progress bar

	// Log modal state
	logModalContent       string // Highlighted content for display
	logModalRawContent    string // Raw content for clipboard
	logModalTitle         string
	logModalError         string
	logModalLoading       bool
	logModalViewport      viewport.Model
	logModalHScrollOffset int // Horizontal scroll offset for long lines
	logCursor             int // 0 = stdout, 1 = stderr, 2 = monitoring, 3 = batch logs

	// Inputs/Outputs modal state
	inputsModalViewport  viewport.Model
	outputsModalViewport viewport.Model
	optionsModalViewport viewport.Model

	// Call-level modal state
	callInputsViewport  viewport.Model
	callOutputsViewport viewport.Model
	callCommandViewport viewport.Model

	// Global timeline modal state (shows all tasks with duration)
	globalTimelineViewport viewport.Model
	globalTimelineTitle    string

	// Resource analysis modal state
	resourceReport *workflow.EfficiencyReport
	resourceError  string

	// Batch logs modal state
	batchLogsViewport      viewport.Model
	batchLogsContent       string // Highlighted content for display
	batchLogsRawContent    string // Raw content for clipboard
	batchLogsError         string
	batchLogsLoading       bool
	batchLogsHScrollOffset int // Horizontal scroll offset for batch logs modal

	// Chat modal state
	chatDataSelections  ChatDataSelection // User's data selections
	chatSelectionCursor int               // Cursor for selection modal
	chatContextNode     *TreeNode         // Node being used for chat context

	// Copy menu modal state
	copyMenuItems  []copyMenuItem
	copyMenuCursor int

	// Failure summary modal state
	failureSummaryViewport viewport.Model
	failureSummaryRaw      string // raw text for clipboard

	// Chat dependencies (optional - nil if not configured)
	llm        adkmodel.LLM
	chatTools  []tool.Tool
	sessionSvc adksession.Service

	// canGoBack controls whether the footer advertises ESC as "back" or "quit"
	canGoBack bool

	// Error detail modal state (full text of the last error, e key)
	lastError          string
	errorModalViewport viewport.Model

	// Cost breakdown modal state ($ key)
	costViewport viewport.Model

	// Components
	keys           KeyMap
	help           help.Model
	detailViewport viewport.Model

	// Status message
	statusMessage        string
	statusMessageExpires time.Time // When the status message should disappear
	statusCopyContext    string    // What was copied (for better feedback)

	// Infrastructure
	monitoringUC *workflowapp.MonitoringUseCase
	fileProvider ports.FileProvider
	batchLogsUC  *workflowapp.GetBatchLogsUseCase

	// Pre-computed preemption summary
	preemption *workflow.PreemptionSummary
}

// ChatDependencies holds optional dependencies for chat functionality.
type ChatDependencies struct {
	LLM        adkmodel.LLM
	Tools      []tool.Tool
	SessionSvc adksession.Service
}

// NewModel creates a model with all dependencies.
// The workflow is parsed by the handler and passed in; tree building happens here.
func NewModel(wf *workflow.Workflow, fetcher ports.WorkflowMetadataFetcher, muc *workflowapp.MonitoringUseCase, fp ports.FileProvider, bluc *workflowapp.GetBatchLogsUseCase) Model {
	return NewModelWithChat(wf, fetcher, muc, fp, bluc, nil)
}

// NewModelWithChat creates a model with all dependencies including optional chat support.
func NewModelWithChat(wf *workflow.Workflow, fetcher ports.WorkflowMetadataFetcher, muc *workflowapp.MonitoringUseCase, fp ports.FileProvider, bluc *workflowapp.GetBatchLogsUseCase, chatDeps *ChatDependencies) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Build the tree and visible nodes in the TUI layer (presentation concern)
	root := tree.BuildTree(wf)
	visible := tree.GetVisibleNodes(root)

	// Calculate preemption summary (domain logic on aggregate)
	preemption := wf.CalculatePreemptionSummary()

	m := Model{
		metadata:           wf,
		tree:               root,
		nodes:              visible,
		preemption:         preemption,
		fetcher:            fetcher,
		monitoringUC:       muc,
		fileProvider:       fp,
		batchLogsUC:        bluc,
		cursor:             0,
		focus:              FocusTree,
		viewMode:           ViewModeTree,
		activeModal:        ModalNone,
		nodeStates:         make(map[string]NodeViewState),
		keys:               DefaultKeyMap(),
		help:               help.New(),
		detailViewport:     viewport.New(80, 20),
		loadingSpinner:     s,
		chatDataSelections: DefaultChatDataSelection(),
		canGoBack:          true, // Default to true, AppModel will set to false if started directly
	}

	// Add chat dependencies if provided
	if chatDeps != nil {
		m.llm = chatDeps.LLM
		m.chatTools = chatDeps.Tools
		m.sessionSvc = chatDeps.SessionSvc
	}

	return m
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadingSpinner.Tick,
		m.fetchTotalCost(),
	)
}

// SetCanGoBack sets whether ESC should show "back" or "quit" in the footer.
func (m *Model) SetCanGoBack(canGoBack bool) {
	m.canGoBack = canGoBack
}

// HasOngoingWork reports whether background work (watch mode, an in-flight
// fetch) would be interrupted by quitting.
func (m *Model) HasOngoingWork() bool {
	return m.watchActive || m.isLoading
}

// ResumeCmd re-arms self-perpetuating timers (the loading spinner) whose
// tick chain dies while the screen is hidden, since spinner ticks are only
// routed to the focused screen.
func (m *Model) ResumeCmd() tea.Cmd {
	if m.isLoading {
		return m.loadingSpinner.Tick
	}
	return nil
}
