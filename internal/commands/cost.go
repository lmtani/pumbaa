package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) ResourcesUsed(operation string) error {
	params := cromwell.ParamsMetadataGet{
		ExpandSubWorkflows: true,
	}
	resp, err := c.CromwellClient.Metadata(operation, params)
	if err != nil {
		return err
	}
	if resp.Status == "Running" {
		return errors.New("workflow status is still running")
	}
	total, err := GetComputeUsageForPricing(resp.Calls)
	if err != nil {
		return err
	}
	var rtr = ResourceTableResponse{Total: total}
	c.Writer.Table(rtr)
	c.Writer.Accent(fmt.Sprintf("- Tasks with cache hit: %d", total.CachedCalls))
	c.Writer.Accent(fmt.Sprintf("- Total time with running VMs: %.0fh", total.TotalTime.Hours()))
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

func GetComputeUsageForPricing(data map[string][]cromwell.CallItem) (cromwell.TotalResources, error) {
	t := cromwell.TotalResources{}
	iterateOverTasks(data, &t)
	return t, nil
}

func iterateOverTasks(data map[string][]cromwell.CallItem, t *cromwell.TotalResources) {
	for key := range data {
		iterateOverElements(data[key], t)
	}
}

func iterateOverElements(c []cromwell.CallItem, t *cromwell.TotalResources) {
	for idx := range c {
		if c[idx].SubWorkflowMetadata.RootWorkflowID != "" {
			iterateOverTasks(c[idx].SubWorkflowMetadata.Calls, t)
		} else {
			parsed, _ := iterateOverElement(c[idx])
			HoursInMonth := 720.0
			if parsed.HitCache {
				t.CachedCalls++
				continue
			}
			if parsed.Preempt {
				t.PreemptHdd += (parsed.Hdd * parsed.Elapsed.Hours()) / HoursInMonth
				t.PreemptSsd += (parsed.Ssd * parsed.Elapsed.Hours()) / HoursInMonth
				t.PreemptMemory += parsed.Memory * parsed.Elapsed.Hours()
				t.PreemptCPU += parsed.CPU * parsed.Elapsed.Hours()
			} else {
				t.Hdd += (parsed.Hdd * parsed.Elapsed.Hours()) / HoursInMonth
				t.Ssd += (parsed.Ssd * parsed.Elapsed.Hours()) / HoursInMonth
				t.Memory += parsed.Memory * parsed.Elapsed.Hours()
				t.CPU += parsed.CPU * parsed.Elapsed.Hours()
			}
			t.TotalTime += parsed.Elapsed
		}
	}
}

func iterateOverElement(call cromwell.CallItem) (cromwell.ParsedCallAttributes, error) {
	size, diskType, err := parseDisc(call.RuntimeAttributes)
	if err != nil {
		return cromwell.ParsedCallAttributes{}, err
	}
	totalSsd := 0.0
	if diskType == "SSD" {
		totalSsd += size
	}
	totalHdd := 0.0
	if diskType == "HDD" {
		totalHdd += size
	}
	nproc, _ := strconv.ParseFloat(call.RuntimeAttributes.CPU, 4)
	memory, err := parseMemory(call.RuntimeAttributes)
	if err != nil {
		color.Yellow(fmt.Sprintf("Task %s returned %s", call.Labels, err))
	}
	elapsed := call.End.Sub(call.Start)
	isPreempt := call.RuntimeAttributes.Preemptible != "0"
	parsed := cromwell.ParsedCallAttributes{
		Preempt:  isPreempt,
		Hdd:      totalHdd,
		Ssd:      totalSsd,
		HitCache: call.CallCaching.Hit,
		Memory:   memory,
		CPU:      nproc,
		Elapsed:  elapsed}
	return parsed, nil
}

func parseDisc(r cromwell.RuntimeAttributes) (float64, string, error) {
	workDisk := strings.Fields(r.Disks)
	if len(workDisk) == 0 {
		return 0, "", fmt.Errorf("No disks, found: %#v", r.Disks)
	}
	diskSize := workDisk[1]
	diskType := workDisk[2]
	size, err := strconv.ParseFloat(diskSize, 4)
	if err != nil {
		return 0, "", err
	}
	boot, err := strconv.ParseFloat(r.BootDiskSizeGb, 8)
	if err != nil {
		return 0, "", err
	}
	return size + boot, diskType, nil
}

func parseMemory(r cromwell.RuntimeAttributes) (float64, error) {
	memmory := strings.Fields(r.Memory)
	if len(memmory) == 0 {
		return 0, fmt.Errorf("No memory, found: %#v", r.Memory)
	}
	size, err := strconv.ParseFloat(memmory[0], 4)
	if err != nil {
		return 0, err
	}
	return size, nil
}
