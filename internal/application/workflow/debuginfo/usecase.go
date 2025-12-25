// Package debuginfo provides the use case for workflow debugging.
// This is a simplified orchestration layer that coordinates parsing,
// tree building, and preemption analysis.
package debuginfo

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow/metadata"
	"github.com/lmtani/pumbaa/internal/domain/workflow/preemption"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/debug/tree"
)

// Usecase defines the public operations for debugging workflows.
type Usecase interface {
	// GetDebugInfo performs orchestration: parsing -> tree building -> preemption analysis
	GetDebugInfo(data []byte) (*DebugInfo, error)
}

// DebugInfo is a high-level composite used by UI layers to render debug views.
type DebugInfo struct {
	Metadata   *metadata.WorkflowMetadata
	Root       *tree.TreeNode
	Visible    []*tree.TreeNode
	Preemption *preemption.WorkflowSummary
}

// usecaseImpl is the concrete implementation.
type usecaseImpl struct {
	analyzer *preemption.Analyzer
}

// NewUsecase creates a new Usecase instance.
func NewUsecase(analyzer *preemption.Analyzer) Usecase {
	if analyzer == nil {
		analyzer = preemption.NewAnalyzer()
	}
	return &usecaseImpl{analyzer: analyzer}
}

// GetDebugInfo performs orchestration: parsing -> tree building -> preemption analysis
func (u *usecaseImpl) GetDebugInfo(data []byte) (*DebugInfo, error) {
	// 1. Parse metadata using infrastructure layer
	wm, err := cromwell.ParseDetailedMetadata(data)
	if err != nil {
		return nil, err
	}

	// 2. Build call tree using TUI tree package
	root := tree.BuildTree(wm)
	visible := tree.GetVisibleNodes(root)

	// 3. Analyze preemption using domain layer
	callData := cromwell.ConvertToPreemptionCallData(wm.Calls)
	summary := u.analyzer.AnalyzeWorkflow(wm.ID, wm.Name, callData)

	return &DebugInfo{
		Metadata:   wm,
		Root:       root,
		Visible:    visible,
		Preemption: summary,
	}, nil
}
