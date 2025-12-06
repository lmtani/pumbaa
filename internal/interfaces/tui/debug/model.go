package debug

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd

	case subWorkflowLoadedMsg:
		m.isLoading = false
		m.loadingMessage = ""
		// Find the node and add children
		node := m.findNodeByID(m.tree, msg.nodeID)
		if node != nil && node.CallData != nil {
			node.CallData.SubWorkflowMetadata = msg.metadata
			// Rebuild children for this node
			addSubWorkflowChildren(node, msg.metadata, node.Depth+1)
			node.Expanded = true
			m.nodes = GetVisibleNodes(m.tree)
			m.updateDetailsContent()
		}
		return m, nil

	case subWorkflowErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.statusMessage = fmt.Sprintf("Error loading subworkflow: %s", msg.err.Error())
		return m, nil

	case logLoadedMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.logModalContent = msg.content
		m.logModalTitle = msg.title
		m.logModalError = ""
		m.logModalLoading = false
		m.showLogModal = true
		// Initialize the modal viewport
		m.logModalViewport = viewport.New(m.width-10, m.height-8)
		m.logModalViewport.SetContent(msg.content)
		return m, nil

	case logErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.logModalError = msg.err.Error()
		m.logModalLoading = false
		m.showLogModal = true
		m.logModalContent = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.treeWidth = m.width * 40 / 100 // 40% for tree
		m.detailsWidth = m.width - m.treeWidth - 4
		m.help.Width = m.width
		m.detailViewport.Width = m.detailsWidth - 4
		m.detailViewport.Height = m.height - 8
		if m.showLogModal {
			m.logModalViewport.Width = m.width - 10
			m.logModalViewport.Height = m.height - 8
		}
		m.updateDetailsContent()

	case tea.KeyMsg:
		// Handle log modal first
		if m.showLogModal {
			switch {
			case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
				m.showLogModal = false
				m.logModalContent = ""
				m.logModalError = ""
			case key.Matches(msg, m.keys.Up):
				m.logModalViewport.LineUp(1)
			case key.Matches(msg, m.keys.Down):
				m.logModalViewport.LineDown(1)
			case key.Matches(msg, m.keys.PageUp):
				m.logModalViewport.ViewUp()
			case key.Matches(msg, m.keys.PageDown):
				m.logModalViewport.ViewDown()
			case key.Matches(msg, m.keys.Home):
				m.logModalViewport.GotoTop()
			case key.Matches(msg, m.keys.End):
				m.logModalViewport.GotoBottom()
			}
			return m, nil
		}

		if m.showHelp {
			if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Quit) {
				m.showHelp = false
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.Up):
			if m.focus == FocusTree {
				if m.cursor > 0 {
					m.cursor--
					m.updateDetailsContent()
				}
			} else if m.viewMode == ViewModeLogs && m.focus == FocusDetails {
				if m.logCursor > 0 {
					m.logCursor--
					m.updateDetailsContent()
				}
			} else {
				m.detailViewport.LineUp(1)
			}

		case key.Matches(msg, m.keys.Down):
			if m.focus == FocusTree {
				if m.cursor < len(m.nodes)-1 {
					m.cursor++
					m.updateDetailsContent()
				}
			} else if m.viewMode == ViewModeLogs && m.focus == FocusDetails {
				if m.logCursor < 1 { // 0 = stdout, 1 = stderr
					m.logCursor++
					m.updateDetailsContent()
				}
			} else {
				m.detailViewport.LineDown(1)
			}

		case key.Matches(msg, m.keys.Left):
			if m.focus == FocusTree && m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.Expanded && len(node.Children) > 0 {
					node.Expanded = false
					m.nodes = GetVisibleNodes(m.tree)
				} else if node.Parent != nil {
					// Move to parent
					for i, n := range m.nodes {
						if n == node.Parent {
							m.cursor = i
							break
						}
					}
				}
				m.updateDetailsContent()
			}

		case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Space):
			// Handle opening log in logs view
			if m.viewMode == ViewModeLogs && m.focus == FocusDetails && m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.CallData != nil {
					var logPath string
					if m.logCursor == 0 {
						logPath = node.CallData.Stdout
					} else {
						logPath = node.CallData.Stderr
					}
					if logPath != "" {
						m.isLoading = true
						m.loadingMessage = "Loading log file..."
						return m, tea.Batch(m.loadingSpinner.Tick, m.openLogFile(logPath))
					}
				}
			} else if m.focus == FocusTree && m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				// Check if this is a subworkflow that needs to fetch metadata
				if node.Type == NodeTypeSubWorkflow && len(node.Children) == 0 && node.SubWorkflowID != "" {
					if node.CallData != nil && node.CallData.SubWorkflowMetadata == nil {
						// Need to fetch subworkflow metadata
						if m.fetcher != nil {
							m.isLoading = true
							m.loadingMessage = "Loading subworkflow metadata..."
							return m, tea.Batch(m.loadingSpinner.Tick, m.fetchSubWorkflowMetadata(node))
						} else {
							m.statusMessage = "Cannot fetch subworkflow: no server connection (use --id flag)"
						}
					}
				} else if len(node.Children) > 0 {
					node.Expanded = !node.Expanded
					m.nodes = GetVisibleNodes(m.tree)
				}
				m.updateDetailsContent()
			}

		case key.Matches(msg, m.keys.Tab):
			if m.focus == FocusTree {
				m.focus = FocusDetails
			} else {
				m.focus = FocusTree
			}

		case key.Matches(msg, m.keys.Details):
			m.viewMode = ViewModeTree
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Command):
			m.viewMode = ViewModeCommand
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Logs):
			m.viewMode = ViewModeLogs
			m.logCursor = 0
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Inputs):
			m.viewMode = ViewModeInputs
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Outputs):
			m.viewMode = ViewModeOutputs
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Timeline):
			m.viewMode = ViewModeTimeline
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Escape):
			m.viewMode = ViewModeTree
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.ExpandAll):
			m.expandAll(m.tree)
			m.nodes = GetVisibleNodes(m.tree)

		case key.Matches(msg, m.keys.CollapseAll):
			m.collapseAll(m.tree)
			m.nodes = GetVisibleNodes(m.tree)

		case key.Matches(msg, m.keys.Home):
			m.cursor = 0
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.End):
			m.cursor = len(m.nodes) - 1
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.PageUp):
			if m.focus == FocusTree {
				m.cursor -= 10
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.updateDetailsContent()
			} else {
				m.detailViewport.ViewUp()
			}

		case key.Matches(msg, m.keys.PageDown):
			if m.focus == FocusTree {
				m.cursor += 10
				if m.cursor >= len(m.nodes) {
					m.cursor = len(m.nodes) - 1
				}
				m.updateDetailsContent()
			} else {
				m.detailViewport.ViewDown()
			}
		}
	}

	return m, cmd
}

