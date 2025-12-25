package cromwell

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
)

// mapMetadataToWorkflow converts a metadata response to a domain Workflow.
func (c *Client) mapMetadataToWorkflow(m *metadataResponse) *workflow.Workflow {
	return mapMetadataResponseToWorkflow(m)
}

// mapMetadataResponseToWorkflow converts a metadata response to a domain Workflow.
// This is the internal function that can be used without a Client instance.
func mapMetadataResponseToWorkflow(m *metadataResponse) *workflow.Workflow {
	wf := &workflow.Workflow{
		ID:          m.ID,
		Name:        m.WorkflowName,
		Status:      workflow.Status(m.Status),
		Start:       m.Start,
		End:         m.End,
		SubmittedAt: m.Submission,
		Labels:      m.Labels,
		Inputs:      m.Inputs,
		Outputs:     m.Outputs,
		Calls:       make(map[string][]workflow.Call),
		Failures:    make([]workflow.Failure, 0),

		// Detailed fields
		WorkflowRoot:            m.WorkflowRoot,
		WorkflowLog:             m.WorkflowLog,
		WorkflowLanguage:        m.ActualWorkflowLanguage,
		WorkflowLanguageVersion: m.ActualWorkflowLanguageVersion,
	}

	// Map submitted files
	if m.SubmittedFiles != nil {
		wf.SubmittedWorkflow = m.SubmittedFiles.Workflow
		wf.SubmittedInputs = m.SubmittedFiles.Inputs
		wf.SubmittedOptions = m.SubmittedFiles.Options
	}

	// Map calls with all detailed fields
	for callName, calls := range m.Calls {
		wf.Calls[callName] = make([]workflow.Call, 0, len(calls))
		for _, call := range calls {
			wf.Calls[callName] = append(wf.Calls[callName], mapCallMetadataToCall(callName, &call))
		}
	}

	// Map failures
	wf.Failures = mapFailures(m.Failures)

	return wf
}

// mapCallMetadataToCall converts a callMetadata to a domain Call with all fields.
func mapCallMetadataToCall(callName string, call *callMetadata) workflow.Call {
	c := workflow.Call{
		// Identification
		Name:       callName,
		ShardIndex: call.ShardIndex,
		Attempt:    call.Attempt,
		JobID:      call.JobID,

		// Status
		Status:        workflow.Status(call.ExecutionStatus),
		BackendStatus: call.BackendStatus,
		ReturnCode:    call.ReturnCode,

		// Timing
		Start:       call.Start,
		End:         call.End,
		VMStartTime: call.VMStartTime,
		VMEndTime:   call.VMEndTime,

		// Execution
		Backend:           call.Backend,
		CommandLine:       call.CommandLine,
		CallRoot:          call.CallRoot,
		RuntimeAttributes: call.RuntimeAttributes,

		// Logs
		Stdout:        call.Stdout,
		Stderr:        call.Stderr,
		MonitoringLog: call.MonitoringLog,

		// Docker
		DockerImageUsed: call.DockerImageUsed,
		DockerSize:      formatBytes(parseDockerSize(call.CompressedDockerSize)),

		// Cost
		VMCostPerHour: call.VMCostPerHour,

		// Inputs/Outputs
		Inputs:  call.Inputs,
		Outputs: call.Outputs,

		// Labels
		Labels: call.Labels,

		// Failures
		Failures: mapFailures(call.Failures),

		// SubWorkflow
		SubWorkflowID: call.SubWorkflowID,
	}

	// Map runtime attributes to convenience fields
	if call.RuntimeAttributes != nil {
		if cpu, ok := call.RuntimeAttributes["cpu"].(string); ok {
			c.CPU = cpu
		} else if cpuNum, ok := call.RuntimeAttributes["cpu"].(float64); ok {
			c.CPU = fmt.Sprintf("%v", cpuNum)
		}
		if mem, ok := call.RuntimeAttributes["memory"].(string); ok {
			c.Memory = mem
		}
		if disk, ok := call.RuntimeAttributes["disks"].(string); ok {
			c.Disk = disk
		}
		if preempt, ok := call.RuntimeAttributes["preemptible"].(string); ok {
			c.Preemptible = preempt
		} else if preemptNum, ok := call.RuntimeAttributes["preemptible"].(float64); ok {
			c.Preemptible = fmt.Sprintf("%v", preemptNum)
		}
		if zones, ok := call.RuntimeAttributes["zones"].(string); ok {
			c.Zones = zones
		}
		if docker, ok := call.RuntimeAttributes["docker"].(string); ok {
			c.DockerImage = docker
		}
	}

	// Map call caching
	if call.CallCaching != nil {
		c.CacheHit = call.CallCaching.Hit
		c.CacheResult = call.CallCaching.Result
	}

	// Map execution events
	c.ExecutionEvents = mapExecutionEvents(call.ExecutionEvents)

	// Map subworkflow metadata recursively
	if call.SubWorkflowMetadata != nil {
		c.SubWorkflowMetadata = mapMetadataResponseToWorkflow(call.SubWorkflowMetadata)
	}

	return c
}

