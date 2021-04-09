package commands

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func SubmitWorkflow(c *cli.Context) error {
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
	r := cromwell.SubmitRequest{
		WorkflowSource:       c.String("wdl"),
		WorkflowInputs:       c.String("inputs"),
		WorkflowDependencies: c.String("dependencies"),
		WorkflowOptions:      c.String("options")}
	resp, err := cromwellClient.Submit(r)
	if err != nil {
		return err
	}
	zap.S().Info(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
}
