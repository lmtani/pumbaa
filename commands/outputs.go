package commands

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/urfave/cli/v2"
)

func OutputsWorkflow(c *cli.Context) error {
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
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
