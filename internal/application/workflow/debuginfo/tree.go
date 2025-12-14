package debuginfo

import (
	"fmt"
	"sort"
	"strings"
)

// treeBuilder encapsulates the logic for building workflow tree nodes.
// It follows the Builder pattern to construct complex tree structures.
type treeBuilder struct {
	baseDepth int
	parent    *TreeNode
}

// newTreeBuilder creates a new tree builder with the given configuration.
func newTreeBuilder(baseDepth int, parent *TreeNode) *treeBuilder {
	return &treeBuilder{
		baseDepth: baseDepth,
		parent:    parent,
	}
}

// BuildTree builds a tree structure from WorkflowMetadata.
func BuildTree(wm *WorkflowMetadata) *TreeNode {
	builder := newTreeBuilder(0, nil)
	return builder.buildWorkflowNode(wm)
}

// AddSubWorkflowChildren adds the calls from a subworkflow as children of the given node.
func AddSubWorkflowChildren(node *TreeNode, subWM *WorkflowMetadata, baseDepth int) {
	builder := newTreeBuilder(baseDepth, node)
	builder.addCallsToParent(node, subWM.Calls)
}

// buildWorkflowNode creates the root workflow node and populates its children.
func (b *treeBuilder) buildWorkflowNode(wm *WorkflowMetadata) *TreeNode {
	root := &TreeNode{
		ID:       wm.ID,
		Name:     wm.Name,
		Type:     NodeTypeWorkflow,
		Status:   wm.Status,
		Start:    wm.Start,
		End:      wm.End,
		Duration: wm.End.Sub(wm.Start),
		Expanded: b.baseDepth == 0,
		Children: []*TreeNode{},
		Parent:   b.parent,
		Depth:    b.baseDepth,
	}

	b.addCallsToParent(root, wm.Calls)
	return root
}

// addCallsToParent processes all calls and adds them as children to the parent node.
func (b *treeBuilder) addCallsToParent(parent *TreeNode, calls map[string][]CallDetails) {
	callNames := sortedCallNames(calls)

	for _, callName := range callNames {
		callList := calls[callName]
		taskName := extractTaskName(callName)

		if isSingleNonShardedCall(callList) {
			b.addSingleCallNode(parent, callName, taskName, callList[0])
		} else {
			b.addShardedCallNode(parent, callName, taskName, callList)
		}
	}

	sortChildrenByStartTime(parent.Children)
}

// addSingleCallNode creates and adds a node for a single non-sharded call.
func (b *treeBuilder) addSingleCallNode(parent *TreeNode, callName, taskName string, call CallDetails) {
	nodeType, isSubWorkflow := determineNodeType(call, NodeTypeCall)
	childDepth := b.calculateChildDepth(parent)

	child := &TreeNode{
		ID:            callName,
		Name:          taskName,
		Type:          nodeType,
		Status:        call.ExecutionStatus,
		Start:         call.Start,
		End:           call.End,
		Duration:      call.End.Sub(call.Start),
		Expanded:      false,
		Parent:        parent,
		CallData:      &call,
		SubWorkflowID: call.SubWorkflowID,
		Depth:         childDepth,
		Children:      []*TreeNode{},
	}

	if isSubWorkflow && call.SubWorkflowMetadata != nil {
		AddSubWorkflowChildren(child, call.SubWorkflowMetadata, childDepth+1)
	}

	parent.Children = append(parent.Children, child)
}

// addShardedCallNode creates a parent node for sharded calls and adds individual shard nodes.
func (b *treeBuilder) addShardedCallNode(parent *TreeNode, callName, taskName string, calls []CallDetails) {
	childDepth := b.calculateChildDepth(parent)

	shardParent := &TreeNode{
		ID:       callName,
		Name:     taskName,
		Type:     NodeTypeCall,
		Status:   AggregateStatus(calls),
		Start:    EarliestStart(calls),
		End:      LatestEnd(calls),
		Expanded: false,
		Parent:   parent,
		Children: []*TreeNode{},
		Depth:    childDepth,
	}
	shardParent.Duration = shardParent.End.Sub(shardParent.Start)

	shardGroups := groupCallsByShardIndex(calls)
	shardIndices := sortedShardIndices(shardGroups)

	for _, shardIdx := range shardIndices {
		b.addShardNode(shardParent, callName, taskName, shardIdx, shardGroups[shardIdx], childDepth+1)
	}

	parent.Children = append(parent.Children, shardParent)
}

