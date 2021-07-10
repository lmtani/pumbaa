package commands

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) SubmitWorkflow(host, iap, wdl, inputs, dependencies, options string) error {
	cromwellClient := cromwell.New(host, iap)
	r := cromwell.SubmitRequest{
		WorkflowSource:       wdl,
		WorkflowInputs:       inputs,
		WorkflowDependencies: dependencies,
		WorkflowOptions:      options}
	resp, err := cromwellClient.Submit(r)
	if err != nil {
		return err
	}
	fmt.Printf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status)
	return nil
}
