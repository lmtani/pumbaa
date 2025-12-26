package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// Message types for async operations
type logLoadedMsg struct {
	content string
	title   string
	path    string
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

type clearStatusMsg struct{}

type costLoadedMsg struct {
	totalCost float64
}

type resourceAnalysisLoadedMsg struct {
	report *workflowDomain.EfficiencyReport
}

type resourceAnalysisErrorMsg struct {
	err error
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case clearStatusMsg:
		m.statusMessage = ""
		m.statusMessageExpires = time.Time{} // Reset expiration
		return m, nil

	case costLoadedMsg:
		m.totalCost = msg.totalCost
		return m, nil

	case clipboardCopiedMsg:
		if msg.success {
			m.setStatusMessage("✓ Copied to clipboard!")
		} else {
			m.setStatusMessage(fmt.Sprintf("✗ Copy failed: %v", msg.err))
		}
		return m, getClearStatusCmd()

	case resourceAnalysisLoadedMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.resourceReport = msg.report
		m.resourceError = ""
		m.viewMode = ViewModeMonitor

		// Cache the report in the current node
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil {
				node.CallData.EfficiencyReport = msg.report
			}
		}

		m.updateDetailsContent()
		return m, nil

	case resourceAnalysisErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.resourceError = msg.err.Error()
		m.viewMode = ViewModeMonitor
		m.updateDetailsContent()
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
			tree.AddSubWorkflowChildren(node, msg.metadata, node.Depth+1)
			node.Expanded = true
			m.nodes = tree.GetVisibleNodes(m.tree)
			m.updateDetailsContent()
		}
		return m, nil

	case subWorkflowErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.setStatusMessage(fmt.Sprintf("Error loading subworkflow: %s", msg.err.Error()))
		return m, getClearStatusCmd()

	case logLoadedMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.logModalRawContent = msg.content                                         // Keep raw content for clipboard
		m.logModalContent = common.HighlightWithFilename(msg.content, msg.path, 0) // No width limit for highlighting
		m.logModalTitle = msg.title
		m.logModalError = ""
		m.logModalLoading = false
		m.showLogModal = true
		m.logModalHScrollOffset = 0
		// Initialize the modal viewport with truncated content
		// Modal uses: width-6, minus border (2), minus padding (4) = width-12
		// Use width-14 for extra safety margin
		viewportWidth := m.width - 14
		m.logModalViewport = viewport.New(viewportWidth, m.height-10)
		truncatedContent := truncateLinesToWidth(m.logModalContent, viewportWidth)
		m.logModalViewport.SetContent(truncatedContent)
		return m, nil

	case logErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		// Show error as temporary status message instead of opening modal
		errorMsg := msg.err.Error()
		// Simplify common error messages
		if strings.Contains(errorMsg, "404") || strings.Contains(errorMsg, "No such object") {
			errorMsg = "Log file not found in storage"
		} else if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "Access Denied") {
			errorMsg = "Access denied to log file"
		} else if len(errorMsg) > 80 {
			// Truncate very long error messages
			errorMsg = errorMsg[:77] + "..."
		}
		m.setStatusMessage("Error loading log: " + errorMsg)
		return m, getClearStatusCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.treeWidth = m.width * 40 / 100 // 40% for tree
		m.detailsWidth = m.width - m.treeWidth - 4
		m.help.Width = m.width
		m.detailViewport.Width = m.detailsWidth - 4
		m.detailViewport.Height = m.height - 14 // Leave room for header, footer, and panel borders
		if m.showLogModal {
			viewportWidth := m.width - 14
			m.logModalViewport.Width = viewportWidth
			m.logModalViewport.Height = m.height - 10
			// Reapply content with new width
			scrolledContent := applyHorizontalScroll(m.logModalContent, m.logModalHScrollOffset, viewportWidth)
			truncatedContent := truncateLinesToWidth(scrolledContent, viewportWidth)
			m.logModalViewport.SetContent(truncatedContent)
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

	// Handle global timeline modal
	if m.showGlobalTimelineModal {
		return m.handleGlobalTimelineModalKeys(msg)
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) || key.Matches(msg, m.keys.Quit) {
			m.showHelp = false
		}
		return m, nil
	}

	return m.handleMainKeys(msg)
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
				m.changeSelectedNode(m.cursor - 1)
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
				m.changeSelectedNode(m.cursor + 1)
			}
		} else if m.viewMode == ViewModeLogs && m.focus == FocusDetails {
			if m.logCursor < 2 { // 0 = stdout, 1 = stderr, 2 = monitoring
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
				m.nodes = tree.GetVisibleNodes(m.tree)
			} else if node.Parent != nil {
				// Move to parent
				for i, n := range m.nodes {
					if n == node.Parent {
						m.changeSelectedNode(i)
						break
					}
				}
			} else {
				// Just collapsed, update view
				m.changeSelectedNode(m.cursor)
			}
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

	case key.Matches(msg, m.keys.Escape):
		m.viewMode = ViewModeTree
		m.updateDetailsContent()

	case key.Matches(msg, m.keys.ExpandAll):
		m.expandAll(m.tree)
		m.nodes = tree.GetVisibleNodes(m.tree)

	case key.Matches(msg, m.keys.CollapseAll):
		m.collapseAll(m.tree)
		m.nodes = tree.GetVisibleNodes(m.tree)

	case key.Matches(msg, m.keys.Home):
		m.changeSelectedNode(0)

	case key.Matches(msg, m.keys.End):
		m.changeSelectedNode(len(m.nodes) - 1)

	case key.Matches(msg, m.keys.PageUp):
		if m.focus == FocusTree {
			newCursor := m.cursor - 10
			if newCursor < 0 {
				newCursor = 0
			}
			m.changeSelectedNode(newCursor)
		} else {
			m.detailViewport.PageUp()
		}

	case key.Matches(msg, m.keys.PageDown):
		if m.focus == FocusTree {
			newCursor := m.cursor + 10
			if newCursor >= len(m.nodes) {
				newCursor = len(m.nodes) - 1
			}
			m.changeSelectedNode(newCursor)
		} else {
			m.detailViewport.PageDown()
		}

	case key.Matches(msg, m.keys.Copy):
		// Copy Docker image to clipboard when on details view
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil && node.CallData.DockerImage != "" {
				return m, copyToClipboard(node.CallData.DockerImage)
			} else {
				m.setStatusMessage("No Docker image to copy")
				return m, getClearStatusCmd()
			}
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
			switch m.logCursor {
			case 0:
				logPath = node.CallData.Stdout
			case 1:
				logPath = node.CallData.Stderr
			case 2:
				logPath = node.CallData.MonitoringLog
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
					m.setStatusMessage("Cannot fetch subworkflow: no server connection (use --id flag)")
					return m, getClearStatusCmd()
				}
			}
		} else if len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			m.nodes = tree.GetVisibleNodes(m.tree)
		}
		m.updateDetailsContent()
	}
	return m, nil
}

