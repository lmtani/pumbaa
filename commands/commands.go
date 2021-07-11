package commands

import "github.com/lmtani/cromwell-cli/pkg/cromwell"

type Commands struct {
	CromwellClient *cromwell.Client
}

func New(c *cromwell.Client) Commands {
	return Commands{CromwellClient: c}
}
