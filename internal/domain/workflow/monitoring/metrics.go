// Package monitoring provides analysis of resource usage from monitoring logs.
package monitoring

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FileProvider defines the interface for reading file contents.
type FileProvider interface {
	Read(ctx context.Context, path string) (string, error)
}

// MonitoringMetrics holds parsed monitoring data from resource_monitor.sh output.
type MonitoringMetrics struct {
	Timestamps []time.Time
	CPU        []float64 // 0-100%
	MemUsed    []float64 // MB
	MemTotal   float64   // MB
	DiskUsed   []float64 // GB
	DiskTotal  float64   // GB
}

// ParseFromTSV parses the TSV output from resource_monitor.sh.
// Expected format:
// timestamp	cpu_percent	mem_used_mb	mem_total_mb	mem_percent	disk_total_gb	disk_used_gb	...
func ParseFromTSV(content string) (*MonitoringMetrics, error) {
	metrics := &MonitoringMetrics{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()

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
