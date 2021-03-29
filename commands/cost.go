package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GetComputeCost(data map[string][]CallItem) (float64, error) {
	totalHdd := 0.0
	totalSsd := 0.0
	totalProcessors := 0
	totalMemory := 0.0
	var timeElapsed time.Duration
	for key := range data {
		calls := data[key]
		for idx := range calls {
			call := calls[idx]
			size, diskType, err := parseDisc(call)
			if err != nil {
				return 0, err
			}
			if diskType == "SSD" {
				totalSsd += size
			}
			if diskType == "HDD" {
				totalHdd += size
			}
			nproc, _ := strconv.Atoi(call.RuntimeAttributes.CPU)
			totalProcessors += nproc
			memory, err := parseMemory(call)
			if err != nil {
				return 0, err
			}
			totalMemory += memory
			elapsed := call.End.Sub(call.Start)
			timeElapsed += elapsed
		}
	}
	fmt.Printf("SSD: %f\n", totalSsd)
	fmt.Printf("HDD: %f\n", totalHdd)
	fmt.Printf("PROCESSORS: %d\n", totalProcessors)
	fmt.Printf("MEMMORY: %f\n", totalMemory)
	fmt.Println(timeElapsed)
	return 0.0, nil
}

func iterateOverCalls(c []CallItem) {

}

func parseDisc(c CallItem) (float64, string, error) {
	// boot := c.RuntimeAttributes.BootDiskSizeGb
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
