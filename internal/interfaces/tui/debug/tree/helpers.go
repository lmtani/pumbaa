// Package tree provides tree visualization logic for workflow debugging.
package tree

import (
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// AggregateStatus determines the overall status from multiple calls.
func AggregateStatus(calls []workflow.Call) string {
	if len(calls) == 0 {
		return "Unknown"
	}

	// Priority order: Failed > Running > Done
	hasRunning := false
	hasFailed := false

	for _, call := range calls {
		switch call.Status {
		case workflow.StatusFailed:
			hasFailed = true
		case workflow.StatusRunning, workflow.StatusSubmitted:
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
		if call.Status != workflow.StatusSucceeded && string(call.Status) != "Done" {
			allDone = false
			break
		}
	}
	if allDone {
		return "Done"
	}

	// Return status of most recent attempt
	return string(getMostRecentAttempt(calls).Status)
}

// EarliestStart returns the earliest start time from a list of calls.
func EarliestStart(calls []workflow.Call) time.Time {
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
func LatestEnd(calls []workflow.Call) time.Time {
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
