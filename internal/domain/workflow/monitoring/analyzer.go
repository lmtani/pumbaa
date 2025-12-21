package monitoring

import (
	"fmt"
	"time"
)

// EfficiencyReport summarizes resource usage efficiency.
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

// Analyze calculates efficiency metrics from parsed monitoring data.
func Analyze(metrics *MonitoringMetrics) *EfficiencyReport {
	report := &EfficiencyReport{
		DataPoints: metrics.DataPoints(),
		MemTotal:   metrics.MemTotal,
		DiskTotal:  metrics.DiskTotal,
		Duration:   metrics.Duration(),
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
	if len(metrics.CPU) > 0 {
		report.CPUAvg = cpuSum / float64(len(metrics.CPU))
	}
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
	if len(metrics.MemUsed) > 0 {
		report.MemAvg = memSum / float64(len(metrics.MemUsed))
	}
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
	if len(metrics.DiskUsed) > 0 {
		report.DiskAvg = diskSum / float64(len(metrics.DiskUsed))
	}
	if metrics.DiskTotal > 0 {
		report.DiskEfficiency = diskMax / metrics.DiskTotal
	}

	// Generate recommendations
	report.Recommendations = generateRecommendations(report, metrics)

	return report
}

// generateRecommendations creates actionable suggestions based on efficiency.
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
