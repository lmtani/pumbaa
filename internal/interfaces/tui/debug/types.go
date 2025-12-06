// Package debug provides an interactive TUI for debugging Cromwell workflow metadata.
package debug

import (
	"time"
)

// NodeType represents the type of node in the call tree.
type NodeType int

const (
	NodeTypeWorkflow NodeType = iota
	NodeTypeCall
	NodeTypeSubWorkflow
	NodeTypeShard
)

// TreeNode represents a node in the workflow call tree.
type TreeNode struct {
	ID            string
	Name          string
	Type          NodeType
	Status        string
	Duration      time.Duration
	Start         time.Time
	End           time.Time
	Expanded      bool
	Children      []*TreeNode
	Parent        *TreeNode
	CallData      *CallDetails
	SubWorkflowID string
	Depth         int
}

// CallDetails contains detailed information about a call/task.
type CallDetails struct {
	// Identification
	Name       string
	ShardIndex int
	Attempt    int
	JobID      string

	// Status
	ExecutionStatus string
	BackendStatus   string
	ReturnCode      *int

	// Timing
	Start       time.Time
	End         time.Time
	VMStartTime time.Time
	VMEndTime   time.Time

	// Execution
	CommandLine string
	Backend     string
	CallRoot    string

	// Logs
	Stdout string
	Stderr string

	// Docker
	DockerImage     string
	DockerImageUsed string
	DockerSize      string

	// Resources
	CPU         string
	Memory      string
	Disk        string
	Preemptible string
	Zones       string

	// Cache
	CacheHit    bool
	CacheResult string

	// Cost
	VMCostPerHour float64

	// Inputs/Outputs
	Inputs  map[string]interface{}
	Outputs map[string]interface{}

	// Events
	ExecutionEvents []ExecutionEvent

	// Labels
	Labels map[string]string

	// SubWorkflow
	SubWorkflowID string
}

// ExecutionEvent represents a single execution event in the timeline.
type ExecutionEvent struct {
	Description string
	Start       time.Time
	End         time.Time
}

// WorkflowMetadata contains the full workflow metadata.
type WorkflowMetadata struct {
	// Basic info
	ID           string
	Name         string
	Status       string
	Start        time.Time
	End          time.Time
	WorkflowRoot string
	WorkflowLog  string

	// Submitted files
	SubmittedWorkflow string
	SubmittedInputs   string
	SubmittedOptions  string

	// Language
	WorkflowLanguage        string
	WorkflowLanguageVersion string

	// Calls (task name -> list of call attempts/shards)
	Calls map[string][]CallDetails

	// Outputs
	Outputs map[string]interface{}

	// Inputs
	Inputs map[string]interface{}

	// Labels
	Labels map[string]string
}

// ViewMode represents the current view mode of the TUI.
type ViewMode int

const (
	ViewModeTree ViewMode = iota
	ViewModeDetails
	ViewModeCommand
	ViewModeLogs
	ViewModeInputs
	ViewModeOutputs
	ViewModeTimeline
	ViewModeHelp
)

// PanelFocus represents which panel has focus.
type PanelFocus int

const (
	FocusTree PanelFocus = iota
	FocusDetails
)
