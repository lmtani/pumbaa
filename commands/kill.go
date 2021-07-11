package commands

import (
	"fmt"
)

func (c *Commands) KillWorkflow(operation string) error {
	resp, err := c.CromwellClient.Kill(operation)
	if err != nil {
		return err
	}
	fmt.Printf("Operation=%s, Status=%s", resp.ID, resp.Status)
	return nil
}
