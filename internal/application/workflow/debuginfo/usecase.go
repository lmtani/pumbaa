package debuginfo

// Package debuginfo already contains all the parsing and tree building logic.
// This file adds a Usecase wrapper to keep the same application-layer architecture
// that other features (e.g., preemption) follow under internal/application.

// Usecase defines the public operations the UI or other parts of the application
// use for debugging workflows.
type Usecase interface {
	ParseMetadata(data []byte) (*WorkflowMetadata, error)
	BuildTree(wm *WorkflowMetadata) *TreeNode
	GetVisibleNodes(root *TreeNode) []*TreeNode
	AddSubWorkflowChildren(node *TreeNode, subWM *WorkflowMetadata, baseDepth int)
	CalculateWorkflowPreemptionSummary(workflowID, workflowName string, calls map[string][]CallDetails) *WorkflowPreemptionSummary
	// High-level orchestration: parse metadata, build tree, visible nodes, and preemption summary
	GetDebugInfo(data []byte) (*DebugInfo, error)
}

// usecaseImpl is the concrete implementation delegating to package-level functions.
type usecaseImpl struct{}

// NewUsecase creates a new Usecase instance.
func NewUsecase() Usecase {
	return &usecaseImpl{}
}

// ParseMetadata delegates to the package-level ParseMetadata function.
func (u *usecaseImpl) ParseMetadata(data []byte) (*WorkflowMetadata, error) {
	return ParseMetadata(data)
}

// BuildTree delegates to the package-level BuildTree function.
func (u *usecaseImpl) BuildTree(wm *WorkflowMetadata) *TreeNode {
	return BuildTree(wm)
}

// GetVisibleNodes delegates to the package-level GetVisibleNodes function.
func (u *usecaseImpl) GetVisibleNodes(root *TreeNode) []*TreeNode {
	return GetVisibleNodes(root)
}

// AddSubWorkflowChildren delegates to the package-level AddSubWorkflowChildren function.
func (u *usecaseImpl) AddSubWorkflowChildren(node *TreeNode, subWM *WorkflowMetadata, baseDepth int) {
	AddSubWorkflowChildren(node, subWM, baseDepth)
}

// CalculateWorkflowPreemptionSummary delegates to the package-level function.
func (u *usecaseImpl) CalculateWorkflowPreemptionSummary(workflowID, workflowName string, calls map[string][]CallDetails) *WorkflowPreemptionSummary {
	return CalculateWorkflowPreemptionSummary(workflowID, workflowName, calls)
}

// DebugInfo is a high-level composite used by UI layers to render debug views.
type DebugInfo struct {
	Metadata   *WorkflowMetadata
	Root       *TreeNode
	Visible    []*TreeNode
	Preemption *WorkflowPreemptionSummary
}

// GetDebugInfo performs orchestration: parsing -> tree building -> preemption analysis
func (u *usecaseImpl) GetDebugInfo(data []byte) (*DebugInfo, error) {
	wm, err := ParseMetadata(data)
	if err != nil {
		return nil, err
	}

	// Build call tree and visible nodes
	root := BuildTree(wm)
	visible := GetVisibleNodes(root)

	// Summarize preemption for the workflow level
	summary := CalculateWorkflowPreemptionSummary(wm.ID, wm.Name, wm.Calls)

	return &DebugInfo{
		Metadata:   wm,
		Root:       root,
		Visible:    visible,
		Preemption: summary,
	}, nil
}
