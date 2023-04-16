package cmd

import (
	"os"

	"github.com/lmtani/cromwell-cli/pkg/cromwell_client"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

var (
	defaultClient = cromwell_client.Default()
	defaultWriter = output.NewColoredWriter(os.Stdout)
)

type Commands struct {
	CromwellClient cromwell_client.Client
	Writer         output.Writer
}

func New() *Commands {
	return &Commands{
		CromwellClient: defaultClient,
		Writer:         defaultWriter,
	}
}