func (m *Model) updateDetailsContent() {
	if m.cursor >= len(m.nodes) {
		return
	}

	node := m.nodes[m.cursor]
	content := m.renderDetailsContent(node)
	m.detailViewport.SetContent(content)
	m.detailViewport.GotoTop()
}

func (m *Model) expandAll(node *TreeNode) {
	node.Expanded = true
	for _, child := range node.Children {
		m.expandAll(child)
	}
}

func (m *Model) collapseAll(node *TreeNode) {
	if node.Depth > 0 {
		node.Expanded = false
	}
	for _, child := range node.Children {
		m.collapseAll(child)
	}
}

// findNodeByID finds a node by its ID in the tree
func (m Model) findNodeByID(node *TreeNode, id string) *TreeNode {
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if found := m.findNodeByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// View renders the model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.isLoading {
		return m.renderLoading()
	}

	if m.showLogModal {
		return m.renderLogModal()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// Header
	header := m.renderHeader()

	// Main content
	treePanel := m.renderTree()
	detailsPanel := m.renderDetails()

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		treePanel,
		detailsPanel,
	)

	// Footer
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		mainContent,
		footer,
	)
}

func (m Model) renderHeader() string {
	status := StatusStyle(m.metadata.Status).Render(StatusIcon(m.metadata.Status) + " " + m.metadata.Status)
	title := titleStyle.Render(m.metadata.Name)
	id := labelStyle.Render("ID: ") + valueStyle.Render(m.metadata.ID)

	duration := ""
	if !m.metadata.End.IsZero() {
		dur := m.metadata.End.Sub(m.metadata.Start)
		duration = labelStyle.Render(" | Duration: ") + valueStyle.Render(formatDuration(dur))
	}

	// Calculate total cost
	cost := ""
	totalCost := m.calculateTotalCost()
	if totalCost > 0 {
		cost = labelStyle.Render(" | Cost: ") + valueStyle.Render(fmt.Sprintf("$%.4f", totalCost))
	}

	headerContent := fmt.Sprintf("%s  %s  %s%s%s", title, status, id, duration, cost)
	return headerStyle.Width(m.width - 2).Render(headerContent)
}

