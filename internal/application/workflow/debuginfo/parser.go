package debuginfo

import (
	"encoding/json"
	"sort"

	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
)

// ParseMetadata parses raw JSON metadata into WorkflowMetadata.
func ParseMetadata(data []byte) (*WorkflowMetadata, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	wm := &WorkflowMetadata{
		Calls:   make(map[string][]CallDetails),
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
		wm.Failures = parseFailures(failures)
	}

	// Parse calls
	if calls, ok := raw["calls"].(map[string]interface{}); ok {
		for callName, callList := range calls {
			if list, ok := callList.([]interface{}); ok {
				var details []CallDetails
				for _, item := range list {
					if callMap, ok := item.(map[string]interface{}); ok {
						cd := parseCallDetails(callMap)
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
		// Calculate preemption stats for all calls
		CalculatePreemptionStats(wm.Calls)
	}

	return wm, nil
}

// parseFailures parses the failures array from Cromwell metadata
func parseFailures(failures []interface{}) []Failure {
	var result []Failure
	for _, f := range failures {
		if fMap, ok := f.(map[string]interface{}); ok {
			failure := Failure{
				Message: getString(fMap, "message"),
			}
			if causedBy, ok := fMap["causedBy"].([]interface{}); ok {
				failure.CausedBy = parseFailures(causedBy)
			}
			result = append(result, failure)
		}
	}
	return result
}

// parseSubWorkflowMetadata parses embedded subworkflow metadata
func parseSubWorkflowMetadata(raw map[string]interface{}) *WorkflowMetadata {
	wm := &WorkflowMetadata{
		Calls:   make(map[string][]CallDetails),
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
				var details []CallDetails
				for _, item := range list {
					if callMap, ok := item.(map[string]interface{}); ok {
						cd := parseCallDetails(callMap)
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

func parseCallDetails(m map[string]interface{}) CallDetails {
	cd := CallDetails{
		Inputs:          make(map[string]interface{}),
		Outputs:         make(map[string]interface{}),
		Labels:          make(map[string]string),
		ExecutionEvents: []ExecutionEvent{},
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
			cd.DockerSize = FormatBytes(int64(v))
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

	// Execution events
	if events, ok := m["executionEvents"].([]interface{}); ok {
		for _, e := range events {
			if em, ok := e.(map[string]interface{}); ok {
				event := ExecutionEvent{
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

// CalculatePreemptionStats calculates preemption statistics for all calls.
// It groups calls by task name and shard, then calculates stats for each group.
func CalculatePreemptionStats(calls map[string][]CallDetails) {
	for callName, callList := range calls {
		// Group by shard index
		shardGroups := make(map[int][]*CallDetails)
		for i := range callList {
			cd := &callList[i]
			shardGroups[cd.ShardIndex] = append(shardGroups[cd.ShardIndex], cd)

			// Recursively calculate for subworkflows
			if cd.SubWorkflowMetadata != nil {
				CalculatePreemptionStats(cd.SubWorkflowMetadata.Calls)
			}
		}

		// Calculate stats for each shard group
		for _, shardCalls := range shardGroups {
			stats := calculateShardPreemptionStats(shardCalls)
			// Apply stats to all attempts in this shard (they share the same stats)
			for _, cd := range shardCalls {
				cd.PreemptionStats = stats
			}
		}

		// Update the slice (since we're modifying pointers)
		calls[callName] = callList
	}
}

// calculateShardPreemptionStats calculates preemption stats for a single task/shard.
func calculateShardPreemptionStats(attempts []*CallDetails) *PreemptionStats {
	if len(attempts) == 0 {
		return nil
	}

	// Sort by attempt number to get the final attempt
	sort.Slice(attempts, func(i, j int) bool {
		return attempts[i].Attempt < attempts[j].Attempt
	})

	stats := &PreemptionStats{
		TotalAttempts:  len(attempts),
		PreemptedCount: len(attempts) - 1,
	}

	// Get preemptible config from the last attempt
	finalAttempt := attempts[len(attempts)-1]
	stats.IsPreemptible = preemption.IsPreemptible(finalAttempt.Preemptible)
	stats.MaxPreemptible = preemption.ParseMaxPreemptible(finalAttempt.Preemptible)

	if stats.IsPreemptible {
		// Calculate efficiency score
		if stats.MaxPreemptible > 0 {
			// Score based on how many retries out of max were used
			stats.EfficiencyScore = 1.0 - (float64(stats.PreemptedCount) / float64(stats.MaxPreemptible))
		} else {
			// Without max info, use 1/attempts as score
			stats.EfficiencyScore = 1.0 / float64(stats.TotalAttempts)
		}

		// Clamp to [0, 1]
		if stats.EfficiencyScore < 0 {
			stats.EfficiencyScore = 0
		}
		if stats.EfficiencyScore > 1 {
			stats.EfficiencyScore = 1
		}
	} else {
		stats.EfficiencyScore = 1.0
	}

	return stats
}

// CalculateWorkflowPreemptionSummary aggregates preemption stats across all calls.
func CalculateWorkflowPreemptionSummary(workflowID, workflowName string, calls map[string][]CallDetails) *WorkflowPreemptionSummary {
	// Convert to preemption.CallData
	callData := ConvertToCallData(calls)

	// Use the preemption analyzer
	analyzer := preemption.NewAnalyzer()
	result := analyzer.AnalyzeWorkflow(workflowID, workflowName, callData)

	// Convert result back to local types
	summary := &WorkflowPreemptionSummary{
		TotalTasks:        result.TotalTasks,
		PreemptibleTasks:  result.PreemptibleTasks,
		TotalAttempts:     result.TotalAttempts,
		TotalPreemptions:  result.TotalPreemptions,
		OverallEfficiency: result.OverallEfficiency,
		ProblematicTasks:  make([]ProblematicTask, len(result.ProblematicTasks)),
		// Cost metrics
		TotalCost:      result.TotalCost,
		WastedCost:     result.WastedCost,
		CostEfficiency: result.CostEfficiency,
		CostUnit:       result.CostUnit,
	}

	for i, pt := range result.ProblematicTasks {
		summary.ProblematicTasks[i] = ProblematicTask{
			Name:            pt.Name,
			ShardCount:      pt.ShardCount,
			Attempts:        pt.TotalAttempts,
			Preemptions:     pt.TotalPreemptions,
			EfficiencyScore: pt.EfficiencyScore,
			// Cost metrics
			TotalCost:      pt.TotalCost,
			WastedCost:     pt.WastedCost,
			CostEfficiency: pt.CostEfficiency,
			ImpactPercent:  pt.ImpactPercent,
		}
	}

	return summary
}

// ConvertToCallData converts CallDetails to preemption.CallData.
func ConvertToCallData(calls map[string][]CallDetails) map[string][]preemption.CallData {
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
				CPU:           ParseCPU(cd.CPU),
				MemoryGB:      ParseMemoryGB(cd.Memory),
				DurationHours: durationHours,
				VMCostPerHour: cd.VMCostPerHour,
			})

			// Recursively process subworkflows
			if cd.SubWorkflowMetadata != nil {
				subData := ConvertToCallData(cd.SubWorkflowMetadata.Calls)
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
