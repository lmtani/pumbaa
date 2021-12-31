package commands

import (
	"fmt"
	"time"

	"github.com/martinlindhe/notify"
)

func (c *Commands) Wait(operation string, sleep int, alarm bool) error {
	resp, err := c.CromwellClient.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Status=%s\n", resp.Status)

	fmt.Printf("Time between status check = %d\n", sleep)
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := c.CromwellClient.Status(operation)
		if err != nil {
			return err
		}
		c.Writer.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
		status = resp.Status
	}

	if alarm {
		notify.Alert("🐖 Cromwell Cli", "alert", "Your workflow ended", "")
	}
	return nil
}
