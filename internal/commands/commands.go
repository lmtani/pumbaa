package commands

import (
	"os"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

var (
	defaultClient = cromwell.Default()
	defaultWriter = output.NewColoredWriter(os.Stdout)
)

type Commands struct {
	CromwellClient cromwell.Client
	Writer         output.Writer
}

func New() *Commands {
	return &Commands{
		CromwellClient: defaultClient,
		Writer:         defaultWriter,
	}
}
