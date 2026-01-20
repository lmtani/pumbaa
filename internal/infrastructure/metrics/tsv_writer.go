// Package metrics provides implementations for reading and writing task metrics.
package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// TSVWriter implements TaskMetricsWriter for TSV files.
type TSVWriter struct{}

// NewTSVWriter creates a new TSVWriter instance.
func NewTSVWriter() *TSVWriter {
	return &TSVWriter{}
}

// WriteToFile writes task metrics to a TSV file.
func (w *TSVWriter) WriteToFile(filename string, metrics []workflow.TaskMetrics) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, "task_name\tshard_index\tcpu_request\tmemory_request_bytes\tdisk_size_request_bytes\tdisk_type\ttotal_bytes_input\tinputs_json\tduration_seconds\tcpu_mean\tmemory_peak_mb\tdisk_peak_bytes\terror")
	if err != nil {
		return err
	}

	for _, task := range metrics {
		inputsJSON, _ := json.Marshal(task.Inputs)
		if inputsJSON == nil {
			inputsJSON = []byte("{}")
		}

		// Sanitize error message to prevent newlines breaking TSV format.
		errorMsg := strings.ReplaceAll(task.Error, "\n", " ")
		errorMsg = strings.ReplaceAll(errorMsg, "\r", "")
		errorMsg = strings.ReplaceAll(errorMsg, "\t", " ")

		_, err = fmt.Fprintf(file, "%s\t%d\t%s\t%d\t%d\t%s\t%d\t%s\t%.2f\t%.2f\t%.2f\t%d\t%s\n",
			task.TaskName,
			task.ShardIndex,
			task.CPURequest,
			task.MemoryRequestBytes,
			task.DiskSizeRequestBytes,
			task.DiskType,
			task.TotalInputBytes,
			string(inputsJSON),
			task.DurationSeconds,
			task.CPUMean,
			task.MemoryPeakMB,
			task.DiskPeakBytes,
			errorMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

// Ensure TSVWriter implements the domain interface at compile time.
var _ ports.TaskMetricsWriter = (*TSVWriter)(nil)
