// Package debuginfo provides detailed workflow metadata parsing for debugging and analysis.
package debuginfo

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

	// Preemption stats (calculated from all attempts of this task)
	PreemptionStats *PreemptionStats

	// Inputs/Outputs
	Inputs  map[string]interface{}
	Outputs map[string]interface{}

	// Events
	ExecutionEvents []ExecutionEvent

	// Labels
	Labels map[string]string

	// SubWorkflow
	SubWorkflowID       string
	SubWorkflowMetadata *WorkflowMetadata
}

// ExecutionEvent represents a single execution event in the timeline.
type ExecutionEvent struct {
	Description string
	Start       time.Time
	End         time.Time
}

// PreemptionStats holds preemption efficiency data for a task.
type PreemptionStats struct {
	TotalAttempts   int     // Total number of attempts for this task/shard
	PreemptedCount  int     // Number of times preempted (attempts - 1)
	IsPreemptible   bool    // Whether the task was configured as preemptible
	EfficiencyScore float64 // 1.0 = first try success, lower = more preemptions
	MaxPreemptible  int     // Max preemptible attempts from config
}

// WorkflowPreemptionSummary holds aggregated preemption stats for a workflow.
type WorkflowPreemptionSummary struct {
	TotalTasks        int               // Total number of tasks/shards
	PreemptibleTasks  int               // Number of preemptible tasks
	TotalAttempts     int               // Total attempts across all tasks
	TotalPreemptions  int               // Total preemptions across all tasks
	OverallEfficiency float64           // Average efficiency (0-1)
	ProblematicTasks  []ProblematicTask // Tasks with low efficiency

	// Cost-weighted metrics
	TotalCost      float64 // Total cost of all attempts (resource-hours)
	WastedCost     float64 // Cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	CostUnit       string  // Unit for cost display (e.g., "resource-hours")
}

// ProblematicTask represents a task with poor preemption efficiency.
type ProblematicTask struct {
	Name            string
	ShardCount      int     // Number of shards (0 or 1 means non-scattered)
	Attempts        int     // Total attempts across all shards
	Preemptions     int     // Total preemptions across all shards
	EfficiencyScore float64 // Average efficiency across all shards

	// Cost-weighted metrics
	TotalCost      float64 // Total cost across all shards (resource-hours)
	WastedCost     float64 // Cost of failed attempts (resource-hours)
	CostEfficiency float64 // 1 - (WastedCost / TotalCost)
	ImpactPercent  float64 // WastedCost / WorkflowTotalWastedCost × 100
}

// Failure represents a workflow or task failure with its cause chain.
type Failure struct {
	Message  string
	CausedBy []Failure
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

	// Failures (workflow-level errors before calls execute)
	Failures []Failure
}
