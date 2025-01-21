package usecase

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
)

// WorkflowGCEUsageInputDTO is the input data for the WorkflowGCEUsage usecase
type WorkflowGCEUsageInputDTO struct {
	WorkflowID string
}

// WorkflowGCEUsageOutputDTO is the output data for the WorkflowGCEUsage usecase
type WorkflowGCEUsageOutputDTO struct {
	PreemptHdd    float64
	PreemptSsd    float64
	PreemptCPU    float64
	PreemptMemory float64
	Hdd           float64
	Ssd           float64
	CPU           float64
	Memory        float64
	CachedCalls   int
	TotalTime     time.Duration
}

// WorkflowGCEUsage is a usecase that calculates the GCE usage of a workflow
type WorkflowGCEUsage struct {
	cromwellClient entities.CromwellServer
}

// NewWorkflowGCEUsage creates a new WorkflowGCEUsage usecase
func NewWorkflowGCEUsage(c entities.CromwellServer) *WorkflowGCEUsage {
	return &WorkflowGCEUsage{cromwellClient: c}
}

// Execute calculates the GCE usage of a workflow
func (w *WorkflowGCEUsage) Execute(i *WorkflowGCEUsageInputDTO) (*WorkflowGCEUsageOutputDTO, error) {
	var output WorkflowGCEUsageOutputDTO
	result, err := w.cromwellClient.Metadata(i.WorkflowID, &entities.ParamsMetadataGet{ExpandSubWorkflows: true})
	if err != nil {
		return nil, err
	}
	if result.Status == "Running" {
		return &output, err
	}

	w.iterateOverTasks(result.Calls, &output)
	return &output, nil
}

func (w *WorkflowGCEUsage) iterateOverTasks(data entities.CallItemSet, t *WorkflowGCEUsageOutputDTO) {
	for key := range data {
		w.iterateOverElements(data[key], t)
	}
}

func (w *WorkflowGCEUsage) iterateOverElement(call *entities.CallItem) (entities.ParsedCallAttributes, error) {
	runtimeAttrs := call.RuntimeAttributes
	size, diskType, err := w.parseDisk(runtimeAttrs)
	if err != nil {
		return entities.ParsedCallAttributes{}, err
	}
	totalSsd, totalHdd := w.calculateDiskSize(diskType, size)
	cpus, _ := strconv.ParseFloat(runtimeAttrs.CPU, 32)
	memory, _ := w.parseMemory(runtimeAttrs)

	parsed := entities.ParsedCallAttributes{
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

func (w *WorkflowGCEUsage) iterateOverElements(c []entities.CallItem, t *WorkflowGCEUsageOutputDTO) {
	HoursInMonth := 720.0

	for i := range c {
		item := &c[i] // pointer to avoid copying
		if item.SubWorkflowMetadata.RootWorkflowID != "" {
			w.iterateOverTasks(item.SubWorkflowMetadata.Calls, t)
			continue
		}

		parsed, _ := w.iterateOverElement(item) // ignoring error for now
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

func (w *WorkflowGCEUsage) parseDisk(runtimeAttrs entities.RuntimeAttributes) (float64, string, error) {
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

func (w *WorkflowGCEUsage) calculateDiskSize(diskType string, size float64) (float64, float64) {
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

func (w *WorkflowGCEUsage) parseMemory(runtimeAttrs entities.RuntimeAttributes) (float64, error) {
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
