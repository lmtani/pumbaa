package commands

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

type OutputsResponse struct {
	ID      string
	Outputs map[string]interface{}
}

func OutputsWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Outputs(c.String("operation"))
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
