package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
)

func KillWorkflow(c *cli.Context) error {
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
	resp, err := cromwellClient.Kill(c.String("operation"))
	if err != nil {
		return err
	}
	color.Cyan(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return nil
}
