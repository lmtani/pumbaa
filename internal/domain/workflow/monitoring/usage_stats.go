package monitoring

import (
	"fmt"
	"time"
)

// UsageStats holds aggregated statistics for a specific resource.
type UsageStats struct {
	Peak       float64
	Avg        float64
	Total      float64
	Efficiency float64
}

// NewUsageStats calculates statistics from raw data points.
// useAvgForEfficiency should be true for CPU (Avg/100%) and false for others (Peak/Total).
func NewUsageStats(values []float64, total float64, useAvgForEfficiency bool) UsageStats {
	if len(values) == 0 {
		return UsageStats{Total: total}
	}
	var sum, max float64
	for _, v := range values {
		sum += v
		if v > max {
			max = v
		}
	}
	avg := sum / float64(len(values))
	eff := max / total
	if useAvgForEfficiency {
		eff = avg / total
	}
	return UsageStats{
		Peak:       max,
		Avg:        avg,
		Total:      total,
		Efficiency: eff,
	}
}

// EfficiencyReport summarizes resource usage efficiency.
type EfficiencyReport struct {
	CPU  UsageStats
	Mem  UsageStats
	Disk UsageStats

	// Meta
	Duration   time.Duration
	DataPoints int

	// Recommendations
	Recommendations []string
}

// Analyze calculates efficiency metrics from parsed monitoring data.
func (m *MonitoringMetrics) Analyze() *EfficiencyReport {
	report := &EfficiencyReport{
		CPU:        NewUsageStats(m.CPU, 100, true),
		Mem:        NewUsageStats(m.MemUsed, m.MemTotal, false),
		Disk:       NewUsageStats(m.DiskUsed, m.DiskTotal, false),
		DataPoints: m.DataPoints(),
		Duration:   m.Duration(),
	}

	// Generate recommendations
	report.Recommendations = generateRecommendations(report, m)

	return report
}

// generateRecommendations creates actionable suggestions based on efficiency.
func generateRecommendations(report *EfficiencyReport, metrics *MonitoringMetrics) []string {
	var recs []string

	// Memory recommendations
	if report.Mem.Efficiency < 0.5 && metrics.MemTotal > 2000 {
		suggestedMem := report.Mem.Peak * 1.3 // 30% headroom
		if suggestedMem < 1000 {
			suggestedMem = 1000 // Minimum 1GB
		}
		recs = append(recs, fmt.Sprintf("Memory: Consider reducing to %.0fMB (%.0f%% unused)",
			suggestedMem, (1-report.Mem.Efficiency)*100))
	}

	// Disk recommendations
	if report.Disk.Efficiency < 0.3 && metrics.DiskTotal > 5 {
		suggestedDisk := report.Disk.Peak * 1.5 // 50% headroom
		if suggestedDisk < 2 {
			suggestedDisk = 2 // Minimum 2GB
		}
		recs = append(recs, fmt.Sprintf("Disk: Consider reducing to %.0fGB (%.0f%% unused)",
			suggestedDisk, (1-report.Disk.Efficiency)*100))
	}

	// CPU recommendations (if consistently low)
	if report.CPU.Efficiency < 0.3 && report.CPU.Peak < 50 {
		recs = append(recs, "CPU: Task is CPU-bound with low utilization, consider checking for I/O bottlenecks")
	}

	if len(recs) == 0 {
		recs = append(recs, "✓ Resource allocation looks well-optimized")
	}

	return recs
}
