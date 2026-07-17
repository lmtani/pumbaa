package cromwell

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// fetchExpandedWorkflow loads the fully expanded metadata (subworkflows
// included) so summaries cover the whole run, not just the top level.
func fetchExpandedWorkflow(ctx context.Context, fetcher ports.WorkflowMetadataFetcher, workflowID string) (*workflow.Workflow, error) {
	data, err := fetcher.GetRawMetadataWithOptions(ctx, workflowID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %v", err)
	}
	wf, err := fetcher.ParseMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %v", err)
	}
	return wf, nil
}
