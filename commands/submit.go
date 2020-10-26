package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

type SubmitResponse struct {
	ID     string
	Status string
}

type SubmitRequest struct {
	workflowSource       string
	workflowInputs       string
	workflowDependencies string
	workflowOptions      string
}

func SubmitWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	r := SubmitRequest{
		workflowSource:       c.String("wdl"),
		workflowInputs:       c.String("inputs"),
		workflowDependencies: c.String("dependencies"),
		workflowOptions:      c.String("options")}
	resp, err := cromwellClient.Submit(r)
	if err != nil {
		return err
	}
	zap.S().Info(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
}
