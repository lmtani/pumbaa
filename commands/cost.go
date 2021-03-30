package commands

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

type ResourceTableResponse struct {
	Total TotalResources
}

func (rtr ResourceTableResponse) Header() []string {
	return []string{"Resource", "Normalized to", "Preemptive", "Normal"}
}

func (rtr ResourceTableResponse) Rows() [][]string {
	rows := make([][]string, 4)

	rows = append(rows, []string{
		"CPUs",
		"1 hour",
		fmt.Sprintf("%.2f", rtr.Total.PreemptCPU),
		fmt.Sprintf("%.2f", rtr.Total.CPU),
	})

	rows = append(rows, []string{
		"Memory (GB)",
		"1 hour",
		fmt.Sprintf("%.2f", rtr.Total.PreemptMemory),
		fmt.Sprintf("%.2f", rtr.Total.Memory),
	})

	rows = append(rows, []string{
		"HDD disk (GB)",
		"1 month",
		fmt.Sprintf("%.2f", rtr.Total.PreemptHdd),
		fmt.Sprintf("%.2f", rtr.Total.Hdd),
	})
	rows = append(rows, []string{
		"SSD disk (GB)",
		"1 month",
		fmt.Sprintf("%.2f", rtr.Total.PreemptSsd),
		fmt.Sprintf("%.2f", rtr.Total.Ssd),
	})
	return rows
}

type ParsedCallAttributes struct {
	Hdd      float64
	Preempt  bool
	Ssd      float64
	Memory   float64
	CPU      float64
	Elapsed  time.Duration
	HitCache bool
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
	CachedCalls   int
}

func ResourcesUsed(c *cli.Context) error {
	cromwellClient := FromInterface(c.Context.Value("cromwell"))
	params := url.Values{}
	params.Add("expandSubWorkflows", "true")
	resp, err := cromwellClient.Metadata(c.String("operation"), params)
	if err != nil {
		return err
	}
	if resp.Status != "Succeeded" {
		return errors.New("Workflow status is not Succeeded")
	}
	total, err := GetComputeUsageForPricing(resp.Calls)
	if err != nil {
		return err
	}
	var rtr = ResourceTableResponse{Total: total}
	output.NewTable(os.Stdout).Render(rtr)
	zap.S().Info(fmt.Sprintf("Total of tasks with cache hit: %d", total.CachedCalls))
	return nil
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
			HoursInMonth := 730.0
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
			if parsed.HitCache {
				t.CachedCalls++
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
		Preempt:  isPreempt,
		Hdd:      totalHdd,
		Ssd:      totalSsd,
		HitCache: call.CallCaching.Hit,
		Memory:   memory,
		CPU:      nproc,
		Elapsed:  elapsed}
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
