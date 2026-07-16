// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sort"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// ResourceVisualizationUseCase handles resource visualization report generation.
type ResourceVisualizationUseCase struct {
	metricsReader           ports.TaskMetricsReader
	recommendationGenerator ports.RecommendationGenerator
	reportRenderer          ports.ResourceReportRenderer
	debugWriterFactory      ports.LLMDebugWriterFactory
}

// NewResourceVisualizationUseCase creates a new resource visualization use case.
// metricsReader and renderer are required; generator can be nil if LLM is not
// configured, and debugWriterFactory can be nil to disable LLM debug logging.
func NewResourceVisualizationUseCase(metricsReader ports.TaskMetricsReader, generator ports.RecommendationGenerator, renderer ports.ResourceReportRenderer, debugWriterFactory ports.LLMDebugWriterFactory) *ResourceVisualizationUseCase {
	return &ResourceVisualizationUseCase{
		metricsReader:           metricsReader,
		recommendationGenerator: generator,
		reportRenderer:          renderer,
		debugWriterFactory:      debugWriterFactory,
	}
}

// ResourceVisualizationInput represents the input for resource visualization.
type ResourceVisualizationInput struct {
	Directory    string // Directory containing TSV files
	OutputFile   string // Output HTML file (default: "resource_report.html")
	SkipLLM      bool   // Skip LLM-based recommendations
	LLMBatchSize int    // Number of tasks per LLM request (Batching)
	LLMDebugFile string // Optional file path to write LLM debug logs (empty = no debug)
}

// ResourceVisualizationOutput contains the result of visualization generation.
type ResourceVisualizationOutput struct {
	OutputFile    string
	WorkflowCount int
	TaskCount     int
}

// TaskDataDTO is a data transfer object for task data JSON serialization.
type TaskDataDTO struct {
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
	DiskPeakBytes        int64            `json:"diskPeakBytes"`
	Error                string           `json:"error"`
	WorkflowID           string           `json:"workflowId"`
}

