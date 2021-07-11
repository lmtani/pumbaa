package commands

import (
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/writer"
)

type Commands struct {
	CromwellClient *cromwell.Client
	writer         writer.IWriter
}

func New(c *cromwell.Client, w writer.IWriter) Commands {
	return Commands{CromwellClient: c, writer: w}
}
