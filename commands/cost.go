package commands

import (
	"strconv"
	"strings"
	"time"
)

type ParsedCallAttributes struct {
	Hdd     float64
	Preempt bool
	Ssd     float64
	Memory  float64
	CPU     float64
	Elapsed time.Duration
}

type TotalResources struct {
	PreemptHdd    float64
	PreemptSsd    float64
	PreemptCPU    float64
	PreemptMemory float64
	Hdd           float64
	Ssd           float64
	CPU           float64
	Memory        float64
}

func GetComputeUsageForPricing(data map[string][]CallItem) (TotalResources, error) {
	t := TotalResources{}
	iterateOverTasks(data, &t)
	return t, nil
}

func iterateOverTasks(data map[string][]CallItem, t *TotalResources) {
	for key := range data {
		iterateOverElements(data[key], t)
	}
}

func iterateOverElements(c []CallItem, t *TotalResources) {
	for idx := range c {
		if c[idx].SubWorkflowMetadata.RootWorkflowID != "" {
			iterateOverTasks(c[idx].SubWorkflowMetadata.Calls, t)
		} else {
			parsed, _ := iterateOverElement(c[idx])
			if parsed.Preempt {
				t.PreemptHdd += parsed.Hdd
				t.PreemptSsd += parsed.Ssd
				t.PreemptMemory += parsed.Memory * parsed.Elapsed.Hours()
				t.PreemptCPU += parsed.CPU * parsed.Elapsed.Hours()
			} else {
				t.Hdd += parsed.Hdd
				t.Ssd += parsed.Ssd
				t.Memory += parsed.Memory * parsed.Elapsed.Hours()
				t.CPU += parsed.CPU * parsed.Elapsed.Hours()
			}
		}
	}
}

func iterateOverElement(call CallItem) (ParsedCallAttributes, error) {
	size, diskType, err := parseDisc(call)
	if err != nil {
		return ParsedCallAttributes{}, err
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
	memory, err := parseMemory(call)
	if err != nil {
		return ParsedCallAttributes{}, err
	}
	elapsed := call.End.Sub(call.Start)
	isPreempt := call.RuntimeAttributes.Preemptible != "0"
	parsed := ParsedCallAttributes{
		Preempt: isPreempt,
		Hdd:     totalHdd,
		Ssd:     totalSsd,
		Memory:  memory,
		CPU:     nproc,
		Elapsed: elapsed}
	return parsed, nil
}

func normalizeUsePerHour(a float64, e time.Duration) float64 {
	hoursPerCPU := a * e.Hours()
	return hoursPerCPU
}

func parseDisc(c CallItem) (float64, string, error) {
	workDisk := strings.Fields(c.RuntimeAttributes.Disks)
	diskSize := workDisk[1]
	diskType := workDisk[2]
	size, err := strconv.ParseFloat(diskSize, 4)
	if err != nil {
		return 0, "", err
	}
	boot, err := strconv.ParseFloat(c.RuntimeAttributes.BootDiskSizeGb, 8)
	if err != nil {
		return 0, "", err
	}
	return size + boot, diskType, nil
}

func parseMemory(c CallItem) (float64, error) {
	memmory := strings.Fields(c.RuntimeAttributes.Memory)
	size, err := strconv.ParseFloat(memmory[0], 4)
	if err != nil {
		return 0, err
	}
	return size, nil
}
