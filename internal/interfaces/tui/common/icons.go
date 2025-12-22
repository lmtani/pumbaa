// Package common provides shared utilities and styles for the TUI.
package common

// Status icons - used for workflow states and operation results
const (
	IconSuccess = "✓"
	IconFailed  = "✗"
	IconRunning = "●"
	IconPending = "○"
	IconWarning = "!"
)

// Navigation icons - used for selection and tree expansion
const (
	IconSelected  = "▸"
	IconExpanded  = "▾"
	IconCollapsed = "▸"
	IconBack      = "◂"
)

// Node type icons - used in workflow tree visualization
const (
	IconWorkflow    = "◆"
	IconTask        = "◇"
	IconShard       = "·"
	IconSubworkflow = "◈"
)

// Data flow icons - used for inputs/outputs sections
const (
	IconInputs  = "→"
	IconOutputs = "←"
	IconLabels  = "#"
	IconOptions = "*"
)
