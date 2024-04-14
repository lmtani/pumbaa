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
	w ports.Writer
}

func NewCromwell(c ports.CromwellServer, l ports.Logger, w ports.Writer) *Cromwell {
	return &Cromwell{s: c, w: w, l: l}
}

func (c *Cromwell) SubmitWorkflow(wdl, inputs, dependencies, options string) error {
	d, err := c.s.Submit(wdl, inputs, dependencies, options)
	if err != nil {
		return err
	}
	c.w.Message(fmt.Sprintf("🐖 Operation= %s , Status=%s", d.ID, d.Status))
	return nil
}

func (c *Cromwell) Inputs(operation string) error {
	resp, err := c.s.Metadata(operation, &types.ParamsMetadataGet{})
	if err != nil {
		return err
	}
	originalInputs := make(map[string]interface{})
	for k, v := range resp.Inputs {
		originalInputs[fmt.Sprintf("%s.%s", resp.WorkflowName, k)] = v
	}
	return c.w.Json(originalInputs)
}

func (c *Cromwell) Kill(operation string) error {
	d, err := c.s.Kill(operation)
	if err != nil {
		return err
	}
	c.w.Message(fmt.Sprintf("🐖 Operation=%s, Status=%s", d.ID, d.Status))
	return nil
}

func (c *Cromwell) Outputs(o string) error {
	d, err := c.s.Outputs(o)
	if err != nil {
		return err
	}
	return c.w.Json(d)
}

func (c *Cromwell) QueryWorkflow(name string, days time.Duration) error {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := types.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	d, err := c.s.Query(&params)
	if err != nil {
		return err
	}
	if len(d.Results) == 0 {
		c.w.Message("No results found")
		return nil
	}
	c.w.QueryTable(d)
	return nil
}

func (c *Cromwell) Metadata(o string) error {
	params := types.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}
	d, err := c.s.Metadata(o, &params)
	if err != nil {
		return err
	}
	return c.w.MetadataTable(d)
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

func (c *Cromwell) ResourceUsages(o string) error {
	m, err := c.s.Metadata(o, &types.ParamsMetadataGet{ExpandSubWorkflows: true})
	if err != nil {
		return err
	}

	if m.Status == "Running" {
		return err
	}

	rp := NewGCPResourceParser()
	d, err := rp.GetComputeUsageForPricing(m.Calls)
	if err != nil {
		return err
	}
	c.w.ResourceTable(d)
	return nil
}
