package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func MetadataWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return err
}
