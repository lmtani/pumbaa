// Package workflow contains the domain entities and business logic for workflows.
// This package follows Domain-Driven Design (DDD) principles:
//   - Aggregate Root: Workflow (main entry point with identity)
//   - Entity: Call (has identity within the aggregate)
//   - Value Objects: Status, Failure, ExecutionEvent, PreemptionSummary, EfficiencyReport
package workflow

import (
	"time"
)

// Status is a Value Object representing the current state of a workflow execution.
// Value Objects are immutable and compared by value, not identity.
type Status string

const (
	StatusSubmitted Status = "Submitted"
	StatusRunning   Status = "Running"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusAborted   Status = "Aborted"
	StatusAborting  Status = "Aborting"
	StatusOnHold    Status = "On Hold"
	StatusUnknown   Status = "Unknown"
)

// Workflow is the Aggregate Root for the workflow domain.
// It encapsulates the entire workflow execution lifecycle and provides
// domain behavior through methods like CalculatePreemptionSummary().
type Workflow struct {
	// Basic info
	ID          string
	Name        string
	Status      Status
	Start       time.Time
	End         time.Time
	SubmittedAt time.Time

	// Labels, Inputs, Outputs
	Labels  map[string]string
	Inputs  map[string]interface{}
	Outputs map[string]interface{}

	// Calls and Failures
	Calls    map[string][]Call
	Failures []Failure

	// Debug/detailed fields
	WorkflowRoot            string
	WorkflowLog             string
	SubmittedWorkflow       string
	SubmittedInputs         string
	SubmittedOptions        string
	WorkflowLanguage        string
	WorkflowLanguageVersion string
}

// Failure is a Value Object representing an error that occurred during execution.
// It forms a tree structure via CausedBy for nested failures.
type Failure struct {
	Message  string
	CausedBy []Failure
}

// Call is an Entity representing a task execution within a workflow.
// It has identity (Name + ShardIndex + Attempt) within the Workflow aggregate.
type Call struct {
	// Identification
	Name       string
	ShardIndex int
	Attempt    int
	JobID      string

	// Status
	Status        Status
	BackendStatus string
	ReturnCode    *int

	// Timing
	Start       time.Time
	End         time.Time
	VMStartTime time.Time
	VMEndTime   time.Time

	// Execution
	Backend           string
	CommandLine       string
	CallRoot          string
	RuntimeAttributes map[string]interface{}

	// Logs
	Stdout        string
	Stderr        string
	MonitoringLog string

	// Docker
	DockerImage     string
	DockerImageUsed string
	DockerSize      string

	// Resources (parsed from RuntimeAttributes for convenience)
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

	// Failures (task-level errors)
	Failures []Failure

	// SubWorkflow
	SubWorkflowID       string
	SubWorkflowMetadata *Workflow

	// Cache for expensive calculations
	EfficiencyReport *EfficiencyReport
}

// ExecutionEvent represents a single execution event in the timeline.
type ExecutionEvent struct {
	Description string
	Start       time.Time
	End         time.Time
}

// IsTerminal returns true if the workflow is in a terminal state.
func (w *Workflow) IsTerminal() bool {
	switch w.Status {
	case StatusSucceeded, StatusFailed, StatusAborted:
		return true
	default:
		return false
	}
}

// Duration returns the duration of the workflow execution.
func (w *Workflow) Duration() time.Duration {
	if w.Start.IsZero() {
		return 0
	}
	end := w.End
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(w.Start)
}

// SubmitRequest represents a request to submit a new workflow.
type SubmitRequest struct {
	WorkflowSource       []byte
	WorkflowInputs       []byte
	WorkflowOptions      []byte
	WorkflowDependencies []byte
	Labels               map[string]string
	WorkflowType         string
	WorkflowTypeVersion  string
}

// SubmitResponse represents the response from submitting a workflow.
type SubmitResponse struct {
	ID     string
	Status Status
}

// QueryFilter represents filters for querying workflows.
type QueryFilter struct {
	Name          string
	Status        []Status
	SubmissionMin time.Time
	SubmissionMax time.Time
	StartMin      time.Time
	StartMax      time.Time
	EndMin        time.Time
	EndMax        time.Time
	Labels        map[string]string
	Page          int
	PageSize      int
}

// QueryResult represents the result of querying workflows.
type QueryResult struct {
	Workflows  []Workflow
	TotalCount int
}
