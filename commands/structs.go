package commands

import (
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
	Metadata cromwell.MetadataResponse
}
