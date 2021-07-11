package commands

import (
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

type Commands struct {
	CromwellClient *cromwell.Client
	writer         output.IWriter
}

func New(c *cromwell.Client, w output.IWriter) Commands {
	return Commands{CromwellClient: c, writer: w}
}
