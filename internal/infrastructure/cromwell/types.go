// Package cromwell provides types for Cromwell API responses.
package cromwell

import "time"

// submitResponse represents the response from submitting a workflow.
type submitResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// statusResponse represents the response from getting workflow status.
type statusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// metadataResponse represents the response from getting workflow metadata.
type metadataResponse struct {
	ID           string                    `json:"id"`
	WorkflowName string                    `json:"workflowName"`
	Status       string                    `json:"status"`
	Start        time.Time                 `json:"start"`
	End          time.Time                 `json:"end"`
	Submission   time.Time                 `json:"submission"`
	Labels       map[string]string         `json:"labels"`
	Inputs       map[string]interface{}    `json:"inputs"`
	Outputs      map[string]interface{}    `json:"outputs"`
	Calls        map[string][]callMetadata `json:"calls"`
	Failures     []failureMetadata         `json:"failures"`

	// Detailed fields
	WorkflowRoot                  string          `json:"workflowRoot"`
	WorkflowLog                   string          `json:"workflowLog"`
	SubmittedFiles                *submittedFiles `json:"submittedFiles"`
	ActualWorkflowLanguage        string          `json:"actualWorkflowLanguage"`
	ActualWorkflowLanguageVersion string          `json:"actualWorkflowLanguageVersion"`
}

// submittedFiles contains the submitted workflow files.
type submittedFiles struct {
	Workflow string `json:"workflow"`
	Inputs   string `json:"inputs"`
	Options  string `json:"options"`
}

// callMetadata represents metadata for a single call.
type callMetadata struct {
	// Basic fields
	ExecutionStatus   string                 `json:"executionStatus"`
	Start             time.Time              `json:"start"`
	End               time.Time              `json:"end"`
	Attempt           int                    `json:"attempt"`
	ShardIndex        int                    `json:"shardIndex"`
	Backend           string                 `json:"backend"`
	ReturnCode        *int                   `json:"returnCode"`
	Stdout            string                 `json:"stdout"`
	Stderr            string                 `json:"stderr"`
	CommandLine       string                 `json:"commandLine"`
	Inputs            map[string]interface{} `json:"inputs"`
	Outputs           map[string]interface{} `json:"outputs"`
	RuntimeAttributes map[string]interface{} `json:"runtimeAttributes"`
	Failures          []failureMetadata      `json:"failures"`
	SubWorkflowID     string                 `json:"subWorkflowId"`

	// Detailed fields
	JobID                string               `json:"jobId"`
	BackendStatus        string               `json:"backendStatus"`
	VMStartTime          time.Time            `json:"vmStartTime"`
	VMEndTime            time.Time            `json:"vmEndTime"`
	CallRoot             string               `json:"callRoot"`
	MonitoringLog        string               `json:"monitoringLog"`
	DockerImageUsed      string               `json:"dockerImageUsed"`
	CompressedDockerSize interface{}          `json:"compressedDockerSize"`
	VMCostPerHour        float64              `json:"vmCostPerHour"`
	CallCaching          *callCachingInfo     `json:"callCaching"`
	Labels               map[string]string    `json:"labels"`
	ExecutionEvents      []executionEventMeta `json:"executionEvents"`
	SubWorkflowMetadata  *metadataResponse    `json:"subWorkflowMetadata"`
}

// callCachingInfo contains caching information for a call.
type callCachingInfo struct {
	Hit    bool   `json:"hit"`
	Result string `json:"result"`
}

// executionEventMeta represents an execution event in metadata.
type executionEventMeta struct {
	Description string `json:"description"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

// failureMetadata represents failure information.
type failureMetadata struct {
	Message  string            `json:"message"`
	CausedBy []failureMetadata `json:"causedBy"`
}

// queryResponse represents the response from querying workflows.
type queryResponse struct {
	Results           []queryResult `json:"results"`
	TotalResultsCount int           `json:"totalResultsCount"`
}

// queryResult represents a single result from a workflow query.
type queryResult struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Submission time.Time         `json:"submission"`
	Start      time.Time         `json:"start"`
	End        time.Time         `json:"end"`
	Labels     map[string]string `json:"labels"`
}

// outputsResponse represents the response from getting workflow outputs.
type outputsResponse struct {
	ID      string                 `json:"id"`
	Outputs map[string]interface{} `json:"outputs"`
}

// logsResponse represents the response from getting workflow logs.
type logsResponse struct {
	ID    string                `json:"id"`
	Calls map[string][]logEntry `json:"calls"`
}

// logEntry represents a single log entry.
type logEntry struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Attempt    int    `json:"attempt"`
	ShardIndex int    `json:"shardIndex"`
}

// costResponse represents the response from getting workflow cost.
type costResponse struct {
	ID       string  `json:"id"`
	Cost     float64 `json:"cost"`
	Status   string  `json:"status"`
	Currency string  `json:"currency"`
}

// healthStatusResponse represents the response from /engine/v1/status
type healthStatusResponse map[string]subsystemStatus

// subsystemStatus represents the status of a single subsystem
type subsystemStatus struct {
	OK      bool     `json:"ok"`
	Message []string `json:"messages,omitempty"`
}

// labelsResponse represents the response from getting workflow labels
type labelsResponse struct {
	ID     string            `json:"id"`
	Labels map[string]string `json:"labels"`
}