func (m Model) handleQuickActions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.nodes) {
		return m, nil
	}

	node := m.nodes[m.cursor]
	keyNum := msg.String()

	// Dispatch based on node type
	switch node.Type {
	case NodeTypeWorkflow, NodeTypeSubWorkflow:
		return m.handleWorkflowQuickAction(keyNum, node)
	case NodeTypeCall:
		// Check if it's a scatter (has children) or a simple task
		if len(node.Children) > 0 {
			// Scatter node - no quick actions
			return m, nil
		}
		return m.handleTaskQuickAction(keyNum, node)
	case NodeTypeShard:
		return m.handleTaskQuickAction(keyNum, node)
	}

	return m, nil
}

// handleWorkflowQuickAction handles quick actions for Workflow and SubWorkflow nodes.
// 1=Inputs [modal], 2=Outputs [modal], 3=Options [modal], 4=Timeline [modal], 5=Workflow Log [modal]
func (m Model) handleWorkflowQuickAction(keyNum string, node *TreeNode) (tea.Model, tea.Cmd) {
	// Get the appropriate metadata
	var meta *WorkflowMetadata
	if node.Type == NodeTypeWorkflow {
		meta = m.metadata
	} else if node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
		meta = node.CallData.SubWorkflowMetadata
	} else {
		meta = m.metadata // Fallback to root
	}

	switch keyNum {
	case "1": // Inputs
		if len(meta.Inputs) > 0 {
			m.showInputsModal = true
			m.inputsModalViewport = viewport.New(m.width-10, m.height-8)
			m.inputsModalViewport.SetContent(m.formatWorkflowInputsForModal())
		} else {
			m.setStatusMessage("No inputs available")
			return m, getClearStatusCmd()
		}
	case "2": // Outputs
		if len(meta.Outputs) > 0 {
			m.showOutputsModal = true
			m.outputsModalViewport = viewport.New(m.width-10, m.height-8)
			m.outputsModalViewport.SetContent(m.formatWorkflowOutputsForModal())
		} else {
			m.setStatusMessage("No outputs available")
			return m, getClearStatusCmd()
		}
	case "3": // Options
		if meta.SubmittedOptions != "" {
			m.showOptionsModal = true
			m.optionsModalViewport = viewport.New(m.width-10, m.height-8)
			m.optionsModalViewport.SetContent(m.formatOptionsForModal())
		} else {
			m.setStatusMessage("No options available")
			return m, getClearStatusCmd()
		}
	case "4": // Timeline
		m.showGlobalTimelineModal = true
		m.globalTimelineTitle = meta.Name
		m.globalTimelineViewport = viewport.New(m.width-10, m.height-8)
		m.globalTimelineViewport.SetContent(m.buildGlobalTimelineContentForMetadata(meta))
	case "5": // Workflow Log
		if meta.WorkflowLog != "" {
			m.isLoading = true
			m.loadingMessage = "Loading workflow log..."
			return m, m.openWorkflowLog(meta.WorkflowLog)
		}
		m.setStatusMessage("No workflow log available")
		return m, getClearStatusCmd()
	}

	return m, nil
}

