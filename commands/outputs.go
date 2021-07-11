package commands

import (
	"encoding/json"
	"fmt"
)

func (c *Commands) OutputsWorkflow(operation string) error {
	resp, err := c.CromwellClient.Outputs(operation)
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
