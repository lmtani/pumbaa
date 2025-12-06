// Package workflow contains the domain entities and business logic for workflows.
package workflow

import (
	"time"
)

// Status represents the current state of a workflow execution.
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

// Workflow represents a WDL workflow execution in Cromwell.
type Workflow struct {
	ID          string
	Name        string
	Status      Status
	Start       time.Time
	End         time.Time
	Labels      map[string]string
	Inputs      map[string]interface{}
	Outputs     map[string]interface{}
	Failures    []Failure
	Calls       map[string][]Call
	SubmittedAt time.Time
}

// Failure represents a failure in workflow execution.
type Failure struct {
	Message  string
	CausedBy []Failure
}

// Call represents a task/call execution within a workflow.
type Call struct {
	Name              string
	Status            Status
	Start             time.Time
	End               time.Time
	Attempt           int
	ShardIndex        int
	Backend           string
	ReturnCode        *int
	Stdout            string
	Stderr            string
	CommandLine       string
	Inputs            map[string]interface{}
	Outputs           map[string]interface{}
	RuntimeAttributes map[string]interface{}
	Failures          []Failure
	SubWorkflowID     string
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
