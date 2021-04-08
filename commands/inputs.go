package commands

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
)

func Inputs(c *cli.Context) error {
	cromwellClient := cromwell.FromInterface(c.Context.Value("cromwell"))
	params := url.Values{}
	resp, err := cromwellClient.Metadata(c.String("operation"), params)
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
