package job

import (
	"fmt"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func KillWorkflow(operation string, c *cromwell_client.Client, w Writer) error {
	resp, err := c.Kill(operation)
	if err != nil {
		return err
	}
	w.Accent(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return nil
}
