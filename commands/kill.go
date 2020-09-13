package commands

import "github.com/urfave/cli/v2"

func KillWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	resp, err := cromwellClient.Kill(c.String("operation"))
	if err != nil {
		return err
	}
	r := []string{resp.ID, resp.Status}
	CreateTable([]string{"Operation", "Status"}, [][]string{r})
	return nil
}
