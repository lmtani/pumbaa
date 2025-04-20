package interfaces

import "github.com/lmtani/pumbaa/internal/entities"

type WorkflowProvider interface {
	Get(uuid string, expandSubworkflow bool) (entities.Workflow, error)
	Query() ([]entities.Workflow, error)
}
