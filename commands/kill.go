package commands

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func KillWorkflow(c *cli.Context) error {
	cromwellClient := cromwell.FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Kill(c.String("operation"))
	if err != nil {
		return err
	}
	zap.S().Info(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return nil
}
