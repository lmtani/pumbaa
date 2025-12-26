// monitoring.go contains resource monitoring types and analysis for workflows.
// This includes parsing monitoring logs and calculating efficiency metrics.
package workflow

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Monitoring log column names
const (
	ColTimestamp   = "timestamp"
	ColCPUPercent  = "cpu_percent"
	ColMemUsedMB   = "mem_used_mb"
	ColMemTotalMB  = "mem_total_mb"
	ColDiskUsedGB  = "disk_used_gb"
	ColDiskTotalGB = "disk_total_gb"
)

// MonitoringMetrics is a Value Object holding parsed monitoring data from resource_monitor.sh output.
// It provides the Analyze() method to compute an EfficiencyReport.
type MonitoringMetrics struct {
	Timestamps []time.Time
	CPU        []float64 // 0-100%
	MemUsed    []float64 // MB
	MemTotal   float64   // MB
	DiskUsed   []float64 // GB
	DiskTotal  float64   // GB
}

// ResourceUsageStats is a Value Object holding aggregated statistics for a specific resource.
type ResourceUsageStats struct {
	Peak       float64
	Avg        float64
	Total      float64
	Efficiency float64
}

// EfficiencyReport is a Value Object summarizing resource usage efficiency.
// It is computed by MonitoringMetrics.Analyze() and represents a snapshot analysis.
type EfficiencyReport struct {
	CPU  ResourceUsageStats
	Mem  ResourceUsageStats
	Disk ResourceUsageStats

	// Meta
	Duration   time.Duration
	DataPoints int

	// Recommendations
	Recommendations []string
}

// ParseMonitoringTSV parses the TSV output from resource_monitor.sh.
// It dynamically maps columns based on the header line.
func ParseMonitoringTSV(content string) (*MonitoringMetrics, error) {
	metrics := &MonitoringMetrics{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	var headerMap map[string]int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")

		// First line is the header
		if headerMap == nil {
			headerMap = make(map[string]int)
			for i, field := range fields {
				headerMap[strings.TrimSpace(field)] = i
			}
			// Validate essential headers
			required := []string{ColTimestamp, ColCPUPercent, ColMemUsedMB, ColMemTotalMB, ColDiskUsedGB, ColDiskTotalGB}
			for _, req := range required {
				if _, ok := headerMap[req]; !ok {
					return nil, fmt.Errorf("missing required column in monitoring log: %s", req)
				}
			}
			continue
		}

		// Helper to extract float field
		getFloat := func(name string) float64 {
			idx, ok := headerMap[name]
			if !ok || idx >= len(fields) {
				return 0
			}
			val, _ := strconv.ParseFloat(fields[idx], 64)
			return val
		}

		// Parse timestamp
		tsIdx, _ := headerMap[ColTimestamp]
		if tsIdx >= len(fields) {
			continue
		}
		ts, err := time.Parse("2006-01-02 15:04:05", fields[tsIdx])
		if err != nil {
			continue
		}

		metrics.Timestamps = append(metrics.Timestamps, ts)
		metrics.CPU = append(metrics.CPU, getFloat(ColCPUPercent))
		metrics.MemUsed = append(metrics.MemUsed, getFloat(ColMemUsedMB))
		metrics.DiskUsed = append(metrics.DiskUsed, getFloat(ColDiskUsedGB))

		// Set totals from first valid line
		if metrics.MemTotal == 0 {
			metrics.MemTotal = getFloat(ColMemTotalMB)
		}
		if metrics.DiskTotal == 0 {
			metrics.DiskTotal = getFloat(ColDiskTotalGB)
		}
	}

	if len(metrics.Timestamps) == 0 {
		return nil, fmt.Errorf("incompatible format: no valid data points found.\n\nExpected TSV format with headers: timestamp, cpu_percent, mem_used_mb, ...")
	}

	return metrics, nil
}

// DataPoints returns the number of data points in the metrics.
func (m *MonitoringMetrics) DataPoints() int {
	return len(m.Timestamps)
}

// Duration returns the total duration covered by the metrics.
func (m *MonitoringMetrics) Duration() time.Duration {
	if len(m.Timestamps) < 2 {
		return 0
	}
	return m.Timestamps[len(m.Timestamps)-1].Sub(m.Timestamps[0])
}

// Analyze calculates efficiency metrics from parsed monitoring data.
func (m *MonitoringMetrics) Analyze() *EfficiencyReport {
	report := &EfficiencyReport{
		CPU:        NewResourceUsageStats(m.CPU, 100, true),
		Mem:        NewResourceUsageStats(m.MemUsed, m.MemTotal, false),
		Disk:       NewResourceUsageStats(m.DiskUsed, m.DiskTotal, false),
		DataPoints: m.DataPoints(),
		Duration:   m.Duration(),
	}

	// Generate recommendations
	report.Recommendations = generateMonitoringRecommendations(report, m)

	return report
}

// NewResourceUsageStats calculates statistics from raw data points.
// useAvgForEfficiency should be true for CPU (Avg/100%) and false for others (Peak/Total).
func NewResourceUsageStats(values []float64, total float64, useAvgForEfficiency bool) ResourceUsageStats {
	if len(values) == 0 {
		return ResourceUsageStats{Total: total}
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
	return ResourceUsageStats{
		Peak:       max,
		Avg:        avg,
		Total:      total,
		Efficiency: eff,
	}
}

// generateMonitoringRecommendations creates actionable suggestions based on efficiency.
func generateMonitoringRecommendations(report *EfficiencyReport, metrics *MonitoringMetrics) []string {
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
