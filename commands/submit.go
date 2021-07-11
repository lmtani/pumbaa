package commands

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) SubmitWorkflow(wdl, inputs, dependencies, options string) error {
	r := cromwell.SubmitRequest{
		WorkflowSource:       wdl,
		WorkflowInputs:       inputs,
		WorkflowDependencies: dependencies,
		WorkflowOptions:      options}
	resp, err := c.CromwellClient.Submit(r)
	if err != nil {
		return err
	}
	c.writer.Accent(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
}