// calculateTotalCost calculates the total cost of the workflow based on VM cost per hour and duration.
func (m Model) calculateTotalCost() float64 {
	var totalCost float64

	for _, calls := range m.metadata.Calls {
		for _, call := range calls {
			if call.VMCostPerHour > 0 && !call.Start.IsZero() && !call.End.IsZero() {
				duration := call.End.Sub(call.Start)
				hours := duration.Hours()
				totalCost += call.VMCostPerHour * hours
			}
		}
	}

	return totalCost
}

func (m Model) renderTree() string {
	var sb strings.Builder

	// Calculate visible area
	treeHeight := m.height - 6
	startIdx := 0
	if m.cursor >= treeHeight {
		startIdx = m.cursor - treeHeight + 1
	}
	endIdx := startIdx + treeHeight
	if endIdx > len(m.nodes) {
		endIdx = len(m.nodes)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.nodes[i]
		line := m.renderTreeNode(node, i == m.cursor)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	style := panelStyle
	if m.focus == FocusTree {
		style = focusedPanelStyle
	}

	content := sb.String()
	return style.Width(m.treeWidth).Height(m.height - 6).Render(content)
}

func (m Model) renderTreeNode(node *TreeNode, selected bool) string {
	// Build prefix
	prefix := ""
	if node.Depth > 0 {
		for i := 0; i < node.Depth-1; i++ {
			prefix += "   "
		}
		prefix += "├──"
	}

	// Expand icon
	expandIcon := ExpandIcon(node.Expanded, len(node.Children) > 0)

	// Status icon
	statusIcon := StatusStyle(node.Status).Render(StatusIcon(node.Status))

	// Node type icon
	typeIcon := NodeTypeIcon(node.Type)

	// Name
	name := node.Name

	// Duration
	duration := ""
	if !node.End.IsZero() && !node.Start.IsZero() {
		dur := node.End.Sub(node.Start)
		duration = mutedStyle.Render(fmt.Sprintf(" (%s)", formatDuration(dur)))
	}

	line := fmt.Sprintf("%s%s %s %s %s%s", prefix, expandIcon, statusIcon, typeIcon, name, duration)

	// Truncate if needed
	maxWidth := m.treeWidth - 4
	if len(line) > maxWidth {
		line = line[:maxWidth-3] + "..."
	}

	if selected {
		return selectedNodeStyle.Render(line)
	}
	return treeNodeStyle.Render(line)
}

func (m Model) renderDetails() string {
	style := panelStyle
	if m.focus == FocusDetails {
		style = focusedPanelStyle
	}

	title := m.getDetailsTitle()
	titleBar := titleStyle.Render(title)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		m.detailViewport.View(),
	)

	return style.Width(m.detailsWidth).Height(m.height - 6).Render(content)
}

func (m Model) getDetailsTitle() string {
	switch m.viewMode {
	case ViewModeCommand:
		return "📜 Command"
	case ViewModeLogs:
		return "📋 Log Paths"
	case ViewModeInputs:
		return "📥 Inputs"
	case ViewModeOutputs:
		return "📤 Outputs"
	case ViewModeTimeline:
		return "⏱ Timeline"
	default:
		return "📊 Details"
	}
}

func (m Model) renderDetailsContent(node *TreeNode) string {
	switch m.viewMode {
	case ViewModeCommand:
		return m.renderCommand(node)
	case ViewModeLogs:
		return m.renderLogs(node)
	case ViewModeInputs:
		return m.renderInputs(node)
	case ViewModeOutputs:
		return m.renderOutputs(node)
	case ViewModeTimeline:
		return m.renderTimeline(node)
	default:
		return m.renderBasicDetails(node)
	}
}

