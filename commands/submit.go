package commands

import (
	"fmt"

	"go.uber.org/zap"
)

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
		zap.S().Fatalw(fmt.Sprintf("%s", err))
	}
	zap.S().Infow(fmt.Sprintf("Operation ID: %s", resp.ID))
	return nil
}
