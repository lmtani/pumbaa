package core

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Kill struct {
	c ports.Cromwell
	w ports.Writer
}

func NewKill(c ports.Cromwell, w ports.Writer) *Kill {
	return &Kill{c: c, w: w}
}

func (k *Kill) Kill(o string) (types.SubmitResponse, error) {
	resp, err := k.c.Kill(o)
	if err != nil {
		return types.SubmitResponse{}, err
	}
	k.w.Accent(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return resp, nil
}