func (m Model) renderBasicDetails(node *TreeNode) string {
	var sb strings.Builder

	if node.Type == NodeTypeWorkflow {
		sb.WriteString(labelStyle.Render("Workflow: ") + valueStyle.Render(node.Name) + "\n")
		sb.WriteString(labelStyle.Render("Status: ") + StatusStyle(node.Status).Render(node.Status) + "\n")
		sb.WriteString(labelStyle.Render("ID: ") + valueStyle.Render(node.ID) + "\n")
		sb.WriteString(labelStyle.Render("Start: ") + valueStyle.Render(node.Start.Format("2006-01-02 15:04:05")) + "\n")
		if !node.End.IsZero() {
			sb.WriteString(labelStyle.Render("End: ") + valueStyle.Render(node.End.Format("2006-01-02 15:04:05")) + "\n")
			sb.WriteString(labelStyle.Render("Duration: ") + valueStyle.Render(formatDuration(node.Duration)) + "\n")
		}
		sb.WriteString("\n")
		sb.WriteString(labelStyle.Render("Total Tasks: ") + valueStyle.Render(fmt.Sprintf("%d", len(m.metadata.Calls))) + "\n")
		return sb.String()
	}

	if node.CallData == nil {
		sb.WriteString(labelStyle.Render("Task: ") + valueStyle.Render(node.Name) + "\n")
		sb.WriteString(labelStyle.Render("Status: ") + StatusStyle(node.Status).Render(node.Status) + "\n")
		return sb.String()
	}

	cd := node.CallData

	// Basic info
	sb.WriteString(labelStyle.Render("Task: ") + valueStyle.Render(node.Name) + "\n")
	sb.WriteString(labelStyle.Render("Status: ") + StatusStyle(cd.ExecutionStatus).Render(cd.ExecutionStatus) + "\n")

	// SubWorkflow info
	if node.Type == NodeTypeSubWorkflow {
		if cd.SubWorkflowMetadata == nil && cd.SubWorkflowID != "" {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("📂 SubWorkflow") + "\n")
			sb.WriteString(labelStyle.Render("ID: ") + valueStyle.Render(cd.SubWorkflowID) + "\n")
			if m.fetcher != nil {
				sb.WriteString(mutedStyle.Render("  Press Enter/→ to load subworkflow details") + "\n")
			} else {
				sb.WriteString(warningStyle.Render("  ⚠ Cannot load: use --id flag to enable server connection") + "\n")
			}
		} else if cd.SubWorkflowMetadata != nil {
			sb.WriteString("\n")
			sb.WriteString(titleStyle.Render("📂 SubWorkflow") + "\n")
			sb.WriteString(labelStyle.Render("ID: ") + valueStyle.Render(cd.SubWorkflowMetadata.ID) + "\n")
			sb.WriteString(labelStyle.Render("Tasks: ") + valueStyle.Render(fmt.Sprintf("%d", len(cd.SubWorkflowMetadata.Calls))) + "\n")
		}
	}

	if cd.ReturnCode != nil {
		sb.WriteString(labelStyle.Render("Return Code: ") + valueStyle.Render(fmt.Sprintf("%d", *cd.ReturnCode)) + "\n")
	}

	sb.WriteString("\n")

	// Timing
	sb.WriteString(titleStyle.Render("⏱ Timing") + "\n")
	sb.WriteString(labelStyle.Render("Start: ") + valueStyle.Render(cd.Start.Format("15:04:05")) + "\n")
	if !cd.End.IsZero() {
		sb.WriteString(labelStyle.Render("End: ") + valueStyle.Render(cd.End.Format("15:04:05")) + "\n")
		sb.WriteString(labelStyle.Render("Duration: ") + valueStyle.Render(formatDuration(cd.End.Sub(cd.Start))) + "\n")
	}
	if !cd.VMStartTime.IsZero() {
		sb.WriteString(labelStyle.Render("VM Start: ") + valueStyle.Render(cd.VMStartTime.Format("15:04:05")) + "\n")
	}
	if !cd.VMEndTime.IsZero() {
		sb.WriteString(labelStyle.Render("VM End: ") + valueStyle.Render(cd.VMEndTime.Format("15:04:05")) + "\n")
	}

	sb.WriteString("\n")

	// Resources
	sb.WriteString(titleStyle.Render("💻 Resources") + "\n")
	if cd.CPU != "" {
		sb.WriteString(labelStyle.Render("CPU: ") + valueStyle.Render(cd.CPU) + "\n")
	}
	if cd.Memory != "" {
		sb.WriteString(labelStyle.Render("Memory: ") + valueStyle.Render(cd.Memory) + "\n")
	}
	if cd.Disk != "" {
		sb.WriteString(labelStyle.Render("Disk: ") + valueStyle.Render(cd.Disk) + "\n")
	}
	if cd.Preemptible != "" {
		sb.WriteString(labelStyle.Render("Preemptible: ") + valueStyle.Render(cd.Preemptible) + "\n")
	}

	sb.WriteString("\n")

	// Docker
	sb.WriteString(titleStyle.Render("🐳 Docker") + "\n")
	if cd.DockerImage != "" {
		sb.WriteString(labelStyle.Render("Image: ") + valueStyle.Render(truncate(cd.DockerImage, 50)) + "\n")
	}
	if cd.DockerSize != "" {
		sb.WriteString(labelStyle.Render("Size: ") + valueStyle.Render(cd.DockerSize) + "\n")
	}

	sb.WriteString("\n")

	// Cache
	sb.WriteString(titleStyle.Render("📦 Cache") + "\n")
	cacheStatus := "Miss"
	if cd.CacheHit {
		cacheStatus = "Hit"
	}
	sb.WriteString(labelStyle.Render("Status: ") + valueStyle.Render(cacheStatus) + "\n")
	if cd.CacheResult != "" {
		sb.WriteString(labelStyle.Render("Result: ") + valueStyle.Render(cd.CacheResult) + "\n")
	}

	// Cost
	if cd.VMCostPerHour > 0 {
		sb.WriteString("\n")
		sb.WriteString(titleStyle.Render("💰 Cost") + "\n")
		sb.WriteString(labelStyle.Render("VM Cost/Hour: ") + valueStyle.Render(fmt.Sprintf("$%.4f", cd.VMCostPerHour)) + "\n")
	}

	return sb.String()
}

