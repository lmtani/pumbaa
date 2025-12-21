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

const (
	ColTimestamp   = "timestamp"
	ColCPUPercent  = "cpu_percent"
	ColMemUsedMB   = "mem_used_mb"
	ColMemTotalMB  = "mem_total_mb"
	ColDiskUsedGB  = "disk_used_gb"
	ColDiskTotalGB = "disk_total_gb"
)

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
// It dynamically maps columns based on the header line.
func ParseFromTSV(content string) (*MonitoringMetrics, error) {
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
