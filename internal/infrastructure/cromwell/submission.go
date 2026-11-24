package cromwell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// Compile-time check: the client serves submissions to the export use case.
var _ ports.SubmissionReader = (*Client)(nil)

// GetSubmission returns the files a run was submitted with, fetched with
// includeKey so the server sends the submission section and not the calls:
// the full metadata of a large run is megabytes, of which this is a few
// kilobytes.
func (c *Client) GetSubmission(ctx context.Context, workflowID string) (*workflow.Submission, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/metadata?includeKey=submittedFiles&includeKey=labels&includeKey=workflowName&includeKey=submission&includeKey=id",
		c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", workflow.ErrWorkflowNotFound, workflowID)
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get submission: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ID             string            `json:"id"`
		WorkflowName   string            `json:"workflowName"`
		Labels         map[string]string `json:"labels"`
		Submission     time.Time         `json:"submission"`
		SubmittedFiles struct {
			Workflow    string `json:"workflow"`
			WorkflowURL string `json:"workflowUrl"`
			Inputs      string `json:"inputs"`
			Options     string `json:"options"`
		} `json:"submittedFiles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode submission: %w", err)
	}

	return &workflow.Submission{
		ID:          result.ID,
		Name:        result.WorkflowName,
		Labels:      result.Labels,
		SubmittedAt: result.Submission,
		Workflow:    result.SubmittedFiles.Workflow,
		WorkflowURL: result.SubmittedFiles.WorkflowURL,
		Inputs:      result.SubmittedFiles.Inputs,
		Options:     result.SubmittedFiles.Options,
	}, nil
}
