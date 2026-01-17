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
	"sort"
	"strconv"
	"strings"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/templates"
)

// ResourceVisualizationUseCase handles resource visualization report generation.
type ResourceVisualizationUseCase struct {
	recommendationGenerator ports.RecommendationGenerator
}

// NewResourceVisualizationUseCase creates a new resource visualization use case.
// generator can be nil if LLM is not configured - recommendations will be skipped.
func NewResourceVisualizationUseCase(generator ports.RecommendationGenerator) *ResourceVisualizationUseCase {
	return &ResourceVisualizationUseCase{
		recommendationGenerator: generator,
	}
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
	TaskName             string           `json:"taskName"`
	ShardIndex           int              `json:"shardIndex"`
	CPURequest           string           `json:"cpuRequest"`
	MemoryRequestBytes   int64            `json:"memoryRequestBytes"`
	DiskSizeRequestBytes int64            `json:"diskSizeRequestBytes"`
	DiskType             string           `json:"diskType"`
	TotalInputBytes      int64            `json:"totalInputBytes"`
	Inputs               map[string]int64 `json:"inputs"`
	DurationSeconds      float64          `json:"durationSeconds"`
	CPUMean              float64          `json:"cpuMean"`
	MemoryPeakMB         float64          `json:"memoryPeakMB"`
	DiskPeakGB           float64          `json:"diskPeakGB"`
	Error                string           `json:"error"`
	WorkflowID           string           `json:"workflowId"`
}

