package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the main model for the debug TUI.
type Model struct {
	// Data
	metadata *WorkflowMetadata
	tree     *TreeNode
	nodes    []*TreeNode

	// UI state
	cursor       int
	focus        PanelFocus
	viewMode     ViewMode
	showHelp     bool
	width        int
	height       int
	treeWidth    int
	detailsWidth int

	// Components
	keys           KeyMap
	help           help.Model
	detailViewport viewport.Model

	// Status message
	statusMessage string
}

// NewModel creates a new debug TUI model.
func NewModel(metadata *WorkflowMetadata) Model {
	tree := BuildTree(metadata)
	nodes := GetVisibleNodes(tree)

	return Model{
		metadata:       metadata,
		tree:           tree,
		nodes:          nodes,
		cursor:         0,
		focus:          FocusTree,
		viewMode:       ViewModeTree,
		keys:           DefaultKeyMap(),
		help:           help.New(),
		detailViewport: viewport.New(80, 20),
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.treeWidth = m.width * 40 / 100 // 40% for tree
		m.detailsWidth = m.width - m.treeWidth - 4
		m.help.Width = m.width
		m.detailViewport.Width = m.detailsWidth - 4
		m.detailViewport.Height = m.height - 8
		m.updateDetailsContent()

	case tea.KeyMsg:
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
			} else {
				m.detailViewport.LineUp(1)
			}

		case key.Matches(msg, m.keys.Down):
			if m.focus == FocusTree {
				if m.cursor < len(m.nodes)-1 {
					m.cursor++
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
			if m.focus == FocusTree && m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if len(node.Children) > 0 {
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

		case key.Matches(msg, m.keys.Command):
			m.viewMode = ViewModeCommand
			m.updateDetailsContent()

		case key.Matches(msg, m.keys.Logs):
			m.viewMode = ViewModeLogs
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

// View renders the model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
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

	headerContent := fmt.Sprintf("%s  %s  %s%s", title, status, id, duration)
	return headerStyle.Width(m.width - 2).Render(headerContent)
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
	return commandStyle.Render(node.CallData.CommandLine)
}

func (m Model) renderLogs(node *TreeNode) string {
	if node.CallData == nil {
		return mutedStyle.Render("No logs available")
	}

	var sb strings.Builder
	cd := node.CallData

	sb.WriteString(labelStyle.Render("stdout: ") + "\n")
	sb.WriteString(pathStyle.Render(cd.Stdout) + "\n\n")

	sb.WriteString(labelStyle.Render("stderr: ") + "\n")
	sb.WriteString(pathStyle.Render(cd.Stderr) + "\n\n")

	sb.WriteString(labelStyle.Render("Call Root: ") + "\n")
	sb.WriteString(pathStyle.Render(cd.CallRoot) + "\n")

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
	helpText := " ↑↓ navigate • enter expand • tab switch panel • c command • L logs • ? help • q quit"
	return helpBarStyle.Width(m.width - 2).Render(helpText)
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
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
