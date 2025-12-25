// Package metadata provides domain entities for detailed workflow metadata.
// These entities represent the complete metadata structure needed for debugging
// and analysis of Cromwell workflow executions.
package metadata

import (
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow/monitoring"
)

// WorkflowMetadata contains the full workflow metadata for debugging.
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
	Failures []workflow.Failure
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
	Stdout        string
	Stderr        string
	MonitoringLog string

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

	// Failures (task-level errors)
	Failures []workflow.Failure

	// SubWorkflow
	SubWorkflowID       string
	SubWorkflowMetadata *WorkflowMetadata

	// Cache for expensive calculations
	EfficiencyReport *monitoring.EfficiencyReport
}

// ExecutionEvent represents a single execution event in the timeline.
type ExecutionEvent struct {
	Description string
	Start       time.Time
	End         time.Time
}
