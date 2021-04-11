package commands

import (
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

type ResourceTableResponse struct {
	Total cromwell.TotalResources
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
