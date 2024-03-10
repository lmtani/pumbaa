package core

import "github.com/lmtani/pumbaa/internal/ports"

type Kill struct {
	c ports.Cromwell
}

func NewKill(c ports.Cromwell) *Kill {
	return &Kill{c: c}
}

func (k *Kill) Kill(o string) error {
	_, err := k.c.Kill(o)
	if err != nil {
		return err
	}
	return nil
}
