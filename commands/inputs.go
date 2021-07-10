package commands

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) Inputs(host, iap, operation string) error {
	cromwellClient := cromwell.New(host, iap)
	params := url.Values{}
	resp, err := cromwellClient.Metadata(operation, params)
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
