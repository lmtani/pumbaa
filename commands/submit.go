package commands

type SubmitResponse struct {
	ID     string
	Status string
}

type SubmitRequest struct {
	workflowSource       string
	workflowInputs       string
	workflowDependencies string
}

func SubmitWorkflow(c Client, w, i, d string) error {
	r := SubmitRequest{workflowSource: w, workflowInputs: i, workflowDependencies: d}
	resp, err := c.Submit(r)
	if err != nil {
		return err
	}
	rows := []string{resp.ID, resp.Status}
	CreateTable([]string{"Operation", "Status"}, [][]string{rows})
	return nil
}
