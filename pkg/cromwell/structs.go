package cromwell

import "time"

type ErrorResponse struct {
	HTTPStatus string
	Status     string
	Message    string
}

type SubmitResponse struct {
	ID     string
	Status string
}

type OutputsResponse struct {
	ID      string
	Outputs map[string]interface{}
}

type QueryResponse struct {
	Results           []QueryResponseWorkflow
	TotalResultsCount int
}

type QueryResponseWorkflow struct {
	ID                    string
	Name                  string
	Status                string
	Submission            string
	Start                 time.Time
	End                   time.Time
	MetadataArchiveStatus string
}

type MetadataResponse struct {
	WorkflowName   string
	RootWorkflowID string
	Calls          map[string][]CallItem
	Inputs         map[string]interface{}
	Outputs        map[string]interface{}
	Start          time.Time
	End            time.Time
	Status         string
}

type CallItem struct {
	ExecutionStatus     string
	Stdout              string
	Stderr              string
	Attempt             int
	ShardIndex          int
	Start               time.Time
	End                 time.Time
	Labels              Label
	MonitoringLog       string
	CommandLine         string
	DockerImageUsed     string
	SubWorkflowID       string
	SubWorkflowMetadata MetadataResponse
	RuntimeAttributes   RuntimeAttributes
	CallCaching         CallCachingData
	ExecutionEvents     []ExecutionEvents
}

type ExecutionEvents struct {
	StartTime   time.Time
	Description string
	EndTime     time.Time
}

type RuntimeAttributes struct {
	BootDiskSizeGb string
	CPU            string
	Disks          string
	Docker         string
	Memory         string
	Preemptible    string
}

type CallCachingData struct {
	Result string
	Hit    bool
}

type Label struct {
	CromwellWorkflowID string `json:"cromwell-workflow-id"`
	WdlTaskName        string `json:"wdl-task-name"`
}

type SubmitRequest struct {
	WorkflowSource       string
	WorkflowInputs       string
	WorkflowDependencies string
	WorkflowOptions      string
}
