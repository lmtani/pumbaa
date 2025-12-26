// Package debuginfo provides the use case for workflow debugging.
// This is a simplified orchestration layer that coordinates parsing,
// tree building, and preemption analysis.
package workflow

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// DebugInfo is a high-level composite used by UI layers to render debug views.
type DebugInfo struct {
	Metadata   *workflow.Workflow
	Root       *tree.TreeNode
	Visible    []*tree.TreeNode
	Preemption *workflow.PreemptionSummary
}

// DebugUseCase is the concrete implementation.
type DebugUseCase struct{}

// NewUsecase creates a new Usecase instance.
func NewUsecase() *DebugUseCase {
	return &DebugUseCase{}
}

// GetDebugInfo performs orchestration: parsing -> tree building -> preemption analysis
func (u *DebugUseCase) GetDebugInfo(data []byte) (*DebugInfo, error) {
	// 1. Parse metadata using infrastructure layer
	wf, err := cromwell.ParseDetailedMetadata(data)
	if err != nil {
		return nil, err
	}

	// 2. Build a call tree using the TUI tree package
	root := tree.BuildTree(wf)
	visible := tree.GetVisibleNodes(root)

	// 3. Analyze preemption - now a method on Workflow (DDD pattern)
	summary := wf.CalculatePreemptionSummary()

	return &DebugInfo{
		Metadata:   wf,
		Root:       root,
		Visible:    visible,
		Preemption: summary,
	}, nil
}
