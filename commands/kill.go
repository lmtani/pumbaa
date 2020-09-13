package commands

func KillWorkflow(c Client, operation string) error {
	resp, err := c.Kill(operation)
	if err != nil {
		return err
	}

	r := []string{resp.ID, resp.Status}
	CreateTable([]string{"Operation", "Status"}, [][]string{r})
	return nil
}
