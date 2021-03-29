package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GetComputeCost(data map[string][]CallItem) (float64, error) {
	for key := range data {
		iterateOverTasks(data[key])
	}

	return 0.0, nil
}

func iterateOverTasks(c []CallItem) {
	totalHdd := 0.0
	totalSsd := 0.0
	totalProcessors := 0.0
	totalMemory := 0.0
	var timeElapsed time.Duration
	for idx := range c {
		preempt, Hdd, Ssd, memory, nproc, _ := iterateOverCalls(c[idx])
		totalHdd += Hdd
		totalSsd += Ssd
		totalMemory += memory
		totalProcessors += nproc
		fmt.Printf("Is preempt: %t\n", preempt)
	}
	fmt.Printf("SSD: %f\n", totalSsd)
	fmt.Printf("HDD: %f\n", totalHdd)
	fmt.Printf("PROCESSORS: %f\n", totalProcessors)
	fmt.Printf("MEMMORY: %f\n", totalMemory)
	fmt.Println(timeElapsed)
}

func iterateOverCalls(call CallItem) (bool, float64, float64, float64, float64, error) {
	size, diskType, err := parseDisc(call)
	if err != nil {
		return false, 0, 0, 0, 0, err
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
		return false, 0, 0, 0, 0, err
	}
	elapsed := call.End.Sub(call.Start)
	normalizedMem := normalizeUsePerHour(memory, elapsed)
	normalizedCPU := normalizeUsePerHour(nproc, elapsed)
	isPreempt := call.RuntimeAttributes.Preemptible != "0"
	return isPreempt, totalHdd, totalSsd, normalizedMem, normalizedCPU, nil
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
