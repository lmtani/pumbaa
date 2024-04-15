package cromwell

import (
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Cromwell struct {
	s ports.CromwellServer
	l ports.Logger
}

func NewCromwell(c ports.CromwellServer, l ports.Logger) *Cromwell {
	return &Cromwell{s: c, l: l}
}

func (c *Cromwell) SubmitWorkflow(wdl, inputs, dependencies, options string) (types.SubmitResponse, error) {
	return c.s.Submit(wdl, inputs, dependencies, options)
}

func (c *Cromwell) Inputs(operation string) (map[string]interface{}, error) {
	resp, err := c.s.Metadata(operation, &types.ParamsMetadataGet{})
	if err != nil {
		return nil, err
	}
	originalInputs := make(map[string]interface{})
	for k, v := range resp.Inputs {
		originalInputs[fmt.Sprintf("%s.%s", resp.WorkflowName, k)] = v
	}
	return originalInputs, nil
}

func (c *Cromwell) Kill(operation string) (types.SubmitResponse, error) {
	return c.s.Kill(operation)
}

func (c *Cromwell) Outputs(o string) (types.OutputsResponse, error) {
	return c.s.Outputs(o)
}

func (c *Cromwell) QueryWorkflow(name string, days time.Duration) (types.QueryResponse, error) {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := types.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	return c.s.Query(&params)
}

func (c *Cromwell) Metadata(o string) (types.MetadataResponse, error) {
	params := types.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}

	return c.s.Metadata(o, &params)
}

func (c *Cromwell) Wait(operation string, sleep int) error {
	resp, err := c.s.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", sleep)
	fmt.Printf("Status=%s\n", resp.Status)
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := c.s.Status(operation)
		if err != nil {
			return err
		}
		fmt.Printf("Status=%s\n", resp.Status)
		status = resp.Status
	}
	return nil
}

func (c *Cromwell) ResourceUsages(o string) (types.TotalResources, error) {
	m, err := c.s.Metadata(o, &types.ParamsMetadataGet{ExpandSubWorkflows: true})
	if err != nil {
		return types.TotalResources{}, err
	}

	if m.Status == "Running" {
		return types.TotalResources{}, err
	}

	rp := NewGCPResourceParser()
	return rp.GetComputeUsageForPricing(m.Calls)
}
