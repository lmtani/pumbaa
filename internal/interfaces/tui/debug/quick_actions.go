package debug

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// quickAction couples a quick-action key with its footer label and behavior,
// so key dispatch and the footer hints derive from the same table and can
// never drift.
type quickAction struct {
	key     string             // primary key, shown in the footer
	alias   string             // optional secondary key
	label   string             // footer description
	visible func(m Model) bool // nil = always advertised in the footer
	run     func(m Model, node *TreeNode) (tea.Model, tea.Cmd)
}

// quickActionsFor returns the quick actions available for the node type.
func (m Model) quickActionsFor(node *TreeNode) []quickAction {
	switch node.Type {
	case NodeTypeWorkflow, NodeTypeSubWorkflow:
		return workflowQuickActions()
	case NodeTypeCall:
		if len(node.Children) > 0 {
			// Scatter node: no quick actions
			return nil
		}
		return taskQuickActions()
	case NodeTypeShard:
		return taskQuickActions()
	}
	return nil
}

func workflowQuickActions() []quickAction {
	return []quickAction{
		{key: "1", label: "inputs", run: Model.openWorkflowInputs},
		{key: "2", label: "outputs", run: Model.openWorkflowOutputs},
		{key: "3", label: "options", run: Model.openWorkflowOptions},
		{key: "4", label: "timeline", run: Model.openWorkflowTimeline},
		{key: "5", label: "log", run: Model.openWorkflowLogModal},
	}
}

func taskQuickActions() []quickAction {
	return []quickAction{
		{key: "1", label: "inputs", run: Model.openCallInputs},
		{key: "2", label: "outputs", run: Model.openCallOutputs},
		{key: "3", label: "cmd", run: Model.openCallCommand},
		{key: "4", label: "logs", run: Model.openTaskLogs},
		{key: "5", label: "efficiency", run: Model.openTaskEfficiency},
		{
			key:     "a",
			alias:   "6",
			label:   "chat",
			visible: func(m Model) bool { return m.llm != nil },
			run:     Model.openChatSelectionModal,
		},
	}
}

// workflowMetaFor returns the metadata backing a workflow/subworkflow node,
// falling back to the root workflow when the subworkflow isn't loaded yet.
func (m Model) workflowMetaFor(node *TreeNode) *WorkflowMetadata {
	if node.Type == NodeTypeSubWorkflow && node.CallData != nil && node.CallData.SubWorkflowMetadata != nil {
		return node.CallData.SubWorkflowMetadata
	}
	return m.metadata
}

// Workflow/subworkflow quick actions

func (m Model) openWorkflowInputs(node *TreeNode) (tea.Model, tea.Cmd) {
	meta := m.workflowMetaFor(node)
	if meta.SubmittedInputs == "" {
		m.setStatusMessage("No inputs available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalInputs
	m.inputsModalViewport = viewport.New(m.width-10, m.height-8)
	m.inputsModalViewport.SetContent(m.formatWorkflowInputsForModal())
	return m, nil
}

func (m Model) openWorkflowOutputs(node *TreeNode) (tea.Model, tea.Cmd) {
	meta := m.workflowMetaFor(node)
	if len(meta.Outputs) == 0 {
		m.setStatusMessage("No outputs available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalOutputs
	m.outputsModalViewport = viewport.New(m.width-10, m.height-8)
	m.outputsModalViewport.SetContent(m.formatWorkflowOutputsForModal())
	return m, nil
}

func (m Model) openWorkflowOptions(node *TreeNode) (tea.Model, tea.Cmd) {
	meta := m.workflowMetaFor(node)
	if meta.SubmittedOptions == "" {
		m.setStatusMessage("No options available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalOptions
	m.optionsModalViewport = viewport.New(m.width-10, m.height-8)
	m.optionsModalViewport.SetContent(m.formatOptionsForModal())
	return m, nil
}

func (m Model) openWorkflowTimeline(node *TreeNode) (tea.Model, tea.Cmd) {
	meta := m.workflowMetaFor(node)
	m.activeModal = ModalGlobalTimeline
	m.globalTimelineTitle = meta.Name
	m.globalTimelineViewport = viewport.New(m.width-10, m.height-8)
	m.globalTimelineViewport.SetContent(m.buildGlobalTimelineContentForMetadata(meta))
	return m, nil
}

func (m Model) openWorkflowLogModal(node *TreeNode) (tea.Model, tea.Cmd) {
	meta := m.workflowMetaFor(node)
	if meta.WorkflowLog == "" {
		m.setStatusMessage("No workflow log available")
		return m, getClearStatusCmd()
	}
	m.isLoading = true
	m.loadingMessage = "Loading workflow log..."
	m.loadingStartTime = time.Now()
	return m, m.openWorkflowLog(meta.WorkflowLog)
}

// Task/shard quick actions

func (m Model) openCallInputs(node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}
	if len(node.CallData.Inputs) == 0 {
		m.setStatusMessage("No inputs available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalCallInputs
	m.callInputsViewport = viewport.New(m.width-10, m.height-8)
	m.callInputsViewport.SetContent(m.formatCallInputsForModal(node))
	return m, nil
}

func (m Model) openCallOutputs(node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}
	if len(node.CallData.Outputs) == 0 {
		m.setStatusMessage("No outputs available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalCallOutputs
	m.callOutputsViewport = viewport.New(m.width-10, m.height-8)
	m.callOutputsViewport.SetContent(m.formatCallOutputsForModal(node))
	return m, nil
}

func (m Model) openCallCommand(node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}
	if node.CallData.CommandLine == "" {
		m.setStatusMessage("No command available")
		return m, getClearStatusCmd()
	}
	m.activeModal = ModalCallCommand
	m.callCommandViewport = viewport.New(m.width-10, m.height-8)
	m.callCommandViewport.SetContent(m.formatCallCommandForModal(node))
	return m, nil
}

func (m Model) openTaskLogs(node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}
	if node.CallData.Stdout == "" && node.CallData.Stderr == "" && node.CallData.MonitoringLog == "" && !m.canShowBatchLogs(node) {
		m.setStatusMessage("No logs available")
		return m, getClearStatusCmd()
	}
	m.viewMode = ViewModeLogs
	m.logCursor = 1 // Start at stderr
	m.updateDetailsContent()
	m.focus = FocusDetails
	return m, nil
}

func (m Model) openTaskEfficiency(node *TreeNode) (tea.Model, tea.Cmd) {
	if node.CallData == nil {
		return m, nil
	}
	if node.CallData.MonitoringLog == "" {
		m.setStatusMessage("No monitoring data available")
		return m, getClearStatusCmd()
	}
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
	m.loadingStartTime = time.Now()
	m.updateDetailsContent()
	return m, m.loadResourceAnalysis(node.CallData.MonitoringLog)
}
