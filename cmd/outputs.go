package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell_client"
)

func OutputsWorkflow(operation string, c *cromwell_client.Client) error {
	resp, err := c.Outputs(operation)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(resp.Outputs, "", "   ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return err
}
