package debug

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MonitoringMetrics holds parsed monitoring data from resource_monitor.sh output
type MonitoringMetrics struct {
	Timestamps []time.Time
	CPU        []float64 // 0-100%
	MemUsed    []float64 // MB
	MemTotal   float64   // MB
	DiskUsed   []float64 // GB
	DiskTotal  float64   // GB
}

// EfficiencyReport summarizes resource usage efficiency
type EfficiencyReport struct {
	// CPU metrics
	CPUPeak       float64
	CPUAvg        float64
	CPUEfficiency float64 // avg/100

	// Memory metrics
	MemPeak       float64 // MB
	MemAvg        float64 // MB
	MemTotal      float64 // MB
	MemEfficiency float64 // peak/total

	// Disk metrics
	DiskPeak       float64 // GB
	DiskAvg        float64 // GB
	DiskTotal      float64 // GB
	DiskEfficiency float64 // peak/total

	// Meta
	Duration   time.Duration
	DataPoints int

	// Recommendations
	Recommendations []string
}

// ParseMonitoringLog parses the TSV output from resource_monitor.sh
// Expected format:
// timestamp	cpu_percent	mem_used_mb	mem_total_mb	mem_percent	disk_total_gb	disk_used_gb	...
func ParseMonitoringLog(content string) (*MonitoringMetrics, error) {
	metrics := &MonitoringMetrics{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip header line
		if strings.HasPrefix(line, "timestamp") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue // Skip malformed lines
		}

		// Parse timestamp (format: 2025-12-20 11:11:15)
		ts, err := time.Parse("2006-01-02 15:04:05", fields[0])
		if err != nil {
			continue // Skip lines with invalid timestamps
		}

		// Parse CPU percent
		cpu, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			cpu = 0
		}

		// Parse memory used (MB)
		memUsed, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			memUsed = 0
		}

		// Parse memory total (MB)
		memTotal, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			memTotal = 0
		}

		// Parse disk used (GB) - field 6
		diskUsed, err := strconv.ParseFloat(fields[6], 64)
		if err != nil {
			diskUsed = 0
		}

		// Parse disk total (GB) - field 5
		diskTotal, err := strconv.ParseFloat(fields[5], 64)
		if err != nil {
			diskTotal = 0
		}

		metrics.Timestamps = append(metrics.Timestamps, ts)
		metrics.CPU = append(metrics.CPU, cpu)
		metrics.MemUsed = append(metrics.MemUsed, memUsed)
		metrics.DiskUsed = append(metrics.DiskUsed, diskUsed)

		// Set totals from first valid line
		if metrics.MemTotal == 0 {
			metrics.MemTotal = memTotal
		}
		if metrics.DiskTotal == 0 {
			metrics.DiskTotal = diskTotal
		}
	}

	if len(metrics.Timestamps) == 0 {
		return nil, fmt.Errorf("incompatible format: no valid data points found.\n\nExpected TSV format from resource_monitor.sh:\ntimestamp  cpu_percent  mem_used_mb  mem_total_mb  ...")
	}

	return metrics, nil
}

// AnalyzeEfficiency calculates efficiency metrics from parsed monitoring data
func AnalyzeEfficiency(metrics *MonitoringMetrics) *EfficiencyReport {
	report := &EfficiencyReport{
		DataPoints: len(metrics.Timestamps),
		MemTotal:   metrics.MemTotal,
		DiskTotal:  metrics.DiskTotal,
	}

	if len(metrics.Timestamps) > 1 {
		report.Duration = metrics.Timestamps[len(metrics.Timestamps)-1].Sub(metrics.Timestamps[0])
	}

	// Calculate CPU stats
	var cpuSum, cpuMax float64
	for _, v := range metrics.CPU {
		cpuSum += v
		if v > cpuMax {
			cpuMax = v
		}
	}
	report.CPUPeak = cpuMax
	report.CPUAvg = cpuSum / float64(len(metrics.CPU))
	report.CPUEfficiency = report.CPUAvg / 100

	// Calculate Memory stats
	var memSum, memMax float64
	for _, v := range metrics.MemUsed {
		memSum += v
		if v > memMax {
			memMax = v
		}
	}
	report.MemPeak = memMax
	report.MemAvg = memSum / float64(len(metrics.MemUsed))
	if metrics.MemTotal > 0 {
		report.MemEfficiency = memMax / metrics.MemTotal
	}

	// Calculate Disk stats
	var diskSum, diskMax float64
	for _, v := range metrics.DiskUsed {
		diskSum += v
		if v > diskMax {
			diskMax = v
		}
	}
	report.DiskPeak = diskMax
	report.DiskAvg = diskSum / float64(len(metrics.DiskUsed))
	if metrics.DiskTotal > 0 {
		report.DiskEfficiency = diskMax / metrics.DiskTotal
	}

	// Generate recommendations
	report.Recommendations = generateRecommendations(report, metrics)

	return report
}

// generateRecommendations creates actionable suggestions based on efficiency
func generateRecommendations(report *EfficiencyReport, metrics *MonitoringMetrics) []string {
	var recs []string

	// Memory recommendations
	if report.MemEfficiency < 0.5 && metrics.MemTotal > 2000 {
		suggestedMem := report.MemPeak * 1.3 // 30% headroom
		if suggestedMem < 1000 {
			suggestedMem = 1000 // Minimum 1GB
		}
		recs = append(recs, fmt.Sprintf("Memory: Consider reducing to %.0fMB (%.0f%% unused)",
			suggestedMem, (1-report.MemEfficiency)*100))
	}

	// Disk recommendations
	if report.DiskEfficiency < 0.3 && metrics.DiskTotal > 5 {
		suggestedDisk := report.DiskPeak * 1.5 // 50% headroom
		if suggestedDisk < 2 {
			suggestedDisk = 2 // Minimum 2GB
		}
		recs = append(recs, fmt.Sprintf("Disk: Consider reducing to %.0fGB (%.0f%% unused)",
			suggestedDisk, (1-report.DiskEfficiency)*100))
	}

	// CPU recommendations (if consistently low)
	if report.CPUEfficiency < 0.3 && report.CPUPeak < 50 {
		recs = append(recs, "CPU: Task is CPU-bound with low utilization, consider checking for I/O bottlenecks")
	}

	if len(recs) == 0 {
		recs = append(recs, "✓ Resource allocation looks well-optimized")
	}

	return recs
}
