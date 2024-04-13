package cromwell

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

type ResourcesUsed struct {
	c ports.Cromwell
	w ports.Writer
}

func NewResourcesUsed(c ports.Cromwell, w ports.Writer) *ResourcesUsed {
	return &ResourcesUsed{c: c, w: w}
}

func (r *ResourcesUsed) Get(o string) error {
	m, err := r.c.Metadata(o, &types.ParamsMetadataGet{ExpandSubWorkflows: true})
	if err != nil {
		r.w.Error(err.Error())
		return err
	}

	if m.Status == "Running" {
		r.w.Error("workflow status is still running")
		return err
	}

	total, err := r.GetComputeUsageForPricing(m.Calls)
	if err != nil {
		r.w.Error(err.Error())
		return err
	}

	var rtr = types.ResourceTableResponse{Total: total}
	r.w.Table(rtr)
	r.w.Accent(fmt.Sprintf("- Tasks with cache hit: %d", total.CachedCalls))
	r.w.Accent(fmt.Sprintf("- Total time with running VMs: %.0fh", total.TotalTime.Hours()))
	return nil
}

func (r *ResourcesUsed) GetComputeUsageForPricing(data types.CallItemSet) (types.TotalResources, error) {
	t := types.TotalResources{}
	r.iterateOverTasks(data, &t)
	return t, nil
}

func (r *ResourcesUsed) iterateOverTasks(data types.CallItemSet, t *types.TotalResources) {
	for key := range data {
		r.iterateOverElements(data[key], t)
	}
}

func (r *ResourcesUsed) iterateOverElement(call *types.CallItem) (types.ParsedCallAttributes, error) {
	runtimeAttrs := call.RuntimeAttributes
	size, diskType, err := r.parseDisk(runtimeAttrs)
	if err != nil {
		return types.ParsedCallAttributes{}, err
	}
	totalSsd, totalHdd := r.calculateDiskSize(diskType, size)
	cpus, _ := strconv.ParseFloat(runtimeAttrs.CPU, 32)
	memory, _ := r.parseMemory(runtimeAttrs)

	parsed := types.ParsedCallAttributes{
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

func (r *ResourcesUsed) iterateOverElements(c []types.CallItem, t *types.TotalResources) {
	HoursInMonth := 720.0

	for i := range c {
		item := &c[i] // pointer to avoid copying
		if item.SubWorkflowMetadata.RootWorkflowID != "" {
			r.iterateOverTasks(item.SubWorkflowMetadata.Calls, t)
			continue
		}

		parsed, _ := r.iterateOverElement(item) // ignoring error for now
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

func (r *ResourcesUsed) parseDisk(runtimeAttrs types.RuntimeAttributes) (float64, string, error) {
	workDisk := strings.Fields(runtimeAttrs.Disks)
	if len(workDisk) == 0 {
		return 0, "", fmt.Errorf("no disks, found: %#v", runtimeAttrs.Disks)
	}
	diskSize := workDisk[1]
	diskType := workDisk[2]
	size, err := strconv.ParseFloat(diskSize, 32)
	if err != nil {
		return 0, "", err
	}
	boot, err := strconv.ParseFloat(runtimeAttrs.BootDiskSizeGb, 32)
	if err != nil {
		return 0, "", err
	}
	return size + boot, diskType, nil
}

func (r *ResourcesUsed) calculateDiskSize(diskType string, size float64) (float64, float64) {
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

func (r *ResourcesUsed) parseMemory(runtimeAttrs types.RuntimeAttributes) (float64, error) {
	memory := strings.Fields(runtimeAttrs.Memory)
	if len(memory) == 0 {
		return 0, fmt.Errorf("no memory, found: %#v", runtimeAttrs.Memory)
	}
	size, err := strconv.ParseFloat(memory[0], 32)
	if err != nil {
		return 0, err
	}
	return size, nil
}
