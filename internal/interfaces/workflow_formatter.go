package interfaces

import (
	"github.com/lmtani/pumbaa/internal/entities"
)

type WorkflowFormatter interface {
	Report(workflow *entities.Workflow) error
	Query(workflows []entities.Workflow) error
}
