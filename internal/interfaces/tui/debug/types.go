// Package debug provides an interactive TUI for debugging Cromwell workflow metadata.
package debug

import (
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
)

// Type aliases from debuginfo package for backward compatibility
type (
	NodeType                  = debuginfo.NodeType
	TreeNode                  = debuginfo.TreeNode
	CallDetails               = debuginfo.CallDetails
	ExecutionEvent            = debuginfo.ExecutionEvent
	PreemptionStats           = debuginfo.PreemptionStats
	WorkflowPreemptionSummary = debuginfo.WorkflowPreemptionSummary
	ProblematicTask           = debuginfo.ProblematicTask
	Failure                   = debuginfo.Failure
	WorkflowMetadata          = debuginfo.WorkflowMetadata
)

// NodeType constants
const (
	NodeTypeWorkflow    = debuginfo.NodeTypeWorkflow
	NodeTypeCall        = debuginfo.NodeTypeCall
	NodeTypeSubWorkflow = debuginfo.NodeTypeSubWorkflow
	NodeTypeShard       = debuginfo.NodeTypeShard
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
