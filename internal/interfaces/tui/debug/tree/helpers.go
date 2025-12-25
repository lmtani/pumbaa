// Package tree provides tree visualization logic for workflow debugging.
package tree

import (
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow/metadata"
)

// AggregateStatus determines the overall status from multiple calls.
func AggregateStatus(calls []metadata.CallDetails) string {
	if len(calls) == 0 {
		return "Unknown"
	}

	// Priority order: Failed > Running > Done
	hasRunning := false
	hasFailed := false

	for _, call := range calls {
		switch call.ExecutionStatus {
		case "Failed":
			hasFailed = true
		case "Running", "Starting":
			hasRunning = true
		}
	}

	if hasFailed {
		return "Failed"
	}
	if hasRunning {
		return "Running"
	}

	// Check if all succeeded
	allDone := true
	for _, call := range calls {
		if call.ExecutionStatus != "Done" {
			allDone = false
			break
		}
	}
	if allDone {
		return "Done"
	}

	// Return status of most recent attempt
	return getMostRecentAttempt(calls).ExecutionStatus
}

// EarliestStart returns the earliest start time from a list of calls.
func EarliestStart(calls []metadata.CallDetails) time.Time {
	if len(calls) == 0 {
		return time.Time{}
	}
	earliest := calls[0].Start
	for _, call := range calls[1:] {
		if call.Start.Before(earliest) {
			earliest = call.Start
		}
	}
	return earliest
}

// LatestEnd returns the latest end time from a list of calls.
func LatestEnd(calls []metadata.CallDetails) time.Time {
	if len(calls) == 0 {
		return time.Time{}
	}
	latest := calls[0].End
	for _, call := range calls[1:] {
		if call.End.After(latest) {
			latest = call.End
		}
	}
	return latest
}