func (m Model) renderCommand(node *TreeNode) string {
	if node.CallData == nil || node.CallData.CommandLine == "" {
		return mutedStyle.Render("No command available")
	}
	// Wrap text to fit the viewport width
	wrapped := wrapText(node.CallData.CommandLine, m.detailsWidth-8)
	return commandStyle.Render(wrapped)
}

func (m Model) renderLogs(node *TreeNode) string {
	if node.CallData == nil {
		return mutedStyle.Render("No logs available")
	}

	var sb strings.Builder
	cd := node.CallData

	// Show selection indicator when details panel is focused
	stdoutPrefix := "  "
	stderrPrefix := "  "
	if m.focus == FocusDetails {
		if m.logCursor == 0 {
			stdoutPrefix = "▶ "
		} else {
			stderrPrefix = "▶ "
		}
	}

	sb.WriteString(stdoutPrefix + labelStyle.Render("stdout: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.Stdout) + "\n\n")

	sb.WriteString(stderrPrefix + labelStyle.Render("stderr: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.Stderr) + "\n\n")

	sb.WriteString("  " + labelStyle.Render("Call Root: ") + "\n")
	sb.WriteString("  " + pathStyle.Render(cd.CallRoot) + "\n\n")

	if m.focus == FocusDetails {
		sb.WriteString(mutedStyle.Render("  Press Enter to view the selected log"))
	}

	return sb.String()
}

func (m Model) renderInputs(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return mutedStyle.Render("No inputs available")
	}

	var sb strings.Builder
	for k, v := range node.CallData.Inputs {
		sb.WriteString(labelStyle.Render(k+": ") + "\n")
		sb.WriteString(valueStyle.Render(fmt.Sprintf("  %v", v)) + "\n\n")
	}
	return sb.String()
}

func (m Model) renderOutputs(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Outputs) == 0 {
		return mutedStyle.Render("No outputs available")
	}

	var sb strings.Builder
	for k, v := range node.CallData.Outputs {
		sb.WriteString(labelStyle.Render(k+": ") + "\n")
		sb.WriteString(pathStyle.Render(fmt.Sprintf("  %v", v)) + "\n\n")
	}
	return sb.String()
}

func (m Model) renderTimeline(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.ExecutionEvents) == 0 {
		return mutedStyle.Render("No timeline available")
	}

	var sb strings.Builder
	for _, event := range node.CallData.ExecutionEvents {
		time := event.Start.Format("15:04:05")
		sb.WriteString(labelStyle.Render(time) + " " + valueStyle.Render(event.Description) + "\n")
	}
	return sb.String()
}

func (m Model) renderFooter() string {
	var footer string
	if m.statusMessage != "" {
		footer = warningStyle.Render(m.statusMessage)
	} else {
		footer = " ↑↓ navigate • enter expand • tab switch panel • d details • c command • L logs • ? help • q quit"
	}
	return helpBarStyle.Width(m.width - 2).Render(footer)
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
}

func (m Model) renderLoading() string {
	loadingBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(2, 4).
		Render(m.loadingSpinner.View() + "  " + m.loadingMessage)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		loadingBox,
	)
}

