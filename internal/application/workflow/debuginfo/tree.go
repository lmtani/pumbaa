package debuginfo

import (
	"fmt"
	"sort"
	"strings"
)

// BuildTree builds a tree structure from WorkflowMetadata.
func BuildTree(wm *WorkflowMetadata) *TreeNode {
	return buildTreeWithDepth(wm, 0, nil)
}

// buildTreeWithDepth recursively builds a tree with proper depth tracking
func buildTreeWithDepth(wm *WorkflowMetadata, baseDepth int, parent *TreeNode) *TreeNode {
	root := &TreeNode{
		ID:       wm.ID,
		Name:     wm.Name,
		Type:     NodeTypeWorkflow,
		Status:   wm.Status,
		Start:    wm.Start,
		End:      wm.End,
		Duration: wm.End.Sub(wm.Start),
		Expanded: baseDepth == 0, // Only expand root workflow
		Children: []*TreeNode{},
		Parent:   parent,
		Depth:    baseDepth,
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
			isSubWorkflow := call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil
			nodeType := NodeTypeCall
			if isSubWorkflow {
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
				Depth:         baseDepth + 1,
				Children:      []*TreeNode{},
			}

			// If this is a subworkflow with embedded metadata, build its children
			if isSubWorkflow && call.SubWorkflowMetadata != nil {
				AddSubWorkflowChildren(child, call.SubWorkflowMetadata, baseDepth+2)
			}

			root.Children = append(root.Children, child)
		} else {
			// Multiple shards - create a parent node
			parent := &TreeNode{
				ID:       callName,
				Name:     taskName,
				Type:     NodeTypeCall,
				Status:   AggregateStatus(calls),
				Start:    EarliestStart(calls),
				End:      LatestEnd(calls),
				Expanded: false,
				Parent:   root,
				Children: []*TreeNode{},
				Depth:    baseDepth + 1,
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

				isSubWorkflow := call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil
				nodeType := NodeTypeShard
				if isSubWorkflow {
					nodeType = NodeTypeSubWorkflow
				}

				child := &TreeNode{
					ID:            fmt.Sprintf("%s_%d", callName, call.ShardIndex),
					Name:          shardName,
					Type:          nodeType,
					Status:        call.ExecutionStatus,
					Start:         call.Start,
					End:           call.End,
					Duration:      call.End.Sub(call.Start),
					Parent:        parent,
					CallData:      &call,
					SubWorkflowID: call.SubWorkflowID,
					Depth:         baseDepth + 2,
					Children:      []*TreeNode{},
				}

				// If this is a subworkflow with embedded metadata, build its children
				if isSubWorkflow && call.SubWorkflowMetadata != nil {
					AddSubWorkflowChildren(child, call.SubWorkflowMetadata, baseDepth+3)
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

// AddSubWorkflowChildren adds the calls from a subworkflow as children of the given node
func AddSubWorkflowChildren(node *TreeNode, subWM *WorkflowMetadata, baseDepth int) {
	// Sort call names for consistent ordering
	var callNames []string
	for name := range subWM.Calls {
		callNames = append(callNames, name)
	}
	sort.Strings(callNames)

	for _, callName := range callNames {
		calls := subWM.Calls[callName]
		// Extract task name from full call name (WorkflowName.TaskName)
		taskName := callName
		if idx := strings.LastIndex(callName, "."); idx != -1 {
			taskName = callName[idx+1:]
		}

		if len(calls) == 1 && calls[0].ShardIndex == -1 {
			call := calls[0]
			isSubWorkflow := call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil
			nodeType := NodeTypeCall
			if isSubWorkflow {
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
				Parent:        node,
				CallData:      &call,
				SubWorkflowID: call.SubWorkflowID,
				Depth:         baseDepth,
				Children:      []*TreeNode{},
			}

			// Recursively add children for nested subworkflows
			if isSubWorkflow && call.SubWorkflowMetadata != nil {
				AddSubWorkflowChildren(child, call.SubWorkflowMetadata, baseDepth+1)
			}

			node.Children = append(node.Children, child)
		} else {
			// Multiple shards
			parent := &TreeNode{
				ID:       callName,
				Name:     taskName,
				Type:     NodeTypeCall,
				Status:   AggregateStatus(calls),
				Start:    EarliestStart(calls),
				End:      LatestEnd(calls),
				Expanded: false,
				Parent:   node,
				Children: []*TreeNode{},
				Depth:    baseDepth,
			}
			parent.Duration = parent.End.Sub(parent.Start)

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

				isSubWorkflow := call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil
				childType := NodeTypeShard
				if isSubWorkflow {
					childType = NodeTypeSubWorkflow
				}

				child := &TreeNode{
					ID:            fmt.Sprintf("%s_%d", callName, call.ShardIndex),
					Name:          shardName,
					Type:          childType,
					Status:        call.ExecutionStatus,
					Start:         call.Start,
					End:           call.End,
					Duration:      call.End.Sub(call.Start),
					Parent:        parent,
					CallData:      &call,
					SubWorkflowID: call.SubWorkflowID,
					Depth:         baseDepth + 1,
					Children:      []*TreeNode{},
				}

				if isSubWorkflow && call.SubWorkflowMetadata != nil {
					AddSubWorkflowChildren(child, call.SubWorkflowMetadata, baseDepth+2)
				}

				parent.Children = append(parent.Children, child)
			}
			node.Children = append(node.Children, parent)
		}
	}

	// Sort children by start time
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Start.Before(node.Children[j].Start)
	})
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
