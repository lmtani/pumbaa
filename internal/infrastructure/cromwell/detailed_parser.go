// Package cromwell provides detailed metadata parsing for Cromwell workflows.
package cromwell

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow/metadata"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
)

// ParseDetailedMetadata parses raw JSON metadata into domain WorkflowMetadata.
// This is the detailed version used for debugging and analysis.
func ParseDetailedMetadata(data []byte) (*metadata.WorkflowMetadata, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	wm := &metadata.WorkflowMetadata{
		Calls:   make(map[string][]metadata.CallDetails),
		Outputs: make(map[string]interface{}),
		Inputs:  make(map[string]interface{}),
		Labels:  make(map[string]string),
	}

	// Basic fields
	wm.ID = getString(raw, "id")
	wm.Name = getString(raw, "workflowName")
	wm.Status = getString(raw, "status")
	wm.WorkflowRoot = getString(raw, "workflowRoot")
	wm.WorkflowLog = getString(raw, "workflowLog")
	wm.WorkflowLanguage = getString(raw, "actualWorkflowLanguage")
	wm.WorkflowLanguageVersion = getString(raw, "actualWorkflowLanguageVersion")

	// Parse timestamps
	wm.Start = parseTime(getString(raw, "start"))
	wm.End = parseTime(getString(raw, "end"))

	// Parse submitted files
	if sf, ok := raw["submittedFiles"].(map[string]interface{}); ok {
		wm.SubmittedWorkflow = getString(sf, "workflow")
		wm.SubmittedInputs = getString(sf, "inputs")
		wm.SubmittedOptions = getString(sf, "options")
	}

	// Parse outputs
	if outputs, ok := raw["outputs"].(map[string]interface{}); ok {
		wm.Outputs = outputs
	}

	// Parse inputs
	if inputs, ok := raw["inputs"].(map[string]interface{}); ok {
		wm.Inputs = inputs
	}

	// Parse labels
	if labels, ok := raw["labels"].(map[string]interface{}); ok {
		for k, v := range labels {
			if s, ok := v.(string); ok {
				wm.Labels[k] = s
			}
		}
	}

	// Parse failures (workflow-level errors)
	if failures, ok := raw["failures"].([]interface{}); ok {
		wm.Failures = parseDetailedFailures(failures)
	}

	// Parse calls
	if calls, ok := raw["calls"].(map[string]interface{}); ok {
		for callName, callList := range calls {
			if list, ok := callList.([]interface{}); ok {
				var details []metadata.CallDetails
				for _, item := range list {
					if callMap, ok := item.(map[string]interface{}); ok {
						cd := parseDetailedCallDetails(callMap)
						// Parse embedded subworkflow metadata if present
						if subMeta, ok := callMap["subWorkflowMetadata"].(map[string]interface{}); ok {
							subWM := parseSubWorkflowMetadata(subMeta)
							cd.SubWorkflowMetadata = subWM
							cd.SubWorkflowID = subWM.ID
						}
						details = append(details, cd)
					}
				}
				wm.Calls[callName] = details
			}
		}
	}

	return wm, nil
}

// parseDetailedFailures parses the failures array from Cromwell metadata.
func parseDetailedFailures(failures []interface{}) []workflow.Failure {
	var result []workflow.Failure
	for _, f := range failures {
		if fMap, ok := f.(map[string]interface{}); ok {
			failure := workflow.Failure{
				Message: getString(fMap, "message"),
			}
			if causedBy, ok := fMap["causedBy"].([]interface{}); ok {
				failure.CausedBy = parseDetailedFailures(causedBy)
			}
			result = append(result, failure)
		}
	}
	return result
}

// parseSubWorkflowMetadata parses embedded subworkflow metadata.
func parseSubWorkflowMetadata(raw map[string]interface{}) *metadata.WorkflowMetadata {
	wm := &metadata.WorkflowMetadata{
		Calls:   make(map[string][]metadata.CallDetails),
		Outputs: make(map[string]interface{}),
		Inputs:  make(map[string]interface{}),
		Labels:  make(map[string]string),
	}

	// Basic fields
	wm.ID = getString(raw, "id")
	wm.Name = getString(raw, "workflowName")
	wm.Status = getString(raw, "status")
	wm.WorkflowRoot = getString(raw, "workflowRoot")

	// Parse timestamps
	wm.Start = parseTime(getString(raw, "start"))
	wm.End = parseTime(getString(raw, "end"))

	// Parse outputs
	if outputs, ok := raw["outputs"].(map[string]interface{}); ok {
		wm.Outputs = outputs
	}

	// Parse inputs
	if inputs, ok := raw["inputs"].(map[string]interface{}); ok {
		wm.Inputs = inputs
	}

	// Parse calls recursively
	if calls, ok := raw["calls"].(map[string]interface{}); ok {
		for callName, callList := range calls {
			if list, ok := callList.([]interface{}); ok {
				var details []metadata.CallDetails
				for _, item := range list {
					if callMap, ok := item.(map[string]interface{}); ok {
						cd := parseDetailedCallDetails(callMap)
						// Recursively parse nested subworkflow metadata
						if subMeta, ok := callMap["subWorkflowMetadata"].(map[string]interface{}); ok {
							subWM := parseSubWorkflowMetadata(subMeta)
							cd.SubWorkflowMetadata = subWM
							cd.SubWorkflowID = subWM.ID
						}
						details = append(details, cd)
					}
				}
				wm.Calls[callName] = details
			}
		}
	}

	return wm
}

