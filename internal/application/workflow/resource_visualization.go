// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/infrastructure/templates"
)

// ResourceVisualizationUseCase handles resource visualization report generation.
type ResourceVisualizationUseCase struct{}

// NewResourceVisualizationUseCase creates a new resource visualization use case.
func NewResourceVisualizationUseCase() *ResourceVisualizationUseCase {
	return &ResourceVisualizationUseCase{}
}

// ResourceVisualizationInput represents the input for resource visualization.
type ResourceVisualizationInput struct {
	Directory  string // Directory containing TSV files
	OutputFile string // Output HTML file (default: "resource_report.html")
}

// ResourceVisualizationOutput contains the result of visualization generation.
type ResourceVisualizationOutput struct {
	OutputFile    string
	WorkflowCount int
	TaskCount     int
}

// TaskData represents a single task row from TSV for JSON serialization.
type TaskData struct {
	TaskName             string  `json:"taskName"`
	ShardIndex           int     `json:"shardIndex"`
	CPURequest           string  `json:"cpuRequest"`
	MemoryRequestBytes   int64   `json:"memoryRequestBytes"`
	DiskSizeRequestBytes int64   `json:"diskSizeRequestBytes"`
	DiskType             string  `json:"diskType"`
	TotalInputBytes      int64   `json:"totalInputBytes"`
	CPUMean              float64 `json:"cpuMean"`
	MemoryPeakMB         float64 `json:"memoryPeakMB"`
	DiskPeakGB           float64 `json:"diskPeakGB"`
	Error                string  `json:"error"`
	WorkflowID           string  `json:"workflowId"`
}

// TaskRecommendation contains optimization recommendations for a task.
type TaskRecommendation struct {
	TaskName        string   `json:"taskName"`
	SampleCount     int      `json:"sampleCount"`
	DiskFormula     string   `json:"diskFormula,omitempty"`
	DiskR2          float64  `json:"diskR2,omitempty"`
	MemoryFormula   string   `json:"memoryFormula,omitempty"`
	MemoryR2        float64  `json:"memoryR2,omitempty"`
	Recommendations []string `json:"recommendations"`
}

// regressionResult holds the result of a linear regression.
type regressionResult struct {
	Slope     float64
	Intercept float64
	R2        float64
}

// linearRegression calculates simple linear regression for (x, y) points.
// Returns slope, intercept, and R² (coefficient of determination).
func linearRegression(xs, ys []float64) regressionResult {
	n := float64(len(xs))
	if n < 2 {
		return regressionResult{}
	}

	// Calculate means
	var sumX, sumY float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate slope and intercept
	var numerator, denominator float64
	for i := range xs {
		numerator += (xs[i] - meanX) * (ys[i] - meanY)
		denominator += (xs[i] - meanX) * (xs[i] - meanX)
	}

	if denominator == 0 {
		return regressionResult{Intercept: meanY}
	}

	slope := numerator / denominator
	intercept := meanY - slope*meanX

	// Calculate R²
	var ssRes, ssTot float64
	for i := range xs {
		predicted := slope*xs[i] + intercept
		ssRes += (ys[i] - predicted) * (ys[i] - predicted)
		ssTot += (ys[i] - meanY) * (ys[i] - meanY)
	}

	var r2 float64
	if ssTot > 0 {
		r2 = 1 - ssRes/ssTot
	}

	return regressionResult{Slope: slope, Intercept: intercept, R2: r2}
}

