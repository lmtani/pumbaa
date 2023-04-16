package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func (c *Commands) ResourcesUsed(operation string) error {
	resp, err := c.CromwellClient.Metadata(operation, &cromwell.ParamsMetadataGet{ExpandSubWorkflows: true})
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

func dashIfZero(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if v == 0.0 {
		s = "-"
	}
	return s
}

func GetComputeUsageForPricing(data cromwell.CallItemSet) (cromwell.TotalResources, error) {
	t := cromwell.TotalResources{}
	iterateOverTasks(data, &t)
	return t, nil
}

func iterateOverTasks(data cromwell.CallItemSet, t *cromwell.TotalResources) {
	for key := range data {
		iterateOverElements(data[key], t)
	}
}

func iterateOverElements(c []cromwell.CallItem, t *cromwell.TotalResources) {
	HoursInMonth := 720.0

	for i := range c {
		item := &c[i] // pointer to avoid copying
		if item.SubWorkflowMetadata.RootWorkflowID != "" {
			iterateOverTasks(item.SubWorkflowMetadata.Calls, t)
			continue
		}

		parsed, _ := iterateOverElement(item) // ignoring error for now
		if parsed.HitCache {
			t.CachedCalls++
			continue
		}

		timeElapsedInHours := parsed.Elapsed.Hours()

		if parsed.Preempt {
			t.PreemptHdd += (parsed.Hdd * timeElapsedInHours) / HoursInMonth
			t.PreemptSsd += (parsed.Ssd * timeElapsedInHours) / HoursInMonth
			t.PreemptMemory += parsed.Memory * timeElapsedInHours
			t.PreemptCPU += parsed.CPU * timeElapsedInHours
		} else {
			t.Hdd += (parsed.Hdd * timeElapsedInHours) / HoursInMonth
			t.Ssd += (parsed.Ssd * timeElapsedInHours) / HoursInMonth
			t.Memory += parsed.Memory * timeElapsedInHours
			t.CPU += parsed.CPU * timeElapsedInHours
		}

		t.TotalTime += parsed.Elapsed
	}
}

func iterateOverElement(call *cromwell.CallItem) (cromwell.ParsedCallAttributes, error) {
	runtimeAttrs := call.RuntimeAttributes
	size, diskType, err := parseDisk(runtimeAttrs)
	if err != nil {
		return cromwell.ParsedCallAttributes{}, err
	}
	totalSsd, totalHdd := calculateDiskSize(diskType, size)
	cpus, _ := strconv.ParseFloat(runtimeAttrs.CPU, 32)
	memory, _ := parseMemory(runtimeAttrs)

	parsed := cromwell.ParsedCallAttributes{
		Preempt:  runtimeAttrs.Preemptible != "0",
		Hdd:      totalHdd,
		Ssd:      totalSsd,
		HitCache: call.CallCaching.Hit,
		Memory:   memory,
		CPU:      cpus,
		Elapsed:  call.End.Sub(call.Start),
	}
	return parsed, nil
}

func calculateDiskSize(diskType string, size float64) (float64, float64) {
	totalSsd := 0.0
	totalHdd := 0.0
	switch diskType {
	case "SSD":
		totalSsd += size
	case "HDD":
		totalHdd += size
	}
	return totalSsd, totalHdd
}

func parseDisk(r cromwell.RuntimeAttributes) (float64, string, error) {
	workDisk := strings.Fields(r.Disks)
	if len(workDisk) == 0 {
		return 0, "", fmt.Errorf("no disks, found: %#v", r.Disks)
	}
	diskSize := workDisk[1]
	diskType := workDisk[2]
	size, err := strconv.ParseFloat(diskSize, 32)
	if err != nil {
		return 0, "", err
	}
	boot, err := strconv.ParseFloat(r.BootDiskSizeGb, 32)
	if err != nil {
		return 0, "", err
	}
	return size + boot, diskType, nil
}

func parseMemory(r cromwell.RuntimeAttributes) (float64, error) {
	memory := strings.Fields(r.Memory)
	if len(memory) == 0 {
		return 0, fmt.Errorf("no memory, found: %#v", r.Memory)
	}
	size, err := strconv.ParseFloat(memory[0], 32)
	if err != nil {
		return 0, err
	}
	return size, nil
}