// parseDetailedCallDetails parses detailed call information from a map.
func parseDetailedCallDetails(m map[string]interface{}) metadata.CallDetails {
	cd := metadata.CallDetails{
		Inputs:          make(map[string]interface{}),
		Outputs:         make(map[string]interface{}),
		Labels:          make(map[string]string),
		ExecutionEvents: []metadata.ExecutionEvent{},
	}

	// Identification
	cd.ShardIndex = getInt(m, "shardIndex")
	cd.Attempt = getInt(m, "attempt")
	cd.JobID = getString(m, "jobId")

	// Status
	cd.ExecutionStatus = getString(m, "executionStatus")
	cd.BackendStatus = getString(m, "backendStatus")
	if rc, ok := m["returnCode"]; ok {
		if rcInt, ok := rc.(float64); ok {
			code := int(rcInt)
			cd.ReturnCode = &code
		}
	}

	// Timing
	cd.Start = parseTime(getString(m, "start"))
	cd.End = parseTime(getString(m, "end"))
	cd.VMStartTime = parseTime(getString(m, "vmStartTime"))
	cd.VMEndTime = parseTime(getString(m, "vmEndTime"))

	// Execution
	cd.CommandLine = getString(m, "commandLine")
	cd.Backend = getString(m, "backend")
	cd.CallRoot = getString(m, "callRoot")

	// Logs
	cd.Stdout = getString(m, "stdout")
	cd.Stderr = getString(m, "stderr")
	cd.MonitoringLog = getString(m, "monitoringLog")

	// Docker
	cd.DockerImageUsed = getString(m, "dockerImageUsed")
	if size := m["compressedDockerSize"]; size != nil {
		switch v := size.(type) {
		case float64:
			cd.DockerSize = formatBytes(int64(v))
		case string:
			cd.DockerSize = v
		}
	}

	// SubWorkflow
	cd.SubWorkflowID = getString(m, "subWorkflowId")

	// Cost
	if cost, ok := m["vmCostPerHour"].(float64); ok {
		cd.VMCostPerHour = cost
	}

	// Runtime attributes
	if ra, ok := m["runtimeAttributes"].(map[string]interface{}); ok {
		cd.CPU = getString(ra, "cpu")
		cd.Memory = getString(ra, "memory")
		cd.Disk = getString(ra, "disks")
		cd.Preemptible = getString(ra, "preemptible")
		cd.Zones = getString(ra, "zones")
		cd.DockerImage = getString(ra, "docker")
	}

	// Cache
	if cc, ok := m["callCaching"].(map[string]interface{}); ok {
		cd.CacheHit = getBool(cc, "hit")
		cd.CacheResult = getString(cc, "result")
	}

	// Inputs/Outputs
	if inputs, ok := m["inputs"].(map[string]interface{}); ok {
		cd.Inputs = inputs
	}
	if outputs, ok := m["outputs"].(map[string]interface{}); ok {
		cd.Outputs = outputs
	}

	// Labels
	if labels, ok := m["labels"].(map[string]interface{}); ok {
		for k, v := range labels {
			if s, ok := v.(string); ok {
				cd.Labels[k] = s
			}
		}
	}

	// Failures (task-level errors)
	if failures, ok := m["failures"].([]interface{}); ok {
		cd.Failures = parseDetailedFailures(failures)
	}

	// Execution events
	if events, ok := m["executionEvents"].([]interface{}); ok {
		for _, e := range events {
			if em, ok := e.(map[string]interface{}); ok {
				event := metadata.ExecutionEvent{
					Description: getString(em, "description"),
					Start:       parseTime(getString(em, "startTime")),
					End:         parseTime(getString(em, "endTime")),
				}
				cd.ExecutionEvents = append(cd.ExecutionEvents, event)
			}
		}
	}

	return cd
}

// ConvertToPreemptionCallData converts metadata CallDetails to preemption.CallData.
func ConvertToPreemptionCallData(calls map[string][]metadata.CallDetails) map[string][]preemption.CallData {
	result := make(map[string][]preemption.CallData)

	for callName, callList := range calls {
		var data []preemption.CallData
		for _, cd := range callList {
			// Calculate duration in hours
			var durationHours float64
			if !cd.Start.IsZero() && !cd.End.IsZero() {
				durationHours = cd.End.Sub(cd.Start).Hours()
			}

			data = append(data, preemption.CallData{
				Name:            cd.Name,
				ShardIndex:      cd.ShardIndex,
				Attempt:         cd.Attempt,
				ExecutionStatus: cd.ExecutionStatus,
				Preemptible:     cd.Preemptible,
				ReturnCode:      cd.ReturnCode,
				// Resource info for cost calculation
				CPU:           parseCPU(cd.CPU),
				MemoryGB:      parseMemoryGB(cd.Memory),
				DurationHours: durationHours,
				VMCostPerHour: cd.VMCostPerHour,
			})

			// Recursively process subworkflows
			if cd.SubWorkflowMetadata != nil {
				subData := ConvertToPreemptionCallData(cd.SubWorkflowMetadata.Calls)
				for subName, subCalls := range subData {
					// Prefix subworkflow call names
					fullName := callName + "." + subName
					result[fullName] = append(result[fullName], subCalls...)
				}
			}
		}
		result[callName] = data
	}

	return result
}

