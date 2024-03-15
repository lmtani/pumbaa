package core

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/ports"
)

type Wait struct {
	c ports.Cromwell
	w ports.Writer
}

func NewWait(c ports.Cromwell, w ports.Writer) *Wait {
	return &Wait{c: c, w: w}
}

func (wt *Wait) Wait(operation string, sleep int) error {
	resp, err := wt.c.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", sleep)
	wt.w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := wt.c.Status(operation)
		if err != nil {
			return err
		}
		wt.w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
		status = resp.Status
	}

	return nil
}
