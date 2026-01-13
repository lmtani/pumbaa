// Package cloudlogging provides Cloud Logging adapter for batch logs.
package cloudlogging

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// CloudLoggingRepository implements BatchLogsRepository using Google Cloud Logging.
type CloudLoggingRepository struct{}

// NewCloudLoggingRepository creates a new Cloud Logging adapter.
func NewCloudLoggingRepository() *CloudLoggingRepository {
	return &CloudLoggingRepository{}
}

// GetLogs retrieves batch logs from Cloud Logging API.
// Uses Application Default Credentials (ADC) for authentication.
// projectID is extracted from the jobName resource.
func (r *CloudLoggingRepository) GetLogs(
	ctx context.Context,
	jobName string,
	filter ports.BatchLogsFilter,
) ([]ports.BatchLogEntry, error) {
	// Parse project, location, and job ID from resource name
	project, _, jobID, err := parseResourceName(jobName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job name: %w", err)
	}

	// Create logadmin client
	client, err := logadmin.NewClient(ctx, project)
	if err != nil {
		return nil, ports.NewBatchLogsError("fetch", jobName, "failed to create Cloud Logging client", err)
	}
	defer client.Close()

	// Build the logs filter
	logsFilter := buildLogsFilter(project, jobID, filter)

	// Query logs
	it := client.Entries(ctx, logadmin.Filter(logsFilter))

	var entries []ports.BatchLogEntry
	count := 0

	// Iterate through log entries
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.PermissionDenied {
				return nil, ports.ErrUnauthorized
			}
			return nil, ports.NewBatchLogsError("fetch", jobName, "error iterating logs", err)
		}

		// Format entry
		formatted := FormatLogEntry(entry, 500) // Max 500 chars per message
		entries = append(entries, formatted)

		count++
		if count >= filter.Limit {
			break
		}
	}

	if len(entries) == 0 {
		return nil, ports.ErrLogsNotFound
	}

	return entries, nil
}

// parseResourceName extracts project, location, and jobID from resource name.
// Expected format: projects/{project}/locations/{location}/jobs/{jobId}
func parseResourceName(jobName string) (project, location, jobID string, err error) {
	var proj, loc, job string

	// Simple parsing using string operations
	parts := splitPath(jobName, "/")
	if len(parts) < 6 {
		return "", "", "", fmt.Errorf("invalid resource name format")
	}

	// parts: [projects, {proj}, locations, {loc}, jobs, {job}]
	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" {
		return "", "", "", fmt.Errorf("invalid resource name structure")
	}

	proj = parts[1]
	loc = parts[3]
	job = parts[5]

	if proj == "" || loc == "" || job == "" {
		return "", "", "", fmt.Errorf("missing required fields in resource name")
	}

	return proj, loc, job, nil
}

// splitPath splits a path string by delimiter.
func splitPath(s, sep string) []string {
	var result []string
	current := ""
	for _, r := range s {
		if r == rune(sep[0]) {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// buildLogsFilter constructs a Cloud Logging filter for batch job logs.
// Uses the same filter format as Cloud Logging API:
//
//	resource.type="batch.googleapis.com/Job"
//	AND (logName="projects/{project}/logs/batch_task_logs" OR logName="projects/{project}/logs/batch_agent_logs")
//	AND labels.task_group_name=~"jobs/{jobID}"
//	AND severity>={minSeverity}
func buildLogsFilter(project, jobID string, filter ports.BatchLogsFilter) string {
	// Base filter: resource type + log names + task group name regex
	// task_group_name contains projects/.../jobs/<jobID>/taskGroups/... so we match with regex
	baseFilter := fmt.Sprintf(
		`resource.type="batch.googleapis.com/Job"
AND (logName="projects/%s/logs/batch_task_logs" OR logName="projects/%s/logs/batch_agent_logs")
AND labels.task_group_name=~"jobs/%s"`,
		project, project, jobID,
	)

	// Add severity filter if specified
	if filter.MinSeverity != "" {
		baseFilter += fmt.Sprintf(`
AND severity>=%s`, filter.MinSeverity)
	}

	// Add time range filters if specified
	if !filter.StartTime.IsZero() {
		baseFilter += fmt.Sprintf(`
AND timestamp>="%s"`, filter.StartTime.Format("2006-01-02T15:04:05Z"))
	} else {
		// If no start time specified, use a default far in the past to ensure coverage
		baseFilter += `
AND timestamp>="1970-01-01T00:00:00Z"`
	}

	if !filter.EndTime.IsZero() {
		baseFilter += fmt.Sprintf(`
AND timestamp<="%s"`, filter.EndTime.Format("2006-01-02T15:04:05Z"))
	}

	return baseFilter
}
