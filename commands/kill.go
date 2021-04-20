package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func KillWorkflow(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	resp, err := cromwellClient.Kill(operation)
	if err != nil {
		return err
	}
	color.Cyan(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return nil
}
