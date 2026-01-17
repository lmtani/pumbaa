// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"bufio"
	"context"
	"encoding/json"
	"html/template"
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
	SkipLLM    bool   // Skip LLM-based recommendations
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

	// Generate recommendations using LLM if available and not skipped
	var recommendationResult *ports.RecommendationResult
	if !input.SkipLLM && uc.recommendationGenerator != nil && uc.recommendationGenerator.IsAvailable() {
		// Convert TaskData to TaskAnalysisData for the generator
		analysisData := uc.convertToAnalysisData(validData)
		if len(analysisData) > 0 {
			var err error
			recommendationResult, err = uc.recommendationGenerator.GenerateRecommendations(ctx, analysisData)
			if err != nil {
				// Log error but don't fail - just skip recommendations
				recommendationResult = &ports.RecommendationResult{}
			}
		}
	}
	// Note: If generator not available, recommendationResult will be nil

	// Serialize recommendation result (includes summary and recommendations)
	recommendationsJSON, err := json.Marshal(recommendationResult)
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