// handleTaskQuickAction handles quick actions for Task and Shard nodes.
// 1=Inputs [modal], 2=Outputs [modal], 3=Command [modal], 4=Logs [inline], 5=Efficiency [inline]
func (m Model) handleTaskQuickAction(keyNum string, node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}

	switch keyNum {
	case "1": // Inputs
		if len(node.CallData.Inputs) > 0 {
			m.showCallInputsModal = true
			m.callInputsViewport = viewport.New(m.width-10, m.height-8)
			m.callInputsViewport.SetContent(m.formatCallInputsForModal(node))
		} else {
			m.setStatusMessage("No inputs available")
			return m, getClearStatusCmd()
		}
	case "2": // Outputs
		if len(node.CallData.Outputs) > 0 {
			m.showCallOutputsModal = true
			m.callOutputsViewport = viewport.New(m.width-10, m.height-8)
			m.callOutputsViewport.SetContent(m.formatCallOutputsForModal(node))
		} else {
			m.setStatusMessage("No outputs available")
			return m, getClearStatusCmd()
		}
	case "3": // Command
		if node.CallData.CommandLine != "" {
			m.showCallCommandModal = true
			m.callCommandViewport = viewport.New(m.width-10, m.height-8)
			m.callCommandViewport.SetContent(m.formatCallCommandForModal(node))
		} else {
			m.setStatusMessage("No command available")
			return m, getClearStatusCmd()
		}
	case "4": // Logs (inline)
		if node.CallData.Stdout != "" || node.CallData.Stderr != "" || node.CallData.MonitoringLog != "" {
			m.viewMode = ViewModeLogs
			m.logCursor = 1 // Start at stderr
			m.updateDetailsContent()
			m.focus = FocusDetails
		} else {
			m.setStatusMessage("No logs available")
			return m, getClearStatusCmd()
		}
	case "5": // Efficiency (inline)
		if node.CallData.MonitoringLog != "" {
			// Check cache first
			if node.CallData.EfficiencyReport != nil {
				m.resourceReport = node.CallData.EfficiencyReport
				m.resourceError = ""
				m.viewMode = ViewModeMonitor
				m.updateDetailsContent()
				return m, nil
			}
			// Not cached, load it
			m.viewMode = ViewModeMonitor
			m.resourceReport = nil
			m.resourceError = ""
			m.isLoading = true
			m.loadingMessage = "Analyzing resource efficiency..."
			m.updateDetailsContent()
			return m, m.loadResourceAnalysis(node.CallData.MonitoringLog)
		}
		m.setStatusMessage("No monitoring data available")
		return m, getClearStatusCmd()
	}

	return m, nil
}

// setStatusMessage sets a temporary status message that clears after a delay
func (m *Model) setStatusMessage(message string) {
	m.statusMessage = message
	m.statusMessageExpires = time.Now().Add(3 * time.Second)
}

// getClearStatusCmd returns a command to clear the status after a delay
func getClearStatusCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}