// --- Helper Functions ---

// getString extracts a string value from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt extracts an int value from a map.
func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// getBool extracts a bool value from a map.
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// parseTime parses a time string in RFC3339 format.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try alternate format
		t, _ = time.Parse("2006-01-02T15:04:05.000Z", s)
	}
	return t
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// parseCPU parses CPU string (e.g., "4", "4.0") to float64.
func parseCPU(s string) float64 {
	if s == "" {
		return 0
	}
	var cpu float64
	for i, c := range s {
		if c == '.' {
			// Parse decimal part
			var decimal float64
			var divisor float64 = 10
			for _, d := range s[i+1:] {
				if d < '0' || d > '9' {
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
			}
			return cpu + decimal
		}
		if c < '0' || c > '9' {
			break
		}
		cpu = cpu*10 + float64(c-'0')
	}
	return cpu
}

// parseMemoryGB parses memory string (e.g., "8 GB", "8GB", "8192 MB") to GB.
func parseMemoryGB(s string) float64 {
	if s == "" {
		return 0
	}

	// Extract numeric part
	var num float64
	var i int
	for i = 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			// Parse decimal part
			var decimal float64
			var divisor float64 = 10
			for j := i + 1; j < len(s); j++ {
				d := s[j]
				if d < '0' || d > '9' {
					i = j
					break
				}
				decimal += float64(d-'0') / divisor
				divisor *= 10
				i = j + 1
			}
			num += decimal
			break
		}
		if c < '0' || c > '9' {
			break
		}
		num = num*10 + float64(c-'0')
	}

	// Check for unit
	rest := strings.ToUpper(strings.TrimSpace(s[i:]))
	if strings.HasPrefix(rest, "MB") {
		return num / 1024
	}
	if strings.HasPrefix(rest, "TB") {
		return num * 1024
	}
	// Default to GB
	return num
}

// CalculatePreemptionStats calculates preemption statistics for all calls in-place.
func CalculatePreemptionStats(calls map[string][]metadata.CallDetails) {
	for callName, callList := range calls {
		// Group by shard index
		shardGroups := make(map[int][]*metadata.CallDetails)
		for i := range callList {
			cd := &callList[i]
			shardGroups[cd.ShardIndex] = append(shardGroups[cd.ShardIndex], cd)

			// Recursively calculate for subworkflows
			if cd.SubWorkflowMetadata != nil {
				CalculatePreemptionStats(cd.SubWorkflowMetadata.Calls)
			}
		}

		// Update the slice (since we're modifying pointers)
		calls[callName] = callList
	}
}

// AggregateCallStatus returns the aggregate status for a list of calls.
func AggregateCallStatus(calls []metadata.CallDetails) string {
	hasDone := false
	hasRunning := false
	hasFailed := false
	hasPreempted := false

	for _, c := range calls {
		switch c.ExecutionStatus {
		case "Done", "Succeeded":
			hasDone = true
		case "Running":
			hasRunning = true
		case "Failed":
			hasFailed = true
		case "Preempted", "RetryableFailure":
			hasPreempted = true
		}
	}

	if hasRunning {
		return "Running"
	}
	if hasDone {
		return "Done"
	}
	if hasFailed {
		return "Failed"
	}
	if hasPreempted && len(calls) > 0 {
		return "Preempted"
	}
	return "Unknown"
}

// EarliestCallStart returns the earliest start time from a list of calls.
func EarliestCallStart(calls []metadata.CallDetails) time.Time {
	var earliest time.Time
	for _, c := range calls {
		if earliest.IsZero() || c.Start.Before(earliest) {
			earliest = c.Start
		}
	}
	return earliest
}

// LatestCallEnd returns the latest end time from a list of calls.
func LatestCallEnd(calls []metadata.CallDetails) time.Time {
	var latest time.Time
	for _, c := range calls {
		if c.End.After(latest) {
			latest = c.End
		}
	}
	return latest
}

// GetMostRecentAttempt returns the call with the highest attempt number.
func GetMostRecentAttempt(calls []metadata.CallDetails) metadata.CallDetails {
	if len(calls) == 0 {
		return metadata.CallDetails{}
	}
	sorted := make([]metadata.CallDetails, len(calls))
	copy(sorted, calls)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Attempt > sorted[j].Attempt
	})
	return sorted[0]
}
