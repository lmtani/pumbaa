package debug

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
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
	report *EfficiencyReport
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
		m.showResourceModal = true
		// Initialize viewport
		m.resourceViewport = viewport.New(m.width-10, m.height-10)
		m.resourceViewport.SetContent("")
		return m, nil

	case resourceAnalysisErrorMsg:
		m.isLoading = false
		m.loadingMessage = ""
		m.resourceError = msg.err.Error()
		m.showResourceModal = true
		m.resourceViewport = viewport.New(m.width-10, m.height-10)
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
			debuginfo.AddSubWorkflowChildren(node, msg.metadata, node.Depth+1)
			node.Expanded = true
			m.nodes = debuginfo.GetVisibleNodes(m.tree)
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

	// Handle resource analysis modal
	if m.showResourceModal {
		return m.handleResourceModalKeys(msg)
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
				m.nodes = debuginfo.GetVisibleNodes(m.tree)
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
			m.setStatusMessage("No inputs available for this workflow")
			return m, getClearStatusCmd()
		}

	case key.Matches(msg, m.keys.Outputs):
		if len(m.metadata.Outputs) > 0 {
			m.showOutputsModal = true
			m.outputsModalViewport = viewport.New(m.width-10, m.height-8)
			m.outputsModalViewport.SetContent(m.formatWorkflowOutputsForModal())
		} else {
			m.setStatusMessage("No outputs available for this workflow")
			return m, getClearStatusCmd()
		}

	case key.Matches(msg, m.keys.Options):
		if m.metadata.SubmittedOptions != "" {
			m.showOptionsModal = true
			m.optionsModalViewport = viewport.New(m.width-10, m.height-8)
			m.optionsModalViewport.SetContent(m.formatOptionsForModal())
		} else {
			m.setStatusMessage("No options available for this workflow")
			return m, getClearStatusCmd()
		}

	case key.Matches(msg, m.keys.WorkflowLog):
		// Open workflow log in modal if available
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.Type == NodeTypeWorkflow || node.Type == NodeTypeSubWorkflow {
				var metadata *WorkflowMetadata
				if node.Type == NodeTypeWorkflow {
					metadata = m.metadata
				} else if node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
					metadata = node.CallData.SubWorkflowMetadata
				}
				if metadata != nil && metadata.WorkflowLog != "" {
					m.isLoading = true
					m.loadingMessage = "Loading workflow log..."
					return m, m.openWorkflowLog(metadata.WorkflowLog)
				} else {
					m.setStatusMessage("No workflow log available")
					return m, getClearStatusCmd()
				}
			} else {
				m.setStatusMessage("Workflow log only available for workflow/subworkflow nodes")
				return m, getClearStatusCmd()
			}
		}

	case key.Matches(msg, m.keys.GlobalTimeline):
		// Check if we're on a workflow or subworkflow node to show its timeline
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			var targetMetadata *WorkflowMetadata
			var title string

			// Determine which metadata to use based on selected node
			if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
				targetMetadata = node.CallData.SubWorkflowMetadata
				title = node.Name
			} else if node.Type == NodeTypeWorkflow {
				targetMetadata = m.metadata
				title = m.metadata.Name
			} else {
				// For call/shard nodes, use root workflow
				targetMetadata = m.metadata
				title = m.metadata.Name
			}

			m.showGlobalTimelineModal = true
			m.globalTimelineTitle = title
			m.globalTimelineViewport = viewport.New(m.width-10, m.height-8)
			m.globalTimelineViewport.SetContent(m.buildGlobalTimelineContentForMetadata(targetMetadata))
		}

	case key.Matches(msg, m.keys.ResourceAnalysis):
		// Load and analyze monitoring log for the selected task
		if m.cursor < len(m.nodes) {
			node := m.nodes[m.cursor]
			if node.CallData != nil && node.CallData.MonitoringLog != "" {
				m.isLoading = true
				m.loadingMessage = "Loading monitoring log..."
				return m, m.loadResourceAnalysis(node.CallData.MonitoringLog)
			} else {
				m.setStatusMessage("No monitoring log available for this task")
				return m, getClearStatusCmd()
			}
		}

	case key.Matches(msg, m.keys.Escape):
		m.viewMode = ViewModeTree
		m.updateDetailsContent()

	case key.Matches(msg, m.keys.ExpandAll):
		m.expandAll(m.tree)
		m.nodes = debuginfo.GetVisibleNodes(m.tree)

	case key.Matches(msg, m.keys.CollapseAll):
		m.collapseAll(m.tree)
		m.nodes = debuginfo.GetVisibleNodes(m.tree)

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
			m.nodes = debuginfo.GetVisibleNodes(m.tree)
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
			m.setStatusMessage("No inputs available for this call")
			return m, getClearStatusCmd()
		}
	} else if msg.String() == "2" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && len(node.CallData.Outputs) > 0 {
			m.showCallOutputsModal = true
			m.callOutputsViewport = viewport.New(m.width-10, m.height-8)
			m.callOutputsViewport.SetContent(m.formatCallOutputsForModal(node))
		} else {
			m.setStatusMessage("No outputs available for this call")
			return m, getClearStatusCmd()
		}
	} else if msg.String() == "3" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && node.CallData.CommandLine != "" {
			m.showCallCommandModal = true
			m.callCommandViewport = viewport.New(m.width-10, m.height-8)
			m.callCommandViewport.SetContent(m.formatCallCommandForModal(node))
		} else {
			m.setStatusMessage("No command available for this call")
			return m, getClearStatusCmd()
		}
	} else if msg.String() == "4" && m.cursor < len(m.nodes) {
		node := m.nodes[m.cursor]
		if node.CallData != nil && (node.CallData.Stdout != "" || node.CallData.Stderr != "" || node.CallData.MonitoringLog != "") {
			m.viewMode = ViewModeLogs
			m.logCursor = 0
			m.updateDetailsContent()
			m.focus = FocusDetails
		} else {
			m.setStatusMessage("No logs available for this call")
			return m, getClearStatusCmd()
		}
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
