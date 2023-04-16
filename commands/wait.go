package commands

import (
	"fmt"
	"time"
)

func (c *Commands) Wait(operation string, sleep int) error {
	resp, err := c.CromwellClient.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", sleep)

	c.Writer.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := c.CromwellClient.Status(operation)
		if err != nil {
			return err
		}
		c.Writer.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
		status = resp.Status
	}

	return nil
}
