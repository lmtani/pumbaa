package commands

import (
	"fmt"

	"go.uber.org/zap"
)

func KillWorkflow(c Client, operation string) error {
	resp, err := c.Kill(operation)
	if err != nil {
		return err
	}
	zap.S().Infow(fmt.Sprintf("%s - %s", resp.ID, resp.Status))
	return nil
}
