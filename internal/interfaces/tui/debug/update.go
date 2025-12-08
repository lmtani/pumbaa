package debug

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

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

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case clipboardCopiedMsg:
		if msg.success {
			m.copyMessage = "✓ Copied!"
		} else {
			m.copyMessage = fmt.Sprintf("✗ %v", msg.err)
		}
		return m, nil

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
		m.detailViewport.Height = m.height - 14 // Leave room for header, footer, and panel borders
		if m.showLogModal {
			m.logModalViewport.Width = m.width - 10
			m.logModalViewport.Height = m.height - 8
		}
		m.updateDetailsContent()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, cmd
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle log modal first
	if m.showLogModal {
		return m.handleLogModalKeys(msg)
	}

	// Handle inputs modal
	if m.showInputsModal {
		return m.handleInputsModalKeys(msg)
	}

	// Handle outputs modal
	if m.showOutputsModal {
		return m.handleOutputsModalKeys(msg)
	}

	// Handle options modal
	if m.showOptionsModal {
		return m.handleOptionsModalKeys(msg)
	}

	// Handle call-level inputs modal
	if m.showCallInputsModal {
		return m.handleCallInputsModalKeys(msg)
	}

	// Handle call-level outputs modal
	if m.showCallOutputsModal {
		return m.handleCallOutputsModalKeys(msg)
	}

	// Handle call-level command modal
	if m.showCallCommandModal {
		return m.handleCallCommandModalKeys(msg)
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Quit) {
			m.showHelp = false
		}
		return m, nil
	}

	return m.handleMainKeys(msg)
}

func (m Model) handleLogModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showLogModal = false
		m.logModalContent = ""
		m.logModalError = ""
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		if m.logModalContent != "" {
			return m, copyToClipboard(m.logModalContent)
		}
	case key.Matches(msg, m.keys.Up):
		m.logModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.logModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.logModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.logModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.logModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.logModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showInputsModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawInputsJSON())
	case key.Matches(msg, m.keys.Up):
		m.inputsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.inputsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.inputsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.inputsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.inputsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.inputsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showOutputsModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawOutputsJSON())
	case key.Matches(msg, m.keys.Up):
		m.outputsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.outputsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.outputsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.outputsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.outputsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.outputsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleOptionsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showOptionsModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		return m, copyToClipboard(m.getRawOptionsJSON())
	case key.Matches(msg, m.keys.Up):
		m.optionsModalViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.optionsModalViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.optionsModalViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.optionsModalViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.optionsModalViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.optionsModalViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallInputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallInputsModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil {
				return m, copyToClipboard(m.getRawCallInputsJSON(node))
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callInputsViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callInputsViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callInputsViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callInputsViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callInputsViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callInputsViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallOutputsModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallOutputsModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil {
				return m, copyToClipboard(m.getRawCallOutputsJSON(node))
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callOutputsViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callOutputsViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callOutputsViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callOutputsViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callOutputsViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callOutputsViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleCallCommandModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Quit):
		m.showCallCommandModal = false
		m.copyMessage = ""
	case key.Matches(msg, m.keys.Copy):
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil && node.CallData.CommandLine != "" {
				return m, copyToClipboard(node.CallData.CommandLine)
			}
		}
	case key.Matches(msg, m.keys.Up):
		m.callCommandViewport.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.callCommandViewport.ScrollDown(1)
	case key.Matches(msg, m.keys.PageUp):
		m.callCommandViewport.PageUp()
	case key.Matches(msg, m.keys.PageDown):
		m.callCommandViewport.PageDown()
	case key.Matches(msg, m.keys.Home):
		m.callCommandViewport.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.callCommandViewport.GotoBottom()
	}
	return m, nil
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.detailViewport.ScrollUp(1)
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
			m.detailViewport.ScrollDown(1)
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
		return m.handleExpandOrOpenLog()

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

	case key.Matches(msg, m.keys.Inputs):
		if len(m.metadata.Inputs) > 0 {
			m.showInputsModal = true
			m.inputsModalViewport = viewport.New(m.width-10, m.height-8)
			m.inputsModalViewport.SetContent(m.formatWorkflowInputsForModal())
		} else {
			m.statusMessage = "No inputs available for this workflow"
		}

	case key.Matches(msg, m.keys.Outputs):
		if len(m.metadata.Outputs) > 0 {
			m.showOutputsModal = true
			m.outputsModalViewport = viewport.New(m.width-10, m.height-8)
			m.outputsModalViewport.SetContent(m.formatWorkflowOutputsForModal())
		} else {
			m.statusMessage = "No outputs available for this workflow"
		}

	case key.Matches(msg, m.keys.Options):
		if m.metadata.SubmittedOptions != "" {
			m.showOptionsModal = true
			m.optionsModalViewport = viewport.New(m.width-10, m.height-8)
			m.optionsModalViewport.SetContent(m.formatOptionsForModal())
		} else {
			m.statusMessage = "No options available for this workflow"
		}

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
			m.detailViewport.PageUp()
		}

	case key.Matches(msg, m.keys.PageDown):
		if m.focus == FocusTree {
			m.cursor += 10
			if m.cursor >= len(m.nodes) {
				m.cursor = len(m.nodes) - 1
			}
			m.updateDetailsContent()
		} else {
			m.detailViewport.PageDown()
		}

	// Call-level quick actions (1-4)
	default:
		return m.handleQuickActions(msg)
	}

	return m, nil
}

func (m Model) handleExpandOrOpenLog() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleQuickActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "1" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && len(node.CallData.Inputs) > 0 {
			m.showCallInputsModal = true
			m.callInputsViewport = viewport.New(m.width-10, m.height-8)
			m.callInputsViewport.SetContent(m.formatCallInputsForModal(node))
		} else {
			m.statusMessage = "No inputs available for this call"
		}
	} else if msg.String() == "2" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && len(node.CallData.Outputs) > 0 {
			m.showCallOutputsModal = true
			m.callOutputsViewport = viewport.New(m.width-10, m.height-8)
			m.callOutputsViewport.SetContent(m.formatCallOutputsForModal(node))
		} else {
			m.statusMessage = "No outputs available for this call"
		}
	} else if msg.String() == "3" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && node.CallData.CommandLine != "" {
			m.showCallCommandModal = true
			m.callCommandViewport = viewport.New(m.width-10, m.height-8)
			m.callCommandViewport.SetContent(m.formatCallCommandForModal(node))
		} else {
			m.statusMessage = "No command available for this call"
		}
	} else if msg.String() == "4" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && (node.CallData.Stdout != "" || node.CallData.Stderr != "") {
			m.viewMode = ViewModeLogs
			m.logCursor = 0
			m.updateDetailsContent()
			m.focus = FocusDetails
		} else {
			m.statusMessage = "No logs available for this call"
		}
	}
	return m, nil
}

// Helper methods for tree manipulation

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