func (m Model) renderLogModal() string {
	modalWidth := m.width - 6
	modalHeight := m.height - 4

	// Modal title
	title := titleStyle.Render("📄 " + m.logModalTitle)

	// Modal content
	var content string
	if m.logModalError != "" {
		content = errorStyle.Render("Error: " + m.logModalError)
	} else if m.logModalLoading {
		content = mutedStyle.Render("Loading...")
	} else {
		content = m.logModalViewport.View()
	}

	// Footer with instructions
	footer := mutedStyle.Render("↑↓/PgUp/PgDn scroll • esc close")

	// Build modal box
	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		footer,
	)

	modal := modalStyle.
		Width(modalWidth).
		Height(modalHeight).
		Render(modalContent)

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// Helper functions
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Message types for async operations
type logLoadedMsg struct {
	content string
	title   string
}

type logErrorMsg struct {
	err error
}

type subWorkflowLoadedMsg struct {
	nodeID   string
	metadata *WorkflowMetadata
}

type subWorkflowErrorMsg struct {
	nodeID string
	err    error
}

// maxLogSize is the maximum log file size we'll read (1 MB)
const maxLogSize = 1 * 1024 * 1024

// fetchSubWorkflowMetadata returns a command to fetch subworkflow metadata
func (m Model) fetchSubWorkflowMetadata(node *TreeNode) tea.Cmd {
	if m.fetcher == nil || node.SubWorkflowID == "" {
		return nil
	}

	workflowID := node.SubWorkflowID
	nodeID := node.ID

	return func() tea.Msg {
		ctx := context.Background()
		data, err := m.fetcher.GetRawMetadataWithOptions(ctx, workflowID, false)
		if err != nil {
			return subWorkflowErrorMsg{nodeID: nodeID, err: err}
		}

		metadata, err := ParseMetadata(data)
		if err != nil {
			return subWorkflowErrorMsg{nodeID: nodeID, err: err}
		}

		return subWorkflowLoadedMsg{nodeID: nodeID, metadata: metadata}
	}
}

// openLogFile returns a command to load a log file asynchronously
func (m Model) openLogFile(path string) tea.Cmd {
	return func() tea.Msg {
		title := "stdout"
		if m.logCursor == 1 {
			title = "stderr"
		}

		if strings.HasPrefix(path, "gs://") {
			// Read from Google Cloud Storage
			content, err := readGCSFile(path)
			if err != nil {
				return logErrorMsg{err: err}
			}
			return logLoadedMsg{content: content, title: title}
		}

		// Read local file
		content, err := readLocalFile(path)
		if err != nil {
			return logErrorMsg{err: err}
		}
		return logLoadedMsg{content: content, title: title}
	}
}

// readGCSFile reads a file from Google Cloud Storage
func readGCSFile(path string) (string, error) {
	// Parse gs://bucket/object path
	path = strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GCS path: gs://%s", path)
	}
	bucket := parts[0]
	object := parts[1]

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Get object attributes to check size
	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %w", err)
	}

	if attrs.Size > maxLogSize {
		return "", fmt.Errorf("log file too large (%.2f MB > 1 MB limit)", float64(attrs.Size)/(1024*1024))
	}

	// Read the object
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read GCS object: %w", err)
	}

	return string(data), nil
}

// readLocalFile reads a local file
func readLocalFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxLogSize {
		return "", fmt.Errorf("log file too large (%.2f MB > 1 MB limit)", float64(info.Size())/(1024*1024))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// wrapText wraps text to fit within maxWidth characters.
// It respects existing line breaks and wraps long lines.
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= maxWidth {
			result.WriteString(line)
			continue
		}

		// Wrap long lines
		for len(line) > maxWidth {
			// Try to find a good break point (space)
			breakPoint := maxWidth
			for j := maxWidth; j > maxWidth/2; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		if len(line) > 0 {
			result.WriteString(line)
		}
	}

	return result.String()
}
