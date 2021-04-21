package commands

import (
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func KillWorkflow(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	resp, err := cromwellClient.Kill(operation)
	if err != nil {
		return err
	}
	fmt.Printf("Operation=%s, Status=%s", resp.ID, resp.Status)
	return nil
}