// mapExecutionEvents converts execution event metadata to domain events.
func mapExecutionEvents(events []executionEventMeta) []workflow.ExecutionEvent {
	result := make([]workflow.ExecutionEvent, 0, len(events))
	for _, e := range events {
		result = append(result, workflow.ExecutionEvent{
			Description: e.Description,
			Start:       parseTimeString(e.StartTime),
			End:         parseTimeString(e.EndTime),
		})
	}
	return result
}

// mapFailures converts failure metadata to domain failures.
func mapFailures(failures []failureMetadata) []workflow.Failure {
	result := make([]workflow.Failure, 0, len(failures))
	for _, f := range failures {
		result = append(result, workflow.Failure{
			Message:  f.Message,
			CausedBy: mapFailures(f.CausedBy),
		})
	}
	return result
}

// ParseDetailedMetadata parses raw JSON metadata into a domain Workflow.
// This is the detailed version used for debugging and analysis.
func ParseDetailedMetadata(data []byte) (*workflow.Workflow, error) {
	var m metadataResponse
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return mapMetadataResponseToWorkflow(&m), nil
}

// ConvertToPreemptionCallData converts workflow calls to preemption.CallData.
func ConvertToPreemptionCallData(calls map[string][]workflow.Call) map[string][]preemption.CallData {
	result := make(map[string][]preemption.CallData)

	for callName, callList := range calls {
		var data []preemption.CallData
		for _, c := range callList {
			// Calculate duration in hours
			var durationHours float64
			if !c.Start.IsZero() && !c.End.IsZero() {
				durationHours = c.End.Sub(c.Start).Hours()
			}

			data = append(data, preemption.CallData{
				Name:            c.Name,
				ShardIndex:      c.ShardIndex,
				Attempt:         c.Attempt,
				ExecutionStatus: string(c.Status),
				Preemptible:     c.Preemptible,
				ReturnCode:      c.ReturnCode,
				CPU:             parseCPU(c.CPU),
				MemoryGB:        parseMemoryGB(c.Memory),
				DurationHours:   durationHours,
				VMCostPerHour:   c.VMCostPerHour,
			})

			// Recursively process subworkflows
			if c.SubWorkflowMetadata != nil {
				subData := ConvertToPreemptionCallData(c.SubWorkflowMetadata.Calls)
				for subName, subCalls := range subData {
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

// parseTimeString parses a time string in RFC3339 format.
func parseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02T15:04:05.000Z", s)
	}
	return t
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	if b == 0 {
		return ""
	}
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

// parseDockerSize parses compressed docker size from interface{} (can be string or number).
func parseDockerSize(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case string:
		if val == "" {
			return 0
		}
		// Parse string as int64
		var result int64
		for _, c := range val {
			if c < '0' || c > '9' {
				break
			}
			result = result*10 + int64(c-'0')
		}
		return result
	default:
		return 0
	}
}

// parseCPU parses CPU string (e.g., "4", "4.0") to float64.
func parseCPU(s string) float64 {
	if s == "" {
		return 0
	}
	var cpu float64
	for i, c := range s {
		if c == '.' {
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

	var num float64
	var i int
	for i = 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
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

	rest := strings.ToUpper(strings.TrimSpace(s[i:]))
	if strings.HasPrefix(rest, "MB") {
		return num / 1024
	}
	if strings.HasPrefix(rest, "TB") {
		return num * 1024
	}
	return num
}
