package debug

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
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

	// Parse calls
	if calls, ok := raw["calls"].(map[string]interface{}); ok {
		for callName, callList := range calls {
			if list, ok := callList.([]interface{}); ok {
				var details []CallDetails
				for _, item := range list {
					if callMap, ok := item.(map[string]interface{}); ok {
						details = append(details, parseCallDetails(callMap))
					}
				}
				wm.Calls[callName] = details
			}
		}
	}

	return wm, nil
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

// BuildTree builds a tree structure from WorkflowMetadata.
func BuildTree(wm *WorkflowMetadata) *TreeNode {
	root := &TreeNode{
		ID:       wm.ID,
		Name:     wm.Name,
		Type:     NodeTypeWorkflow,
		Status:   wm.Status,
		Start:    wm.Start,
		End:      wm.End,
		Duration: wm.End.Sub(wm.Start),
		Expanded: true,
		Children: []*TreeNode{},
		Depth:    0,
	}

	// Sort call names for consistent ordering
	var callNames []string
	for name := range wm.Calls {
		callNames = append(callNames, name)
	}
	sort.Strings(callNames)

	for _, callName := range callNames {
		calls := wm.Calls[callName]
		// Extract task name from full call name (WorkflowName.TaskName)
		taskName := callName
		if idx := strings.LastIndex(callName, "."); idx != -1 {
			taskName = callName[idx+1:]
		}

		// If there's only one call without shards, add it directly
		if len(calls) == 1 && calls[0].ShardIndex == -1 {
			call := calls[0]
			nodeType := NodeTypeCall
			if call.SubWorkflowID != "" {
				nodeType = NodeTypeSubWorkflow
			}

			child := &TreeNode{
				ID:            callName,
				Name:          taskName,
				Type:          nodeType,
				Status:        call.ExecutionStatus,
				Start:         call.Start,
				End:           call.End,
				Duration:      call.End.Sub(call.Start),
				Expanded:      false,
				Parent:        root,
				CallData:      &call,
				SubWorkflowID: call.SubWorkflowID,
				Depth:         1,
			}
			root.Children = append(root.Children, child)
		} else {
			// Multiple shards - create a parent node
			parent := &TreeNode{
				ID:       callName,
				Name:     taskName,
				Type:     NodeTypeCall,
				Status:   aggregateStatus(calls),
				Start:    earliestStart(calls),
				End:      latestEnd(calls),
				Expanded: false,
				Parent:   root,
				Children: []*TreeNode{},
				Depth:    1,
			}
			parent.Duration = parent.End.Sub(parent.Start)

			// Sort calls by shard index
			sort.Slice(calls, func(i, j int) bool {
				return calls[i].ShardIndex < calls[j].ShardIndex
			})

			for i := range calls {
				call := calls[i]
				shardName := taskName
				if call.ShardIndex >= 0 {
					shardName = fmt.Sprintf("%s [shard %d]", taskName, call.ShardIndex)
				}
				if call.Attempt > 1 {
					shardName += fmt.Sprintf(" (attempt %d)", call.Attempt)
				}

				child := &TreeNode{
					ID:       fmt.Sprintf("%s_%d", callName, call.ShardIndex),
					Name:     shardName,
					Type:     NodeTypeShard,
					Status:   call.ExecutionStatus,
					Start:    call.Start,
					End:      call.End,
					Duration: call.End.Sub(call.Start),
					Parent:   parent,
					CallData: &call,
					Depth:    2,
				}
				parent.Children = append(parent.Children, child)
			}
			root.Children = append(root.Children, parent)
		}
	}

	// Sort children by start time
	sort.Slice(root.Children, func(i, j int) bool {
		return root.Children[i].Start.Before(root.Children[j].Start)
	})

	return root
}

// GetVisibleNodes returns a flat list of visible nodes for rendering.
func GetVisibleNodes(root *TreeNode) []*TreeNode {
	var result []*TreeNode
	collectVisible(root, &result)
	return result
}

func collectVisible(node *TreeNode, result *[]*TreeNode) {
	*result = append(*result, node)
	if node.Expanded {
		for _, child := range node.Children {
			collectVisible(child, result)
		}
	}
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

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

func aggregateStatus(calls []CallDetails) string {
	for _, c := range calls {
		if c.ExecutionStatus == "Failed" {
			return "Failed"
		}
	}
	for _, c := range calls {
		if c.ExecutionStatus == "Running" {
			return "Running"
		}
	}
	allDone := true
	for _, c := range calls {
		if c.ExecutionStatus != "Done" {
			allDone = false
			break
		}
	}
	if allDone {
		return "Done"
	}
	return "Unknown"
}

func earliestStart(calls []CallDetails) time.Time {
	var earliest time.Time
	for _, c := range calls {
		if earliest.IsZero() || c.Start.Before(earliest) {
			earliest = c.Start
		}
	}
	return earliest
}

func latestEnd(calls []CallDetails) time.Time {
	var latest time.Time
	for _, c := range calls {
		if c.End.After(latest) {
			latest = c.End
		}
	}
	return latest
}
