package commands

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) OutputsWorkflow(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	resp, err := cromwellClient.Outputs(operation)
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
