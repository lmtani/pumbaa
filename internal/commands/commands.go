package commands

import (
	"os"

	"github.com/lmtani/cromwell-cli/internal/prompt"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

var (
	defaultClient = cromwell.Default()
	defaultWriter = output.NewColoredWriter(os.Stdout)
	defaultPrompt = prompt.New()
)

type Prompt interface {
	SelectByKey(taskOptions []string) (string, error)
	SelectByIndex(t prompt.TemplateOptions, sfn func(input string, index int) bool, items interface{}) (int, error)
}

type Writer interface {
	Primary(string)
	Accent(string)
	Error(string)
	Table(output.Table)
}

type Commands struct {
	CromwellClient cromwell.Client
	Prompt         Prompt
	Writer         Writer
}

func New() *Commands {
	return &Commands{
		CromwellClient: defaultClient,
		Prompt:         defaultPrompt,
		Writer:         defaultWriter,
	}
}
