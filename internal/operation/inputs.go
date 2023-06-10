package operation

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func Inputs(operation string, c *cromwell_client.Client) error {
	resp, err := c.Metadata(operation, &cromwell_client.ParamsMetadataGet{})
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
