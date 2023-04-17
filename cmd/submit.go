package cmd

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell_client"
)

func SubmitWorkflow(wdl, inputs, dependencies, options string, c *cromwell_client.Client, w Writer) error {
	r := cromwell_client.SubmitRequest{
		WorkflowSource:       wdl,
		WorkflowInputs:       inputs,
		WorkflowDependencies: dependencies,
		WorkflowOptions:      options}
	resp, err := c.Submit(&r)
	if err != nil {
		return err
	}
	w.Accent(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
}
