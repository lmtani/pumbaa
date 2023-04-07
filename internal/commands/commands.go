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

type Commands struct {
	CromwellClient cromwell.Client
	Prompt         prompt.Prompt
	Writer         output.Writer
}

func New() *Commands {
	return &Commands{
		CromwellClient: defaultClient,
		Prompt:         defaultPrompt,
		Writer:         defaultWriter,
	}
}
