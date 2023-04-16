package cmd

import (
	"fmt"
)

func (c *Commands) KillWorkflow(operation string) error {
	resp, err := c.CromwellClient.Kill(operation)
	if err != nil {
		return err
	}
	c.Writer.Accent(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return nil
}
