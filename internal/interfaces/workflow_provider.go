package interfaces

import "github.com/lmtani/pumbaa/internal/entities"

type WorkflowProvider interface {
	Get(uuid string) (entities.Workflow, error)
	Query() ([]entities.Workflow, error)
}
