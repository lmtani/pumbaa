package commands

func SubmitWorkflow(c Client, w, i, d string) error {
	err := c.Submit(w, i, d)
	if err != nil {
		return err
	}
	return nil
}
