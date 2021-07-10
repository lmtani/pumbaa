package commands

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func (c *Commands) ResourcesUsed(host, iap, operation string) error {
	params := url.Values{}
	params.Add("expandSubWorkflows", "true")
	cromwellClient := cromwell.New(host, iap)
	resp, err := cromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	if resp.Status == "Running" {
		return errors.New("Workflow status is still running")
	}
	total, err := cromwell.GetComputeUsageForPricing(resp.Calls)
	if err != nil {
		return err
	}
	var rtr = ResourceTableResponse{Total: total}
	output.NewTable(os.Stdout).Render(rtr)
	color.Cyan(fmt.Sprintf("- Tasks with cache hit: %d", total.CachedCalls))
	color.Cyan(fmt.Sprintf("- Total time with running VMs: %.0fh", total.TotalTime.Hours()))
	return nil
}

func (rtr ResourceTableResponse) Header() []string {
	return []string{"Resource", "Normalized to", "Preemptive", "Normal"}
}

func (rtr ResourceTableResponse) Rows() [][]string {
	rows := make([][]string, 4)

	rows = append(rows, []string{
		"CPUs",
		"1 hour",
		dashIfZero(rtr.Total.PreemptCPU),
		dashIfZero(rtr.Total.CPU),
	})

	rows = append(rows, []string{
		"Memory (GB)",
		"1 hour",
		dashIfZero(rtr.Total.PreemptMemory),
		dashIfZero(rtr.Total.Memory),
	})

	rows = append(rows, []string{
		"HDD disk (GB)",
		"1 month",
		dashIfZero(rtr.Total.PreemptHdd),
		dashIfZero(rtr.Total.Hdd),
	})
	rows = append(rows, []string{
		"SSD disk (GB)",
		"1 month",
		dashIfZero(rtr.Total.PreemptSsd),
		dashIfZero(rtr.Total.Ssd),
	})
	return rows
}

func dashIfZero(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if v == 0.0 {
		s = "-"
	}
	return s
}
