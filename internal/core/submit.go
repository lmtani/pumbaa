package core

import (
	"fmt"
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Submit struct {
	c ports.Cromwell
	w ports.Writer
}

func NewSubmit(c ports.Cromwell, w ports.Writer) *Submit {
	return &Submit{c: c, w: w}
}

func (s *Submit) SubmitWorkflow(wdl, inputs, dependencies, options string) error {
	r := types.SubmitRequest{
		WorkflowSource:       wdl,
		WorkflowInputs:       inputs,
		WorkflowDependencies: dependencies,
		WorkflowOptions:      options}
	resp, err := s.c.Submit(&r)
	if err != nil {
		return err
	}
	s.w.Accent(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
}
