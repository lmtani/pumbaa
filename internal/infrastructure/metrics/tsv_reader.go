// Package metrics provides implementations for reading task metrics from various sources.
package metrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// TSVReader implements TaskMetricsReader for TSV files.
type TSVReader struct{}

// NewTSVReader creates a new TSVReader instance.
func NewTSVReader() *TSVReader {
	return &TSVReader{}
}

// ReadFromDirectory reads task metrics from all TSV files in a directory.
// Returns the collection of metrics, a list of workflow IDs found, and any error.
func (r *TSVReader) ReadFromDirectory(dir string) (*workflow.TaskMetricsCollection, []string, error) {
	tsvFiles, err := filepath.Glob(filepath.Join(dir, "*.tsv"))
	if err != nil {
		return nil, nil, err
	}

	if len(tsvFiles) == 0 {
		return nil, nil, nil
	}

	var allMetrics []workflow.TaskMetrics
	var workflows []string

	for _, tsvFile := range tsvFiles {
		workflowID := strings.TrimSuffix(filepath.Base(tsvFile), ".tsv")
		workflows = append(workflows, workflowID)

		metrics, err := r.parseFile(tsvFile, workflowID)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}

	return workflow.NewTaskMetricsCollection(allMetrics), workflows, nil
}

// parseFile parses a single TSV file and returns task metrics.
func (r *TSVReader) parseFile(filename, workflowID string) ([]workflow.TaskMetrics, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metrics []workflow.TaskMetrics
	scanner := bufio.NewScanner(file)
	var headers []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")

		// First line is header
		if headers == nil {
			headers = fields
			continue
		}

		// Parse data row
		metric := workflow.TaskMetrics{WorkflowID: workflowID}
		for i, header := range headers {
			if i >= len(fields) {
				break
			}
			value := fields[i]

			switch header {
			case "task_name":
				metric.TaskName = value
			case "shard_index":
				metric.ShardIndex, _ = strconv.Atoi(value)
			case "cpu_request":
				metric.CPURequest = value
			case "memory_request_bytes":
				metric.MemoryRequestBytes, _ = strconv.ParseInt(value, 10, 64)
			case "disk_size_request_bytes":
				metric.DiskSizeRequestBytes, _ = strconv.ParseInt(value, 10, 64)
			case "disk_type":
				metric.DiskType = value
			case "total_bytes_input":
				metric.TotalInputBytes, _ = strconv.ParseInt(value, 10, 64)
			case "inputs_json":
				_ = json.Unmarshal([]byte(value), &metric.Inputs)
			case "duration_seconds":
				metric.DurationSeconds, _ = strconv.ParseFloat(value, 64)
			case "cpu_mean":
				metric.CPUMean, _ = strconv.ParseFloat(value, 64)
			case "memory_peak_mb":
				metric.MemoryPeakMB, _ = strconv.ParseFloat(value, 64)
			case "disk_peak_gb":
				// Legacy support: convert GB to Bytes (assuming 1GB = 1024^3 bytes)
				val, _ := strconv.ParseFloat(value, 64)
				metric.DiskPeakBytes = int64(val * 1024 * 1024 * 1024)
			case "disk_peak_bytes":
				metric.DiskPeakBytes, _ = strconv.ParseInt(value, 10, 64)
			case "error":
				metric.Error = value
			}
		}

		if metric.TaskName != "" {
			metrics = append(metrics, metric)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}
