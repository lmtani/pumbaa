// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// ResourceReportUseCase handles resource usage analysis for all tasks in a workflow.
type ResourceReportUseCase struct {
	metadataReader ports.WorkflowMetadataReader
	fileProvider   ports.FileProvider
	metricsWriter  ports.TaskMetricsWriter
	sizeCache      ports.FileSizeCache
}

// NewResourceReportUseCase creates a new resource report use case.
func NewResourceReportUseCase(reader ports.WorkflowMetadataReader, fp ports.FileProvider, writer ports.TaskMetricsWriter, cache ports.FileSizeCache) *ResourceReportUseCase {
	return &ResourceReportUseCase{
		metadataReader: reader,
		fileProvider:   fp,
		metricsWriter:  writer,
		sizeCache:      cache,
	}
}

// ResourceReportInput represents the input for resource report generation.
type ResourceReportInput struct {
	WorkflowID  string
	Concurrency int // Number of concurrent workers (default: 5)
}

// ResourceReportOutput contains the result of resource report generation.
type ResourceReportOutput struct {
	WorkflowID   string
	WorkflowName string
	Tasks        []workflowDomain.TaskMetrics
	OutputFile   string
}

// ProgressCallback is called to report progress during execution.
type ProgressCallback func(completed, total int, currentTask string)

// ExecuteWithProgress generates a resource report for all tasks in a workflow with progress reporting.
func (uc *ResourceReportUseCase) ExecuteWithProgress(ctx context.Context, input ResourceReportInput, progress ProgressCallback) (*ResourceReportOutput, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
	}

	if uc.metricsWriter == nil {
		return nil, application.NewUseCaseError("resource_report", "task metrics writer is required", nil)
	}
	if uc.sizeCache == nil {
		return nil, application.NewUseCaseError("resource_report", "file size cache is required", nil)
	}

	concurrency := input.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	// Get workflow metadata
	wf, err := uc.metadataReader.GetMetadata(ctx, input.WorkflowID)
	if err != nil {
		return nil, application.NewUseCaseError("resource_report", "failed to get workflow metadata", err)
	}

	// Collect all calls recursively (including from subworkflows)
	callsToProcess := uc.collectCallsRecursively(ctx, wf)

	if len(callsToProcess) == 0 {
		return &ResourceReportOutput{
			WorkflowID:   wf.ID,
			WorkflowName: wf.Name,
			Tasks:        []workflowDomain.TaskMetrics{},
			OutputFile:   fmt.Sprintf("%s.tsv", input.WorkflowID),
		}, nil
	}

	sizeCache := uc.sizeCache
	_ = sizeCache.Load()

	// Process calls concurrently
	results := make([]workflowDomain.TaskMetrics, len(callsToProcess))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	var completedCount int
	var mu sync.Mutex

	for i, call := range callsToProcess {
		wg.Add(1)
		go func(idx int, c workflowDomain.Call) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			report := uc.processCall(ctx, wf.ID, c, sizeCache)
			results[idx] = report

			mu.Lock()
			completedCount++
			if progress != nil {
				progress(completedCount, len(callsToProcess), c.Name)
			}
			mu.Unlock()
		}(i, call)
	}

	wg.Wait()

	// Save cache to disk for future use
	_ = sizeCache.Save()

	// Sort results by task name, then by shard index for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].TaskName != results[j].TaskName {
			return results[i].TaskName < results[j].TaskName
		}
		return results[i].ShardIndex < results[j].ShardIndex
	})

	// Generate output file
	outputFile := fmt.Sprintf("%s.tsv", input.WorkflowID)
	if err := uc.metricsWriter.WriteToFile(outputFile, results); err != nil {
		return nil, application.NewUseCaseError("resource_report", "failed to write TSV file", err)
	}

	return &ResourceReportOutput{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Tasks:        results,
		OutputFile:   outputFile,
	}, nil
}

// Execute generates a resource report without progress reporting.
func (uc *ResourceReportUseCase) Execute(ctx context.Context, input ResourceReportInput) (*ResourceReportOutput, error) {
	return uc.ExecuteWithProgress(ctx, input, nil)
}

// collectCallsRecursively collects all calls with monitoring logs from a workflow,
// including calls from subworkflows (recursively).
func (uc *ResourceReportUseCase) collectCallsRecursively(ctx context.Context, wf *workflowDomain.Workflow) []workflowDomain.Call {
	var calls []workflowDomain.Call

	for _, callList := range wf.Calls {
		for _, call := range callList {
			// If this is a subworkflow, fetch its metadata and process recursively
			if call.SubWorkflowID != "" {
				subWf, err := uc.metadataReader.GetMetadata(ctx, call.SubWorkflowID)
				if err != nil {
					// Log error but continue with other calls
					continue
				}
				subCalls := uc.collectCallsRecursively(ctx, subWf)
				calls = append(calls, subCalls...)
			} else if call.MonitoringLog != "" && !call.CacheHit {
				// Regular task with monitoring log
				calls = append(calls, call)
			}
		}
	}

	return calls
}

// processCall processes a single call and returns its resource report.
func (uc *ResourceReportUseCase) processCall(ctx context.Context, workflowID string, call workflowDomain.Call, sizeCache ports.FileSizeCache) workflowDomain.TaskMetrics {
	// Calculate total input bytes and per-input sizes
	totalInputBytes, inputs := uc.calculateInputFileSizes(ctx, call.Inputs, sizeCache)
	baseReport := workflowDomain.NewTaskMetricsFromCall(call, workflowID, totalInputBytes, inputs)

	// Read and parse monitoring log
	content, err := uc.fileProvider.Read(ctx, call.MonitoringLog)
	if err != nil {
		baseReport.Error = fmt.Sprintf("failed to read monitoring log: %v", err)
		return baseReport
	}

	if content == "" {
		baseReport.Error = fmt.Sprintf("%s exists but no content", call.MonitoringLog)
		return baseReport
	}

	metrics, err := workflowDomain.ParseMonitoringTSV(content)
	if err != nil {
		baseReport.Error = fmt.Sprintf("failed to parse monitoring log: %v", err)
		return baseReport
	}

	report := metrics.Analyze()
	return baseReport.WithMonitoringReport(report)
}

// calculateInputFileSizes calculates the total size of input files and per-input sizes.
// It extracts file paths from the inputs and queries their sizes.
func (uc *ResourceReportUseCase) calculateInputFileSizes(ctx context.Context, inputs map[string]interface{}, cache ports.FileSizeCache) (int64, map[string]int64) {
	if inputs == nil {
		return 0, nil
	}

	total := int64(0)
	inputSizes := make(map[string]int64)

	for key, value := range inputs {
		// Clean the input key (remove workflow name prefix if likely present)
		inputName := workflowDomain.ExtractTaskName(key)

		// Use domain Value Object to extract file paths
		paths := workflowDomain.ExtractFilePaths(value)
		if len(paths) == 0 {
			continue
		}

		var keySize int64
		for _, path := range paths {
			pathStr := path.String()
			// Check cache first
			if size, ok := cache.Get(pathStr); ok {
				log.Printf("Cache hit for file size: %s (%d bytes)", pathStr, size)
				keySize += size
				continue
			}

			// Query file size
			size, err := uc.fileProvider.GetSize(ctx, pathStr)
			if err != nil {
				continue
			}

			// Cache the result
			cache.Set(pathStr, size)
			keySize += size
		}
		inputSizes[inputName] = keySize
		total += keySize
	}

	return total, inputSizes
}
