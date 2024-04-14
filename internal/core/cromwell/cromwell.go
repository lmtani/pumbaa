package cromwell

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type Cromwell struct {
	s ports.CromwellServer
	w ports.Writer
}

func NewCromwell(c ports.CromwellServer, w ports.Writer) *Cromwell {
	return &Cromwell{s: c, w: w}
}

func (c *Cromwell) SubmitWorkflow(wdl, inputs, dependencies, options string) error {
	r := types.SubmitRequest{
		WorkflowSource:       wdl,
		WorkflowInputs:       inputs,
		WorkflowDependencies: dependencies,
		WorkflowOptions:      options}
	resp, err := c.s.Submit(&r)
	if err != nil {
		return err
	}
	c.w.Accent(fmt.Sprintf("🐖 Operation= %s , Status=%s", resp.ID, resp.Status))
	return nil
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

	b, err := json.MarshalIndent(originalInputs, "", "   ")
	if err != nil {
		return nil, err
	}
	fmt.Println(string(b))
	return originalInputs, nil
}

func (c *Cromwell) Kill(operation string) (types.SubmitResponse, error) {
	resp, err := c.s.Kill(operation)
	if err != nil {
		return types.SubmitResponse{}, err
	}
	c.w.Accent(fmt.Sprintf("Operation=%s, Status=%s", resp.ID, resp.Status))
	return resp, nil
}

func (c *Cromwell) Outputs(o string) error {
	resp, err := c.s.Outputs(o)
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

func (c *Cromwell) QueryWorkflow(name string, days time.Duration) error {
	var submission time.Time
	if days != 0 {
		submission = time.Now().Add(-time.Hour * 24 * days)
	}
	params := types.ParamsQueryGet{
		Submission: submission,
		Name:       name,
	}
	resp, err := c.s.Query(&params)
	if err != nil {
		return err
	}
	var qtr = types.QueryTableResponse(resp)

	c.w.Table(qtr)
	c.w.Accent(fmt.Sprintf("- Found %d workflows", resp.TotalResultsCount))
	return err
}

func (c *Cromwell) Wait(operation string, sleep int) error {
	resp, err := c.s.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Time between status check = %d\n", sleep)
	c.w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := c.s.Status(operation)
		if err != nil {
			return err
		}
		c.w.Accent(fmt.Sprintf("Status=%s\n", resp.Status))
		status = resp.Status
	}
	return nil
}

func (c *Cromwell) Get(o string) error {
	m, err := c.s.Metadata(o, &types.ParamsMetadataGet{ExpandSubWorkflows: true})
	if err != nil {
		c.w.Error(err.Error())
		return err
	}

	if m.Status == "Running" {
		c.w.Error("workflow status is still running")
		return err
	}

	rp := NewGCPResourceParser()
	total, err := rp.GetComputeUsageForPricing(m.Calls)
	if err != nil {
		c.w.Error(err.Error())
		return err
	}

	var rtr = types.ResourceTableResponse{Total: total}
	c.w.Table(rtr)
	c.w.Accent(fmt.Sprintf("- Tasks with cache hit: %d", total.CachedCalls))
	c.w.Accent(fmt.Sprintf("- Total time with running VMs: %.0fh", total.TotalTime.Hours()))
	return nil
}