// TaskRecommendation contains optimization recommendations for a task.
type TaskRecommendation struct {
	TaskName        string   `json:"taskName"`
	SampleCount     int      `json:"sampleCount"`
	ResourceCost    float64  `json:"resourceCost"` // Total dimensionless cost for prioritization
	CPUCost         float64  `json:"cpuCost"`      // CPU contribution (CPU × Hours)
	MemoryCost      float64  `json:"memoryCost"`   // Memory contribution (GB × Hours)
	DiskCost        float64  `json:"diskCost"`     // Disk contribution (GB × Hours)
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
	// Group valid data by task name, filtering out noisy tasks
	taskGroups := make(map[string][]TaskData)
	for _, task := range allData {
		if task.Error != "" {
			continue // Skip tasks with errors
		}
		// Filter noisy tasks: CPU=0% AND duration < 60s indicates unreliable metrics
		if task.CPUMean == 0 && task.DurationSeconds < 60 {
			continue
		}
		taskGroups[task.TaskName] = append(taskGroups[task.TaskName], task)
	}

	var recommendations []TaskRecommendation
	const minSamples = 3
	const minR2 = 0.7

	for taskName, tasks := range taskGroups {
		// Calculate stratified resource costs for this task group
		var totalCPUCost, totalMemoryCost, totalDiskCost float64
		for _, t := range tasks {
			cpuVal := 1.0
			if t.CPURequest != "" {
				if parsed, err := strconv.ParseFloat(t.CPURequest, 64); err == nil && parsed > 0 {
					cpuVal = parsed
				}
			}
			memGB := float64(t.MemoryRequestBytes) / (1024 * 1024 * 1024)
			diskGB := float64(t.DiskSizeRequestBytes) / (1024 * 1024 * 1024)
			durationHours := t.DurationSeconds / 3600
			if durationHours > 0 {
				totalCPUCost += cpuVal * durationHours
				totalMemoryCost += memGB * durationHours
				totalDiskCost += diskGB * durationHours
			}
		}

		// Total resource cost combines all components
		totalResourceCost := totalCPUCost * totalMemoryCost * totalDiskCost / float64(len(tasks)*len(tasks))

		rec := TaskRecommendation{
			TaskName:     taskName,
			SampleCount:  len(tasks),
			ResourceCost: math.Round(totalResourceCost*100) / 100,
			CPUCost:      math.Round(totalCPUCost*100) / 100,
			MemoryCost:   math.Round(totalMemoryCost*100) / 100,
			DiskCost:     math.Round(totalDiskCost*100) / 100,
		}

		// Need at least minSamples for meaningful regression
		if len(tasks) >= minSamples {
			// 1. Collect Data Arrays
			var diskPeaksGB, memPeaksMB []float64
			var totalCPUMean float64
			var totalMemReq, totalDiskReq int64

			// Collect all unique input keys and their values per task
			inputKeys := make(map[string]bool)
			// Map to check for constant input sizes: key -> size (if constant), -1 if variable
			inputConstSizes := make(map[string]int64)

			// Initialize inputConstSizes with the first task's inputs
			for k, v := range tasks[0].Inputs {
				inputConstSizes[k] = v
			}

			for _, t := range tasks {
				diskPeaksGB = append(diskPeaksGB, t.DiskPeakGB)
				memPeaksMB = append(memPeaksMB, t.MemoryPeakMB)
				totalCPUMean += t.CPUMean
				totalMemReq += t.MemoryRequestBytes
				totalDiskReq += t.DiskSizeRequestBytes

				for k := range t.Inputs {
					inputKeys[k] = true
				}

				// Check for constants
				for k, v := range inputConstSizes {
					if v != -1 {
						if currentVal, ok := t.Inputs[k]; !ok || currentVal != v {
							inputConstSizes[k] = -1 // Not constant or missing
						}
					}
				}
				// Keys present in current task but not in initialization (unlikely if same WDL, but possible)
				// are ignored for "constant" candidate status as they weren't in the first one.
				// To be strictly correct we should ensure it exists in ALL. The logic above handles mis-match by checking existence.
			}

			// Identify large constant inputs (> 100MB) to use in formula explanations
			var constantInputs []string
			for k, v := range inputConstSizes {
				if v > 100*1024*1024 { // 100MB threshold
					constantInputs = append(constantInputs, k)
				}
			}

			// 2. Generic Consumption Stats
			avgCPUMean := totalCPUMean / float64(len(tasks))
			avgMemReqMB := float64(totalMemReq) / float64(len(tasks)) / (1024 * 1024)
			avgDiskReqGB := float64(totalDiskReq) / float64(len(tasks)) / (1024 * 1024 * 1024)

			var avgMemPeak, avgDiskPeak float64
			for _, t := range tasks {
				avgMemPeak += t.MemoryPeakMB
				avgDiskPeak += t.DiskPeakGB
			}
			avgMemPeak /= float64(len(tasks))
			avgDiskPeak /= float64(len(tasks))

			// 3. Helper to find best regression
			// We prefer larger inputs when R² values are similar - this gives more intuitive slopes
			type bestFit struct {
				Key       string // "total" or input name
				R2        float64
				Slope     float64
				Intercept float64
				AvgSizeGB float64 // Average size of this input (to prefer larger inputs)
			}

			findBestFit := func(ys []float64) bestFit {
				// Start with TotalInputBytes as baseline
				var baselineXs []float64
				var totalAvgSize float64
				for _, t := range tasks {
					sizeGB := float64(t.TotalInputBytes) / (1024 * 1024 * 1024)
					baselineXs = append(baselineXs, sizeGB)
					totalAvgSize += sizeGB
				}
				totalAvgSize /= float64(len(tasks))
				reg := linearRegression(baselineXs, ys)
				best := bestFit{Key: "total", R2: reg.R2, Slope: reg.Slope, Intercept: reg.Intercept, AvgSizeGB: totalAvgSize}

				// Collect all candidates with R² >= minR2
				type candidate struct {
					key       string
					r2        float64
					slope     float64
					intercept float64
					avgSizeGB float64
				}
				var candidates []candidate

				// Check each individual input
				for key := range inputKeys {
					// Skip if this key is effectively constant (variance ~ 0)
					if v, ok := inputConstSizes[key]; ok && v != -1 {
						continue
					}

					var connectedXs []float64
					var avgSize float64
					for _, t := range tasks {
						val := t.Inputs[key]
						sizeGB := float64(val) / (1024 * 1024 * 1024)
						connectedXs = append(connectedXs, sizeGB)
						avgSize += sizeGB
					}
					avgSize /= float64(len(tasks))

					r := linearRegression(connectedXs, ys)
					if r.R2 >= minR2 {
						candidates = append(candidates, candidate{
							key:       key,
							r2:        r.R2,
							slope:     r.Slope,
							intercept: r.Intercept,
							avgSizeGB: avgSize,
						})
					}
				}

				// Among candidates with similar R² (within 0.05), prefer the one with largest average size
				// This gives more intuitive slopes (smaller multipliers)
				for _, c := range candidates {
					// If R² is similar or better AND this input is larger, prefer it
					if c.r2 >= best.R2-0.05 && c.avgSizeGB > best.AvgSizeGB {
						best = bestFit{Key: c.key, R2: c.r2, Slope: c.slope, Intercept: c.intercept, AvgSizeGB: c.avgSizeGB}
					} else if c.r2 > best.R2 {
						// Always prefer if R² is strictly better
						best = bestFit{Key: c.key, R2: c.r2, Slope: c.slope, Intercept: c.intercept, AvgSizeGB: c.avgSizeGB}
					}
				}
				return best
			}

			// 4. Generate Formula String Helper
			generateFormula := func(fit bestFit, target string) string {
				slope := math.Ceil(fit.Slope*10) / 10
				intercept := fit.Intercept

				// Clamp negative intercepts to 0 (they make no practical sense)
				if intercept < 0 {
					intercept = 0
				}

				// Try to explain intercept with constants
				var explainedParts []string

				// Only try to decompose if we have a positive intercept
				if intercept > 0 {
					for _, k := range constantInputs {
						// Don't use the variable itself as a constant (boundary case)
						if k == fit.Key {
							continue
						}

						sizeGB := float64(inputConstSizes[k]) / (1024 * 1024 * 1024)
						// If constant size fits within the intercept (with some buffer 0.1GB), subtract it
						if intercept >= sizeGB-0.1 {
							intercept -= sizeGB
							explainedParts = append(explainedParts, fmt.Sprintf(`size(%s, "GB")`, k))
						}
					}
				}

				intercept = math.Ceil(intercept)
				// Clamp again after ceiling in case we went negative from subtraction
				if intercept < 0 {
					intercept = 0
				}

				var variablePart string
				if fit.Key == "total" {
					variablePart = fmt.Sprintf("%.1f * size(inputs, \"GB\")", slope)
				} else {
					variablePart = fmt.Sprintf("%.1f * size(%s, \"GB\")", slope, fit.Key)
				}

				// Construct final formula
				var formulaBuilder strings.Builder
				formulaBuilder.WriteString(fmt.Sprintf("%s = ceil(", target))
				formulaBuilder.WriteString(variablePart)

				if intercept > 0 {
					formulaBuilder.WriteString(fmt.Sprintf(" + %.0f", intercept))
				}
				formulaBuilder.WriteString(")")

				for _, part := range explainedParts {
					formulaBuilder.WriteString(" + ")
					formulaBuilder.WriteString(part)
				}

				return formulaBuilder.String()
			}

			// 5. Run Analysis for Disk
			diskFit := findBestFit(diskPeaksGB)
			if diskFit.R2 >= minR2 {
				rec.DiskR2 = math.Round(diskFit.R2*100) / 100
				rec.DiskFormula = generateFormula(diskFit, "Int disk_gb")
			}

			// 6. Run Analysis for Memory
			memFit := findBestFit(memPeaksMB)
			if memFit.R2 >= minR2 {
				rec.MemoryR2 = math.Round(memFit.R2*100) / 100
				// Adjust slope/intercept to GB for memory formula if needed,
				// but usually memory is just MB. User asked for "Int memory_gb", so assuming GB.
				// The regression was run on MB (Y). We need to convert results to GB.

				memFitGB := bestFit{
					Key:       memFit.Key,
					R2:        memFit.R2,
					Slope:     memFit.Slope / 1024,
					Intercept: memFit.Intercept / 1024,
				}
				rec.MemoryFormula = generateFormula(memFitGB, "Int memory_gb")
			}

			// 7. Generic recommendations
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

	// Sort recommendations by resource cost (highest first) for prioritization
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].ResourceCost > recommendations[j].ResourceCost
	})

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

	// Filter out data with errors for visualization
	var validData []TaskData
	for _, d := range allData {
		if d.Error == "" {
			validData = append(validData, d)
		}
	}

	// Generate JSON for template
	dataJSON, err := json.Marshal(validData)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode data as JSON", err)
	}

	workflowsJSON, err := json.Marshal(workflows)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode workflows as JSON", err)
	}

	// Generate recommendations using LLM if available, otherwise skip
	var recommendations []ports.TaskRecommendation
	if uc.recommendationGenerator != nil && uc.recommendationGenerator.IsAvailable() {
		// Convert TaskData to TaskAnalysisData for the generator
		analysisData := uc.convertToAnalysisData(validData)
		if len(analysisData) > 0 {
			var err error
			recommendations, err = uc.recommendationGenerator.GenerateRecommendations(ctx, analysisData)
			if err != nil {
				// Log error but don't fail - just skip recommendations
				recommendations = []ports.TaskRecommendation{}
			}
		}
	}
	// Note: If generator not available, recommendations will be empty

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

