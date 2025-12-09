package debug

import (
	"github.com/lmtani/pumbaa/internal/application/workflow/debuginfo"
)

// ParseMetadata parses raw JSON metadata into WorkflowMetadata.
// Delegates to the debuginfo package.
func ParseMetadata(data []byte) (*WorkflowMetadata, error) {
	return debuginfo.ParseMetadata(data)
}

// BuildTree builds a tree structure from WorkflowMetadata.
// Delegates to the debuginfo package.
func BuildTree(wm *WorkflowMetadata) *TreeNode {
	return debuginfo.BuildTree(wm)
}

// GetVisibleNodes returns a flat list of visible nodes for rendering.
// Delegates to the debuginfo package.
func GetVisibleNodes(root *TreeNode) []*TreeNode {
	return debuginfo.GetVisibleNodes(root)
}

// CalculateWorkflowPreemptionSummary aggregates preemption stats across all calls.
// Delegates to the debuginfo package.
func CalculateWorkflowPreemptionSummary(workflowID, workflowName string, calls map[string][]CallDetails) *WorkflowPreemptionSummary {
	return debuginfo.CalculateWorkflowPreemptionSummary(workflowID, workflowName, calls)
}
