package job

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func Wait(operation string, sleep int, c *cromwell_client.Client, w Writer) error {
	resp, err := c.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", sleep)

	w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := c.Status(operation)
		if err != nil {
			return err
		}
		w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
		status = resp.Status
	}

	return nil
}
