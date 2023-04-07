package app

import (
	"github.com/lmtani/cromwell-cli/internal/util"
	"github.com/urfave/cli/v2"
)

func localDeploy() *cli.Command {
	return &cli.Command{
		Name:    "local-deploy",
		Aliases: []string{"ld"},
		Usage:   "Install Cromwell Server locally with default configuration and start it",
		Action: func(c *cli.Context) error {
			return util.StartCromwellServer()
		},
	}
}
