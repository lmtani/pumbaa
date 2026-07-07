package dashboard

import (
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// Diff messages (workflow comparison)
type diffLoadedMsg struct {
	diff *workflow.RunDiff
}

type diffErrorMsg struct {
	err error
}

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

// Health status messages
type healthStatusLoadedMsg struct {
	status *workflow.HealthStatus
}

type healthStatusErrorMsg struct {
	err error
}

// Labels messages
type labelsLoadedMsg struct {
	labels map[string]string
}

type labelsErrorMsg struct {
	err error
}

type labelsUpdatedMsg struct {
	success bool
	err     error
}

// clearStatusMsg clears the temporary status message after expiry.
type clearStatusMsg struct{}

// tickMsg is for periodic health checks
type tickMsg struct{}