// convertToAnalysisData groups TaskData by task name and converts to TaskAnalysisData for the LLM.
func (uc *ResourceVisualizationUseCase) convertToAnalysisData(validData []TaskData) []ports.TaskAnalysisData {
	// Group by task name
	taskGroups := make(map[string][]TaskData)
	for _, task := range validData {
		if task.Error == "" {
			taskGroups[task.TaskName] = append(taskGroups[task.TaskName], task)
		}
	}

	var result []ports.TaskAnalysisData
	for taskName, tasks := range taskGroups {
		if len(tasks) < 3 {
			continue // Need at least 3 samples for meaningful analysis
		}

		analysisData := ports.TaskAnalysisData{
			TaskName:    taskName,
			SampleCount: len(tasks),
			InputSizes:  make(map[string][]int64),
		}

		// Collect metrics per sample
		for _, t := range tasks {
			analysisData.DiskPeaksGB = append(analysisData.DiskPeaksGB, t.DiskPeakGB)
			analysisData.MemoryPeaksMB = append(analysisData.MemoryPeaksMB, t.MemoryPeakMB)
			analysisData.CPUMeans = append(analysisData.CPUMeans, t.CPUMean)
			analysisData.DurationSeconds = append(analysisData.DurationSeconds, t.DurationSeconds)

			// Collect input sizes
			for name, size := range t.Inputs {
				analysisData.InputSizes[name] = append(analysisData.InputSizes[name], size)
			}
		}

		// Use first sample for resource requests (should be consistent across shards)
		first := tasks[0]
		analysisData.CPURequest = first.CPURequest
		analysisData.MemoryReqGB = float64(first.MemoryRequestBytes) / (1024 * 1024 * 1024)
		analysisData.DiskReqGB = float64(first.DiskSizeRequestBytes) / (1024 * 1024 * 1024)

		// Calculate resource cost
		var totalCost float64
		for _, t := range tasks {
			cpuVal := 1.0
			if parsed, err := strconv.ParseFloat(t.CPURequest, 64); err == nil && parsed > 0 {
				cpuVal = parsed
			}
			memGB := float64(t.MemoryRequestBytes) / (1024 * 1024 * 1024)
			diskGB := float64(t.DiskSizeRequestBytes) / (1024 * 1024 * 1024)
			durationHours := t.DurationSeconds / 3600
			if durationHours > 0 {
				totalCost += cpuVal * memGB * diskGB * durationHours
			}
		}
		analysisData.ResourceCost = totalCost

		result = append(result, analysisData)
	}

	// Sort by resource cost (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ResourceCost > result[j].ResourceCost
	})

	return result
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
			case "inputs_json":
				_ = json.Unmarshal([]byte(value), &task.Inputs)
			case "duration_seconds":
				task.DurationSeconds, _ = strconv.ParseFloat(value, 64)
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
