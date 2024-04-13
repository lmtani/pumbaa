package cromwell

import (
	"encoding/json"
	"fmt"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Inputs struct {
	c ports.Cromwell
}

func NewInputs(c ports.Cromwell) *Inputs {
	return &Inputs{c: c}
}

func (i *Inputs) Inputs(operation string) (map[string]interface{}, error) {
	resp, err := i.c.Metadata(operation, &types.ParamsMetadataGet{})
	if err != nil {
		return nil, err
	}
	originalInputs := make(map[string]interface{})
	for k, v := range resp.Inputs {
		originalInputs[fmt.Sprintf("%s.%s", resp.WorkflowName, k)] = v
	}

	b, err := json.MarshalIndent(originalInputs, "", "   ")
	if err != nil {
		return nil, err
	}
	fmt.Println(string(b))
	return originalInputs, nil
}