// addShardNode creates and adds a node for a single shard (aggregating all attempts).
func (b *treeBuilder) addShardNode(parent *TreeNode, callName, taskName string, shardIdx int, shardCalls []CallDetails, depth int) {
	mostRecentCall := getMostRecentAttempt(shardCalls)
	shardStatus := AggregateStatus(shardCalls)
	shardName := buildShardName(taskName, shardIdx, mostRecentCall.Attempt)
	nodeType, isSubWorkflow := determineNodeType(mostRecentCall, NodeTypeShard)

	child := &TreeNode{
		ID:            fmt.Sprintf("%s_%d", callName, shardIdx),
		Name:          shardName,
		Type:          nodeType,
		Status:        shardStatus,
		Start:         EarliestStart(shardCalls),
		End:           LatestEnd(shardCalls),
		Duration:      LatestEnd(shardCalls).Sub(EarliestStart(shardCalls)),
		Parent:        parent,
		CallData:      &mostRecentCall,
		SubWorkflowID: mostRecentCall.SubWorkflowID,
		Depth:         depth,
		Children:      []*TreeNode{},
	}

	if isSubWorkflow && mostRecentCall.SubWorkflowMetadata != nil {
		AddSubWorkflowChildren(child, mostRecentCall.SubWorkflowMetadata, depth+1)
	}

	parent.Children = append(parent.Children, child)
}

// calculateChildDepth determines the depth for child nodes based on context.
func (b *treeBuilder) calculateChildDepth(parent *TreeNode) int {
	// For workflow roots, children are at baseDepth + 1
	// For subworkflow additions, we use the builder's baseDepth directly
	if parent.Type == NodeTypeWorkflow {
		return b.baseDepth + 1
	}
	return b.baseDepth
}

// --- Helper Functions ---

// sortedCallNames returns call names sorted alphabetically.
func sortedCallNames(calls map[string][]CallDetails) []string {
	names := make([]string, 0, len(calls))
	for name := range calls {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// extractTaskName extracts the task name from a full call name (WorkflowName.TaskName).
func extractTaskName(callName string) string {
	if idx := strings.LastIndex(callName, "."); idx != -1 {
		return callName[idx+1:]
	}
	return callName
}

// isSingleNonShardedCall checks if there's only one call without sharding.
func isSingleNonShardedCall(calls []CallDetails) bool {
	return len(calls) == 1 && calls[0].ShardIndex == -1
}

// determineNodeType determines the node type based on whether it's a subworkflow.
func determineNodeType(call CallDetails, defaultType NodeType) (NodeType, bool) {
	isSubWorkflow := call.SubWorkflowID != "" || call.SubWorkflowMetadata != nil
	if isSubWorkflow {
		return NodeTypeSubWorkflow, true
	}
	return defaultType, false
}

// groupCallsByShardIndex groups calls by their shard index.
func groupCallsByShardIndex(calls []CallDetails) map[int][]CallDetails {
	groups := make(map[int][]CallDetails)
	for _, call := range calls {
		groups[call.ShardIndex] = append(groups[call.ShardIndex], call)
	}
	return groups
}

// sortedShardIndices returns shard indices sorted in ascending order.
func sortedShardIndices(groups map[int][]CallDetails) []int {
	indices := make([]int, 0, len(groups))
	for idx := range groups {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	return indices
}

// getMostRecentAttempt returns the call with the highest attempt number.
func getMostRecentAttempt(calls []CallDetails) CallDetails {
	if len(calls) == 0 {
		return CallDetails{}
	}
	sorted := make([]CallDetails, len(calls))
	copy(sorted, calls)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Attempt > sorted[j].Attempt
	})
	return sorted[0]
}

// buildShardName constructs the display name for a shard node.
func buildShardName(taskName string, shardIdx, attempt int) string {
	name := taskName
	if shardIdx >= 0 {
		name = fmt.Sprintf("%s [shard %d]", taskName, shardIdx)
	}
	if attempt > 1 {
		name += fmt.Sprintf(" (attempt %d)", attempt)
	}
	return name
}

// sortChildrenByStartTime sorts nodes by their start time.
func sortChildrenByStartTime(children []*TreeNode) {
	sort.Slice(children, func(i, j int) bool {
		return children[i].Start.Before(children[j].Start)
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