// generateRecommendations analyzes task data and generates optimization recommendations.
func generateRecommendations(allData []TaskData) []TaskRecommendation {
	// Group valid data by task name
	taskGroups := make(map[string][]TaskData)
	for _, task := range allData {
		if task.Error != "" {
			continue // Skip tasks with errors
		}
		taskGroups[task.TaskName] = append(taskGroups[task.TaskName], task)
	}

	var recommendations []TaskRecommendation
	const minSamples = 3
	const minR2 = 0.7

	for taskName, tasks := range taskGroups {
		rec := TaskRecommendation{
			TaskName:    taskName,
			SampleCount: len(tasks),
		}

		// Need at least minSamples for meaningful regression
		if len(tasks) >= minSamples {
			// Prepare data points
			var inputSizesGB, diskPeaksGB, memPeaksMB []float64
			var totalCPUMean float64
			var totalMemReq, totalDiskReq int64

			for _, t := range tasks {
				inputGB := float64(t.TotalInputBytes) / (1024 * 1024 * 1024)
				inputSizesGB = append(inputSizesGB, inputGB)
				diskPeaksGB = append(diskPeaksGB, t.DiskPeakGB)
				memPeaksMB = append(memPeaksMB, t.MemoryPeakMB)
				totalCPUMean += t.CPUMean
				totalMemReq += t.MemoryRequestBytes
				totalDiskReq += t.DiskSizeRequestBytes
			}

			avgCPUMean := totalCPUMean / float64(len(tasks))
			avgMemReqMB := float64(totalMemReq) / float64(len(tasks)) / (1024 * 1024)
			avgDiskReqGB := float64(totalDiskReq) / float64(len(tasks)) / (1024 * 1024 * 1024)

			// Calculate average peaks for efficiency
			var avgMemPeak, avgDiskPeak float64
			for _, t := range tasks {
				avgMemPeak += t.MemoryPeakMB
				avgDiskPeak += t.DiskPeakGB
			}
			avgMemPeak /= float64(len(tasks))
			avgDiskPeak /= float64(len(tasks))

			// Disk regression
			diskReg := linearRegression(inputSizesGB, diskPeaksGB)
			if diskReg.R2 >= minR2 {
				rec.DiskR2 = math.Round(diskReg.R2*100) / 100
				slope := math.Ceil(diskReg.Slope*10) / 10
				intercept := math.Ceil(diskReg.Intercept)
				if intercept > 0 {
					rec.DiskFormula = fmt.Sprintf("Int disk_gb = ceil(%.1f * size(inputs, \"GB\") + %.0f)", slope, intercept)
				} else {
					rec.DiskFormula = fmt.Sprintf("Int disk_gb = ceil(%.1f * size(inputs, \"GB\"))", slope)
				}
			}

			// Memory regression (convert MB to GB for formula)
			memReg := linearRegression(inputSizesGB, memPeaksMB)
			if memReg.R2 >= minR2 {
				rec.MemoryR2 = math.Round(memReg.R2*100) / 100
				slopeGB := math.Ceil(memReg.Slope/1024*10) / 10
				interceptGB := math.Ceil(memReg.Intercept / 1024)
				if interceptGB > 0 {
					rec.MemoryFormula = fmt.Sprintf("Int memory_gb = ceil(%.1f * size(inputs, \"GB\") + %.0f)", slopeGB, interceptGB)
				} else {
					rec.MemoryFormula = fmt.Sprintf("Int memory_gb = ceil(%.1f * size(inputs, \"GB\"))", slopeGB)
				}
			}

			// Generic recommendations
			if avgCPUMean < 30 {
				rec.Recommendations = append(rec.Recommendations,
					fmt.Sprintf("CPU utilization is low (%.0f%%). Consider reducing CPU request.", avgCPUMean))
			}

			memEfficiency := (avgMemPeak / avgMemReqMB) * 100
			if memEfficiency < 50 {
				rec.Recommendations = append(rec.Recommendations,
					fmt.Sprintf("Memory utilization is low (%.0f%%). Consider reducing memory request.", memEfficiency))
			}

			diskEfficiency := (avgDiskPeak / avgDiskReqGB) * 100
			if diskEfficiency < 30 {
				rec.Recommendations = append(rec.Recommendations,
					fmt.Sprintf("Disk utilization is low (%.0f%%). Consider reducing disk request.", diskEfficiency))
			}
		}

		// Only add if there are any recommendations or formulas
		if len(rec.Recommendations) > 0 || rec.DiskFormula != "" || rec.MemoryFormula != "" {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// Execute generates the HTML visualization report.
func (uc *ResourceVisualizationUseCase) Execute(ctx context.Context, input ResourceVisualizationInput) (*ResourceVisualizationOutput, error) {
	if input.Directory == "" {
		return nil, application.NewInputValidationError("directory", "is required")
	}

	// Set default output file
	outputFile := input.OutputFile
	if outputFile == "" {
		outputFile = "resource_report.html"
	}

	// Find all TSV files in the directory
	tsvFiles, err := filepath.Glob(filepath.Join(input.Directory, "*.tsv"))
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to find TSV files", err)
	}

	if len(tsvFiles) == 0 {
		return nil, application.NewUseCaseError("resource_visualization", "no TSV files found in directory", nil)
	}

	// Parse all TSV files and collect data
	var allData []TaskData
	var workflows []string

	for _, tsvFile := range tsvFiles {
		workflowID := strings.TrimSuffix(filepath.Base(tsvFile), ".tsv")
		workflows = append(workflows, workflowID)

		tasks, err := uc.parseTSV(tsvFile, workflowID)
		if err != nil {
			// Skip files that can't be parsed, but log the error
			continue
		}
		allData = append(allData, tasks...)
	}

	if len(allData) == 0 {
		return nil, application.NewUseCaseError("resource_visualization", "no valid data found in TSV files", nil)
	}

	// Generate JSON for template
	dataJSON, err := json.Marshal(allData)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode data as JSON", err)
	}

	workflowsJSON, err := json.Marshal(workflows)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode workflows as JSON", err)
	}

	// Generate recommendations
	recommendations := generateRecommendations(allData)
	recommendationsJSON, err := json.Marshal(recommendations)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode recommendations as JSON", err)
	}

	// Render HTML template
	html, err := templates.RenderReport(templates.ReportData{
		DataJSON:            template.JS(dataJSON),
		WorkflowsJSON:       template.JS(workflowsJSON),
		RecommendationsJSON: template.JS(recommendationsJSON),
	})
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to render HTML template", err)
	}

	// Write HTML to file
	if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to write HTML file", err)
	}

	// Count unique tasks
	taskNames := make(map[string]bool)
	for _, t := range allData {
		taskNames[t.TaskName] = true
	}

	return &ResourceVisualizationOutput{
		OutputFile:    outputFile,
		WorkflowCount: len(workflows),
		TaskCount:     len(taskNames),
	}, nil
}

