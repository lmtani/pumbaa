package commands

import (
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

type ResourceTableResponse struct {
	Total TotalResources
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

type QueryTableResponse struct {
	Results           []cromwell.QueryResponseWorkflow
	TotalResultsCount int
}

type MetadataTableResponse struct {
	WorkflowName   string
	RootWorkflowID string
	Calls          map[string][]cromwell.CallItem
	Inputs         map[string]interface{}
	Outputs        map[string]interface{}
	Start          time.Time
	End            time.Time
	Status         string
}
