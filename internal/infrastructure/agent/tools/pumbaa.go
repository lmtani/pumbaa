// Package tools provides Cromwell and GCS tools for the agent.
package tools

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// CromwellRepository defines the interface for Cromwell operations needed by tools.
type CromwellRepository interface {
	Query(ctx context.Context, filter workflow.QueryFilter) (*workflow.QueryResult, error)
	GetStatus(ctx context.Context, workflowID string) (workflow.Status, error)
	GetMetadata(ctx context.Context, workflowID string) (*workflow.Workflow, error)
	GetOutputs(ctx context.Context, workflowID string) (map[string]interface{}, error)
	GetLogs(ctx context.Context, workflowID string) (map[string][]workflow.CallLog, error)
}

const MaxGCSFileSize int64 = 5 * 1024 * 1024 // 5 MB

// PumbaaInput represents the input for the unified Pumbaa tool.
type PumbaaInput struct {
	// Action is the operation to perform
	// Cromwell actions: "query", "status", "metadata", "outputs", "logs"
	// GCS actions: "gcs_download"
	Action string `json:"action"`

	// WorkflowID is the UUID of the workflow (for Cromwell actions except query)
	WorkflowID string `json:"workflow_id,omitempty"`

	// Status filter for query action (e.g., "Running", "Succeeded", "Failed")
	Status string `json:"status,omitempty"`

	// Name filter for query action
	Name string `json:"name,omitempty"`

	// Path for gcs_download action (gs://bucket/path)
	Path string `json:"path,omitempty"`
}

// PumbaaOutput represents the output of the Pumbaa tool.
type PumbaaOutput struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Action  string      `json:"action"`
	Data    interface{} `json:"data,omitempty"`
}

// GetPumbaaTool returns a single unified tool that handles all operations.
// This avoids Vertex AI limitation: "Multiple tools are supported only when they are all search tools"
func GetPumbaaTool(repo CromwellRepository) tool.Tool {
	t, err := functiontool.New(
		functiontool.Config{
			Name: "pumbaa",
			Description: `Unified tool for Cromwell workflow management and GCS file access.

Available actions:
- "query": Search Cromwell workflows. Optional: status (Running, Succeeded, Failed, Submitted, Aborted), name.
- "status": Get workflow status. Required: workflow_id.
- "metadata": Get workflow metadata (calls, inputs, outputs). Required: workflow_id.
- "outputs": Get workflow output files. Required: workflow_id.
- "logs": Get log file paths for debugging. Required: workflow_id.
- "gcs_download": Read file from Google Cloud Storage. Required: path (gs://bucket/file).`,
		},
		func(ctx tool.Context, input PumbaaInput) (PumbaaOutput, error) {
			// Note: Don't use log.Printf here as it interferes with TUI

			bgCtx := context.Background()

			switch input.Action {
			case "query":
				return handleQuery(bgCtx, repo, input)
			case "status":
				return handleStatus(bgCtx, repo, input)
			case "metadata":
				return handleMetadata(bgCtx, repo, input)
			case "outputs":
				return handleOutputs(bgCtx, repo, input)
			case "logs":
				return handleLogs(bgCtx, repo, input)
			case "gcs_download":
				return handleGCSDownload(bgCtx, input)
			default:
				return PumbaaOutput{
					Success: false,
					Action:  input.Action,
					Error:   fmt.Sprintf("unknown action: %s. Valid: query, status, metadata, outputs, logs, gcs_download", input.Action),
				}, nil
			}
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create pumbaa tool: %v", err))
	}
	return t
}

// ============================================================================
// Cromwell Handlers
// ============================================================================

func handleQuery(ctx context.Context, repo CromwellRepository, input PumbaaInput) (PumbaaOutput, error) {
	filter := workflow.QueryFilter{
		Name:     input.Name,
		PageSize: 100, // Limit to 100 records
	}
	if input.Status != "" {
		filter.Status = []workflow.Status{workflow.Status(input.Status)}
	}

	result, err := repo.Query(ctx, filter)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "query", Error: err.Error()}, nil
	}

	workflows := make([]map[string]interface{}, 0, len(result.Workflows))
	for _, wf := range result.Workflows {
		workflows = append(workflows, map[string]interface{}{
			"id":           wf.ID,
			"name":         wf.Name,
			"status":       string(wf.Status),
			"submitted_at": wf.SubmittedAt,
			"start":        wf.Start,
			"end":          wf.End,
			"labels":       wf.Labels,
		})
	}

	return PumbaaOutput{
		Success: true,
		Action:  "query",
		Data:    map[string]interface{}{"total": result.TotalCount, "workflows": workflows},
	}, nil
}

