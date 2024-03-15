package types

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
	SubmittedFiles SubmittedFiles
	RootWorkflowID string
	Calls          CallItemSet
	Inputs         map[string]interface{}
	Outputs        map[string]interface{}
	Start          time.Time
	End            time.Time
	Status         string
	Failures       []Failure
}

type SubmittedFiles struct {
	Options string
}

type Failure struct {
	CausedBy []Failure
	Message  string
}

type BackendLogs struct {
	Log string
}

type CallItemSet map[string][]CallItem

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
	BackendLogs         BackendLogs
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

type TotalResources struct {
	PreemptHdd    float64
	PreemptSsd    float64
	PreemptCPU    float64
	PreemptMemory float64
	Hdd           float64
	Ssd           float64
	CPU           float64
	Memory        float64
	CachedCalls   int
	TotalTime     time.Duration
}

type ParsedCallAttributes struct {
	Hdd      float64
	Preempt  bool
	Ssd      float64
	Memory   float64
	CPU      float64
	Elapsed  time.Duration
	HitCache bool
}

type ParamsMetadataGet struct {
	IncludeKey         []string `url:"includeKey,omitempty"`
	ExcludeKey         []string `url:"excludeKey,omitempty"`
	ExpandSubWorkflows bool     `url:"expandSubWorkflows,omitempty"`
}

type ParamsQueryGet struct {
	Submission          time.Time `url:"submission,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	Start               time.Time `url:"start,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	End                 time.Time `url:"end,omitempty" layout:"2006-01-02T15:04:05.000Z"`
	Status              string    `url:"status,omitempty"`
	Name                string    `url:"name,omitempty"`
	Id                  string    `url:"id,omitempty"`
	IncludeSubworkflows bool      `url:"includeSubworkflows,omitempty"`
}