// Execute generates the HTML visualization report.
func (uc *ResourceVisualizationUseCase) Execute(ctx context.Context, input ResourceVisualizationInput) (*ResourceVisualizationOutput, error) {
	// 1. Validate input
	if input.Directory == "" {
		return nil, application.NewInputValidationError("directory", "is required")
	}

	outputFile := input.OutputFile
	if outputFile == "" {
		outputFile = "resource_report.html"
	}

	log.Printf("[resource_visualization] Starting analysis of directory: %s", input.Directory)

	// 2. Read metrics from directory (delegated to metricsReader)
	log.Printf("[resource_visualization] Reading TSV files from directory...")
	collection, workflows, err := uc.metricsReader.ReadFromDirectory(input.Directory)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to read TSV files", err)
	}

	if len(workflows) == 0 {
		return nil, application.NewUseCaseError("resource_visualization", "no TSV files found in directory", nil)
	}

	log.Printf("[resource_visualization] Found %d workflow(s) with %d total task records", len(workflows), collection.Len())

	if collection.Len() == 0 {
		return nil, application.NewUseCaseError("resource_visualization", "no valid data found in TSV files", nil)
	}

	// 3. Filter valid executions (delegated to domain)
	log.Printf("[resource_visualization] Filtering out execution errors...")
	validCollection := collection.FilterByValidExecution()
	log.Printf("[resource_visualization] %d records after filtering (removed %d with execution errors)",
		validCollection.Len(), collection.Len()-validCollection.Len())

	// 4. Convert to DTOs for JSON serialization
	validDataDTOs := uc.toTaskDataDTOs(validCollection)

	dataJSON, err := json.Marshal(validDataDTOs)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode data as JSON", err)
	}

	workflowsJSON, err := json.Marshal(workflows)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode workflows as JSON", err)
	}

	// 5. Generate recommendations
	var recommendationResult *ports.RecommendationResult
	var llmModelInfo string
	analysisData := uc.toAnalysisData(validCollection)
	log.Printf("[resource_visualization] Aggregated into %d unique tasks for analysis", len(analysisData))

	if !input.SkipLLM && uc.recommendationGenerator != nil && uc.recommendationGenerator.IsAvailable() {
		llmModelInfo = uc.recommendationGenerator.ModelInfo()
		log.Printf("[resource_visualization] Using LLM for recommendations: %s", llmModelInfo)

		// Set up debug writer if configured
		var debugWriter ports.LLMDebugWriter
		if input.LLMDebugFile != "" && uc.debugWriterFactory != nil {
			var debugErr error
			debugWriter, debugErr = uc.debugWriterFactory(input.LLMDebugFile)
			if debugErr != nil {
				log.Printf("[resource_visualization] Warning: failed to create debug writer: %v", debugErr)
			} else {
				uc.recommendationGenerator.SetDebugWriter(debugWriter)
				defer func() {
					debugWriter.Close()
					uc.recommendationGenerator.SetDebugWriter(nil)
				}()
				log.Printf("[resource_visualization] LLM debug logging enabled: %s", input.LLMDebugFile)
			}
		}

		// Use LLM to generate recommendations
		if len(analysisData) > 0 {
			log.Printf("[resource_visualization] Generating LLM recommendations for %d tasks (batch size: %d)...",
				len(analysisData), input.LLMBatchSize)
			recommendationResult, err = uc.recommendationGenerator.GenerateRecommendations(ctx, analysisData, input.LLMBatchSize)
			if err != nil {
				log.Printf("[resource_visualization] LLM recommendation failed: %v. Falling back to basic stats.", err)
				recommendationResult = uc.generateBasicStats(validCollection)
				llmModelInfo = "" // Clear model info since we fell back
			} else {
				log.Printf("[resource_visualization] LLM generated %d recommendations", len(recommendationResult.Recommendations))
			}
		}
	} else if len(analysisData) > 0 {
		// LLM not available or skipped - generate basic statistics
		if input.SkipLLM {
			log.Printf("[resource_visualization] LLM skipped by user request, generating basic statistics...")
		} else {
			log.Printf("[resource_visualization] LLM not available, generating basic statistics...")
		}
		recommendationResult = uc.generateBasicStats(validCollection)
	}

	// 6. Ensure all tasks are included in results
	if recommendationResult != nil {
		recommendationResult = uc.ensureAllTasksIncluded(recommendationResult, analysisData)
		log.Printf("[resource_visualization] Final recommendation count: %d tasks", len(recommendationResult.Recommendations))
	}

	recommendationsJSON, err := json.Marshal(recommendationResult)
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to encode recommendations as JSON", err)
	}

	// 7. Render report
	log.Printf("[resource_visualization] Rendering HTML report...")
	html, err := uc.reportRenderer.Render(ports.ResourceReportData{
		DataJSON:            dataJSON,
		WorkflowsJSON:       workflowsJSON,
		RecommendationsJSON: recommendationsJSON,
		LLMModelInfo:        llmModelInfo,
	})
	if err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to render HTML template", err)
	}

	// 8. Write HTML to file
	log.Printf("[resource_visualization] Writing report to: %s", outputFile)
	if err := os.WriteFile(outputFile, []byte(html), 0644); err != nil {
		return nil, application.NewUseCaseError("resource_visualization", "failed to write HTML file", err)
	}

	log.Printf("[resource_visualization] Report generation complete!")

	return &ResourceVisualizationOutput{
		OutputFile:    outputFile,
		WorkflowCount: len(workflows),
		TaskCount:     collection.UniqueTaskNames(),
	}, nil
}

