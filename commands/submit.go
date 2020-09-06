package commands

import (
	"fmt"

	"go.uber.org/zap"
)

type SubmitResponse struct {
	ID     string
	Status string
}

func SubmitWorkflow(c Client, w, i string) error {
	resp, err := c.Submit(w, i)
	if err != nil {
		zap.S().Fatalw(fmt.Sprintf("%s", err))
	}
	zap.S().Infow(fmt.Sprintf("Operation ID: %s", resp.ID))
	return nil
}
