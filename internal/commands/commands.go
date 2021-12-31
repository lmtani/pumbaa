package commands

import (
	"log"
	"os"

	"github.com/lmtani/cromwell-cli/internal/prompt"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

var (
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags)
	defaultClient = cromwell.Default()
	defaultWriter = output.NewColoredWriter()
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
}

type Commands struct {
	CromwellClient cromwell.Client
	Logger         *log.Logger
	Prompt         Prompt
	Writer         Writer
}

func New() *Commands {
	return &Commands{
		CromwellClient: defaultClient,
		Logger:         defaultLogger,
		Prompt:         defaultPrompt,
		Writer:         defaultWriter,
	}
}
