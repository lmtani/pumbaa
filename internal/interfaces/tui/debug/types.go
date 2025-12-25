// Package debug provides an interactive TUI for debugging Cromwell workflow metadata.
package debug

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow/metadata"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// Type aliases from new packages for backward compatibility
type (
	NodeType                  = tree.NodeType
	TreeNode                  = tree.TreeNode
	CallDetails               = metadata.CallDetails
	ExecutionEvent            = metadata.ExecutionEvent
	WorkflowPreemptionSummary = preemption.WorkflowSummary
	ProblematicTask           = preemption.ProblematicTask
	Failure                   = workflow.Failure
	WorkflowMetadata          = metadata.WorkflowMetadata
)

// NodeType constants
const (
	NodeTypeWorkflow    = tree.NodeTypeWorkflow
	NodeTypeCall        = tree.NodeTypeCall
	NodeTypeSubWorkflow = tree.NodeTypeSubWorkflow
	NodeTypeShard       = tree.NodeTypeShard
)

// ViewMode represents the current view mode of the TUI.
type ViewMode int

const (
	ViewModeTree ViewMode = iota
	ViewModeDetails
	ViewModeCommand
	ViewModeLogs
	ViewModeInputs
	ViewModeOutputs
	ViewModeHelp
	ViewModeMonitor
)

// PanelFocus represents which panel has focus.
type PanelFocus int

const (
	FocusTree PanelFocus = iota
	FocusDetails
)

// NodeViewState persists the view state for a specific node.
type NodeViewState struct {
	ViewMode   ViewMode
	LogCursor  int
	PanelFocus PanelFocus
}
