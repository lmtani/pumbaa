// Package cromwell provides an implementation of the workflow repository using the Cromwell API.
package cromwell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// Client implements workflow.Repository for Cromwell.
type Client struct {
	BaseURL    string
	httpClient *http.Client
}

// Config holds configuration for the Cromwell client.
type Config struct {
	Host    string
	Timeout time.Duration
}

// NewClient creates a new Cromwell client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		BaseURL: cfg.Host,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Submit submits a new workflow to Cromwell.
func (c *Client) Submit(ctx context.Context, req workflow.SubmitRequest) (*workflow.SubmitResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add workflow source
	if err := c.addFileField(writer, "workflowSource", "workflow.wdl", req.WorkflowSource); err != nil {
		return nil, err
	}

	// Add optional fields
	if len(req.WorkflowInputs) > 0 {
		if err := c.addFileField(writer, "workflowInputs", "inputs.json", req.WorkflowInputs); err != nil {
			return nil, err
		}
	}

	if len(req.WorkflowOptions) > 0 {
		if err := c.addFileField(writer, "workflowOptions", "options.json", req.WorkflowOptions); err != nil {
			return nil, err
		}
	}

	if len(req.WorkflowDependencies) > 0 {
		if err := c.addFileField(writer, "workflowDependencies", "dependencies.zip", req.WorkflowDependencies); err != nil {
			return nil, err
		}
	}

	// Add workflow type
	if req.WorkflowType != "" {
		if err := writer.WriteField("workflowType", req.WorkflowType); err != nil {
			return nil, err
		}
	}

	if req.WorkflowTypeVersion != "" {
		if err := writer.WriteField("workflowTypeVersion", req.WorkflowTypeVersion); err != nil {
			return nil, err
		}
	}

	// Add labels
	if len(req.Labels) > 0 {
		labelsJSON, err := json.Marshal(req.Labels)
		if err != nil {
			return nil, err
		}
		if err := writer.WriteField("labels", string(labelsJSON)); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/workflows/v1", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result submitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &workflow.SubmitResponse{
		ID:     result.ID,
		Status: workflow.Status(result.Status),
	}, nil
}

// GetMetadata retrieves detailed metadata for a workflow.
func (c *Client) GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/metadata", c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result metadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return c.mapMetadataToWorkflow(&result), nil
}

// GetStatus retrieves the status of a workflow.
func (c *Client) GetStatus(ctx context.Context, workflowID string) (workflow.Status, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/status", c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return workflow.StatusUnknown, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return workflow.StatusUnknown, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return workflow.StatusUnknown, workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return workflow.StatusUnknown, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return workflow.StatusUnknown, err
	}

	return workflow.Status(result.Status), nil
}

// Abort aborts a running workflow.
func (c *Client) Abort(ctx context.Context, workflowID string) error {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/abort", c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	return nil
}

// Query queries workflows based on filters.
func (c *Client) Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/query", c.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	q := httpReq.URL.Query()

	// Exclude subworkflows by default
	q.Add("includeSubworkflows", "false")

	// Include labels in results
	q.Add("additionalQueryResultFields", "labels")

	if filter.Name != "" {
		q.Add("name", filter.Name)
	}
	for _, status := range filter.Status {
		q.Add("status", string(status))
	}
	for k, v := range filter.Labels {
		q.Add("label", fmt.Sprintf("%s:%s", k, v))
	}
	if filter.PageSize > 0 {
		q.Add("pageSize", fmt.Sprintf("%d", filter.PageSize))
	}
	httpReq.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	workflows := make([]workflow.Workflow, 0, len(result.Results))
	for _, r := range result.Results {
		wf := workflow.Workflow{
			ID:          r.ID,
			Name:        r.Name,
			Status:      workflow.Status(r.Status),
			SubmittedAt: r.Submission,
			Start:       r.Start,
			End:         r.End,
			Labels:      r.Labels,
		}
		workflows = append(workflows, wf)
	}

	return &workflow.QueryResult{
		Workflows:  workflows,
		TotalCount: result.TotalResultsCount,
	}, nil
}

// GetOutputs retrieves the outputs of a completed workflow.
func (c *Client) GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/outputs", c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result outputsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Outputs, nil
}

// GetLogs retrieves the logs for a workflow.
func (c *Client) GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/logs", c.BaseURL, workflowID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var result logsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	logs := make(map[string][]workflow.CallLog)
	for callName, callLogs := range result.Calls {
		logs[callName] = make([]workflow.CallLog, 0, len(callLogs))
		for _, l := range callLogs {
			logs[callName] = append(logs[callName], workflow.CallLog{
				Stdout:     l.Stdout,
				Stderr:     l.Stderr,
				Attempt:    l.Attempt,
				ShardIndex: l.ShardIndex,
			})
		}
	}

	return logs, nil
}

// GetRawMetadata retrieves the raw JSON metadata for a workflow.
func (c *Client) GetRawMetadata(ctx context.Context, workflowID string) ([]byte, error) {
	return c.GetRawMetadataWithOptions(ctx, workflowID, false)
}

// GetRawMetadataWithOptions retrieves the raw JSON metadata for a workflow with options.
func (c *Client) GetRawMetadataWithOptions(ctx context.Context, workflowID string, expandSubWorkflows bool) ([]byte, error) {
	url := fmt.Sprintf("%s/api/workflows/v1/%s/metadata", c.BaseURL, workflowID)
	if expandSubWorkflows {
		url += "?expandSubWorkflows=true"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", workflow.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, workflow.ErrWorkflowNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, workflow.APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	return io.ReadAll(resp.Body)
}

// addFileField adds a file field to a multipart form.
func (c *Client) addFileField(writer *multipart.Writer, fieldName, fileName string, data []byte) error {
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}