func handleStatus(ctx context.Context, repo CromwellRepository, input PumbaaInput) (PumbaaOutput, error) {
	if input.WorkflowID == "" {
		return PumbaaOutput{Success: false, Action: "status", Error: "workflow_id is required"}, nil
	}

	status, err := repo.GetStatus(ctx, input.WorkflowID)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "status", Error: err.Error()}, nil
	}

	return PumbaaOutput{
		Success: true,
		Action:  "status",
		Data:    map[string]interface{}{"id": input.WorkflowID, "status": string(status)},
	}, nil
}

func handleMetadata(ctx context.Context, repo CromwellRepository, input PumbaaInput) (PumbaaOutput, error) {
	if input.WorkflowID == "" {
		return PumbaaOutput{Success: false, Action: "metadata", Error: "workflow_id is required"}, nil
	}

	wf, err := repo.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "metadata", Error: err.Error()}, nil
	}

	calls := make(map[string][]map[string]interface{})
	for callName, callInstances := range wf.Calls {
		instances := make([]map[string]interface{}, 0, len(callInstances))
		for _, call := range callInstances {
			instances = append(instances, map[string]interface{}{
				"name": call.Name, "status": string(call.Status),
				"start": call.Start, "end": call.End,
				"attempt": call.Attempt, "shard_index": call.ShardIndex,
			})
		}
		calls[callName] = instances
	}

	return PumbaaOutput{
		Success: true,
		Action:  "metadata",
		Data: map[string]interface{}{
			"id": wf.ID, "name": wf.Name, "status": string(wf.Status),
			"submitted_at": wf.SubmittedAt, "start": wf.Start, "end": wf.End,
			"inputs": wf.Inputs, "outputs": wf.Outputs, "calls": calls, "labels": wf.Labels,
		},
	}, nil
}

func handleOutputs(ctx context.Context, repo CromwellRepository, input PumbaaInput) (PumbaaOutput, error) {
	if input.WorkflowID == "" {
		return PumbaaOutput{Success: false, Action: "outputs", Error: "workflow_id is required"}, nil
	}

	outputs, err := repo.GetOutputs(ctx, input.WorkflowID)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "outputs", Error: err.Error()}, nil
	}

	return PumbaaOutput{
		Success: true,
		Action:  "outputs",
		Data:    map[string]interface{}{"id": input.WorkflowID, "outputs": outputs},
	}, nil
}

func handleLogs(ctx context.Context, repo CromwellRepository, input PumbaaInput) (PumbaaOutput, error) {
	if input.WorkflowID == "" {
		return PumbaaOutput{Success: false, Action: "logs", Error: "workflow_id is required"}, nil
	}

	logs, err := repo.GetLogs(ctx, input.WorkflowID)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "logs", Error: err.Error()}, nil
	}

	callLogs := make(map[string]interface{})
	for callName, logEntries := range logs {
		entries := make([]map[string]interface{}, 0, len(logEntries))
		for _, entry := range logEntries {
			entries = append(entries, map[string]interface{}{
				"stdout": entry.Stdout, "stderr": entry.Stderr,
				"attempt": entry.Attempt, "shard_index": entry.ShardIndex,
			})
		}
		callLogs[callName] = entries
	}

	return PumbaaOutput{
		Success: true,
		Action:  "logs",
		Data:    map[string]interface{}{"id": input.WorkflowID, "calls": callLogs},
	}, nil
}

// ============================================================================
// GCS Handler
// ============================================================================

func handleGCSDownload(ctx context.Context, input PumbaaInput) (PumbaaOutput, error) {
	if input.Path == "" {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: "path is required (e.g., gs://bucket/file)"}, nil
	}

	if !strings.HasPrefix(input.Path, "gs://") {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: "path must start with gs://"}, nil
	}

	path := strings.TrimPrefix(input.Path, "gs://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: "invalid path format, expected gs://bucket/object"}, nil
	}
	bucket, object := parts[0], parts[1]

	client, err := storage.NewClient(ctx)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("failed to create GCS client: %v", err)}, nil
	}
	defer client.Close()

	obj := client.Bucket(bucket).Object(object)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("object not found: %s", input.Path)}, nil
		}
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("failed to get attrs: %v", err)}, nil
	}

	if attrs.Size > MaxGCSFileSize {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("file too large: %d bytes (max 5MB)", attrs.Size)}, nil
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("failed to read: %v", err)}, nil
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return PumbaaOutput{Success: false, Action: "gcs_download", Error: fmt.Sprintf("failed to read content: %v", err)}, nil
	}

	return PumbaaOutput{
		Success: true,
		Action:  "gcs_download",
		Data: map[string]interface{}{
			"bucket":       bucket,
			"object":       object,
			"size":         attrs.Size,
			"content_type": attrs.ContentType,
			"content":      string(content),
		},
	}, nil
}
