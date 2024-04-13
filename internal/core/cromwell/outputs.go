package cromwell

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/pumbaa/internal/ports"
)

type Outputs struct {
	c ports.Cromwell
}

func NewOutputs(c ports.Cromwell) *Outputs {
	return &Outputs{c: c}
}

func (out *Outputs) Outputs(o string) error {
	resp, err := out.c.Outputs(o)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(resp.Outputs, "", "   ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return err
}
