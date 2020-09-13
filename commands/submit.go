package commands

import "github.com/urfave/cli/v2"

type SubmitResponse struct {
	ID     string
	Status string
}

type SubmitRequest struct {
	workflowSource       string
	workflowInputs       string
	workflowDependencies string
}

func SubmitWorkflow(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	r := SubmitRequest{workflowSource: c.String("wdl"), workflowInputs: c.String("inputs"), workflowDependencies: c.String("dependencies")}
	resp, err := cromwellClient.Submit(r)
	if err != nil {
		return err
	}
	rows := []string{resp.ID, resp.Status}
	CreateTable([]string{"Operation", "Status"}, [][]string{rows})
	return nil
}