// toTaskDataDTOs converts domain metrics to DTOs for JSON serialization.
func (uc *ResourceVisualizationUseCase) toTaskDataDTOs(collection *workflow.TaskMetricsCollection) []TaskDataDTO {
	metrics := collection.Metrics()
	dtos := make([]TaskDataDTO, len(metrics))
	for i, m := range metrics {
		dtos[i] = TaskDataDTO{
			TaskName:             m.TaskName,
			ShardIndex:           m.ShardIndex,
			CPURequest:           m.CPURequest,
			MemoryRequestBytes:   m.MemoryRequestBytes,
			DiskSizeRequestBytes: m.DiskSizeRequestBytes,
			DiskType:             m.DiskType,
			TotalInputBytes:      m.TotalInputBytes,
			Inputs:               m.Inputs,
			DurationSeconds:      m.DurationSeconds,
			CPUMean:              m.CPUMean,
			MemoryPeakMB:         m.MemoryPeakMB,
			DiskPeakBytes:        m.DiskPeakBytes,
			Error:                m.Error,
			WorkflowID:           m.WorkflowID,
		}
	}
	return dtos
}

// toAnalysisData converts domain aggregated metrics to ports.TaskAnalysisData.
func (uc *ResourceVisualizationUseCase) toAnalysisData(collection *workflow.TaskMetricsCollection) []ports.TaskAnalysisData {
	aggregated := collection.ToAggregatedMetrics()
	result := make([]ports.TaskAnalysisData, len(aggregated))
	for i, agg := range aggregated {
		result[i] = ports.TaskAnalysisData{
			TaskName:        agg.TaskName,
			SampleCount:     agg.SampleCount,
			CPURequest:      agg.CPURequest,
			MemoryReqGB:     agg.MemoryReqGB,
			DiskReqGB:       agg.DiskReqGB,
			DiskPeaksGB:     agg.DiskPeaksGB,
			MemoryPeaksMB:   agg.MemoryPeaksMB,
			CPUMeans:        agg.CPUMeans,
			DurationSeconds: agg.DurationSeconds,
			InputSizes:      agg.InputSizes,
			ResourceCost:    agg.ResourceCost,
		}
	}
	return result
}

// generateBasicStats creates recommendation cards with basic resource efficiency statistics.
func (uc *ResourceVisualizationUseCase) generateBasicStats(collection *workflow.TaskMetricsCollection) *ports.RecommendationResult {
	stats := collection.CalculateEfficiencyStats()
	recommendations := make([]ports.TaskRecommendation, len(stats))
	for i, s := range stats {
		recommendations[i] = ports.TaskRecommendation{
			TaskName:      s.TaskName,
			SampleCount:   s.SampleCount,
			OverallStatus: ports.RecommendationSeverity(s.OverallStatus),
			ResourceCost:  s.ResourceCost,
		}
	}
	return &ports.RecommendationResult{
		Summary:         "Basic resource usage metrics. Enable the LLM to receive detailed recommendations.",
		Recommendations: recommendations,
	}
}

// ensureAllTasksIncluded ensures all tasks from analysisData are present in recommendations.
func (uc *ResourceVisualizationUseCase) ensureAllTasksIncluded(result *ports.RecommendationResult, analysisData []ports.TaskAnalysisData) *ports.RecommendationResult {
	includedTasks := make(map[string]bool)
	for _, rec := range result.Recommendations {
		includedTasks[rec.TaskName] = true
	}

	for _, task := range analysisData {
		if !includedTasks[task.TaskName] {
			// Create a default "Info" recommendation for tasks skipped by LLM
			result.Recommendations = append(result.Recommendations, ports.TaskRecommendation{
				TaskName:      task.TaskName,
				SampleCount:   task.SampleCount,
				OverallStatus: ports.SeverityGood,
				ResourceCost:  task.ResourceCost,
				Recommendations: []ports.RecommendationItem{
					{
						Message:  "No specific optimization recommendations generated.",
						Severity: ports.SeverityGood,
					},
				},
			})
		}
	}

	// Sort recommendations by cost (highest first)
	sort.Slice(result.Recommendations, func(i, j int) bool {
		return result.Recommendations[i].ResourceCost > result.Recommendations[j].ResourceCost
	})

	return result
}
