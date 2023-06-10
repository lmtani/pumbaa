package job

import (
	"github.com/lmtani/pumbaa/internal/pkg/output"
)

type Prompt interface {
	SelectByKey(taskOptions []string) (string, error)
	SelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error)
}

type Writer interface {
	Primary(string)
	Accent(string)
	Error(string)
	Table(output.Table)
}
