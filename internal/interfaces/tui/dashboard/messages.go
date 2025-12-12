package dashboard

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// Messages for dashboard model

type workflowsLoadedMsg struct {
	workflows  []workflow.Workflow
	totalCount int
}

type workflowsErrorMsg struct {
	err error
}

type abortResultMsg struct {
	success bool
	id      string
	err     error
}

type debugMetadataLoadedMsg struct {
	workflowID string
	metadata   []byte
}

type debugMetadataErrorMsg struct {
	err error
}

// NavigateToDebugMsg is sent when user wants to open debug view
type NavigateToDebugMsg struct {
	WorkflowID string
}
