package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Commands) Inputs(operation string) error {
	params := url.Values{}
	resp, err := c.CromwellClient.Metadata(operation, params)
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