// parseTSV parses a TSV file and returns task data.
func (uc *ResourceVisualizationUseCase) parseTSV(filename string, workflowID string) ([]TaskData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []TaskData
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
		task := TaskData{WorkflowID: workflowID}
		for i, header := range headers {
			if i >= len(fields) {
				break
			}
			value := fields[i]

			switch header {
			case "task_name":
				task.TaskName = value
			case "shard_index":
				task.ShardIndex, _ = strconv.Atoi(value)
			case "cpu_request":
				task.CPURequest = value
			case "memory_request_bytes":
				task.MemoryRequestBytes, _ = strconv.ParseInt(value, 10, 64)
			case "disk_size_request_bytes":
				task.DiskSizeRequestBytes, _ = strconv.ParseInt(value, 10, 64)
			case "disk_type":
				task.DiskType = value
			case "total_bytes_input":
				task.TotalInputBytes, _ = strconv.ParseInt(value, 10, 64)
			case "cpu_mean":
				task.CPUMean, _ = strconv.ParseFloat(value, 64)
			case "memory_peak_mb":
				task.MemoryPeakMB, _ = strconv.ParseFloat(value, 64)
			case "disk_peak_gb":
				task.DiskPeakGB, _ = strconv.ParseFloat(value, 64)
			case "error":
				task.Error = value
			}
		}

		if task.TaskName != "" {
			tasks = append(tasks, task)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}
