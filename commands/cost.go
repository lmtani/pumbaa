package commands

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

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

func ResourcesUsed(c *cli.Context) error {
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
	params := url.Values{}
	params.Add("expandSubWorkflows", "true")
	resp, err := cromwellClient.Metadata(c.String("operation"), params)
	if err != nil {
		return err
	}
	if resp.Status == "Running" {
		return errors.New("Workflow status is still running")
	}
	total, err := GetComputeUsageForPricing(resp.Calls)
	if err != nil {
		return err
	}
	var rtr = ResourceTableResponse{Total: total}
	output.NewTable(os.Stdout).Render(rtr)
	color.Cyan(fmt.Sprintf("- Tasks with cache hit: %d", total.CachedCalls))
	color.Cyan(fmt.Sprintf("- Total time with running VMs: %.0fh", total.TotalTime.Hours()))
	return nil
}

func GetComputeUsageForPricing(data map[string][]cromwell.CallItem) (TotalResources, error) {
	t := TotalResources{}
	iterateOverTasks(data, &t)
	return t, nil
}

func iterateOverTasks(data map[string][]cromwell.CallItem, t *TotalResources) {
	for key := range data {
		iterateOverElements(data[key], t)
	}
}

func iterateOverElements(c []cromwell.CallItem, t *TotalResources) {
	for idx := range c {
		if c[idx].SubWorkflowMetadata.RootWorkflowID != "" {
			iterateOverTasks(c[idx].SubWorkflowMetadata.Calls, t)
		} else {
			parsed, _ := iterateOverElement(c[idx])
			HoursInMonth := 720.0
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
			t.TotalTime += parsed.Elapsed
		}
	}
}

func iterateOverElement(call cromwell.CallItem) (ParsedCallAttributes, error) {
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

func parseDisc(c cromwell.CallItem) (float64, string, error) {
	workDisk := strings.Fields(c.RuntimeAttributes.Disks)
	if len(workDisk) == 0 {
		zap.S().Warn(fmt.Sprintf("No disks for: %#v", c.Labels))
		return 0, "", nil
	}
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

func parseMemory(c cromwell.CallItem) (float64, error) {
	memmory := strings.Fields(c.RuntimeAttributes.Memory)
	if len(memmory) == 0 {
		zap.S().Warn(fmt.Sprintf("No memory for: %#v", c.Labels))
		return 0, nil
	}
	size, err := strconv.ParseFloat(memmory[0], 4)
	if err != nil {
		return 0, err
	}
	return size, nil
}
