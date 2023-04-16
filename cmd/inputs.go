package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) Inputs(operation string) error {
	resp, err := c.CromwellClient.Metadata(operation, &cromwell.ParamsMetadataGet{})
	if err != nil {
		return err
	}
	originalInputs := make(map[string]interface{})
	for k, v := range resp.Inputs {
		originalInputs[fmt.Sprintf("%s.%s", resp.WorkflowName, k)] = v
	}

	b, err := json.MarshalIndent(originalInputs, "", "   ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
