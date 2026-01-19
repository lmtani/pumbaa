// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"golang.org/x/sync/errgroup"
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
	Warnings     []string // Non-fatal issues encountered during processing
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
	collectResult := uc.collectCalls(ctx, wf)
	callsToProcess := collectResult.calls
	warnings := collectResult.warnings

	if len(callsToProcess) == 0 {
		return &ResourceReportOutput{
			WorkflowID:   wf.ID,
			WorkflowName: wf.Name,
			Tasks:        []workflowDomain.TaskMetrics{},
			OutputFile:   fmt.Sprintf("%s.tsv", input.WorkflowID),
			Warnings:     warnings,
		}, nil
	}

	// Process calls concurrently
	results := make([]workflowDomain.TaskMetrics, len(callsToProcess))
	var completedCount int
	var mu sync.Mutex
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(concurrency)

	for i, call := range callsToProcess {
		idx := i
		c := call
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}

			report := uc.processCall(groupCtx, wf.ID, c, uc.sizeCache)
			if err := groupCtx.Err(); err != nil {
				return err
			}
			results[idx] = report

			mu.Lock()
			completedCount++
			if progress != nil {
				progress(completedCount, len(callsToProcess), c.Name)
			}
			mu.Unlock()

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, application.NewUseCaseError("resource_report", "resource report cancelled", err)
	}

	uc.sortTaskMetrics(results)

	outputFile, err := uc.writeReport(input.WorkflowID, results)
	if err != nil {
		return nil, application.NewUseCaseError("resource_report", "failed to write TSV file", err)
	}

	return &ResourceReportOutput{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Tasks:        results,
		OutputFile:   outputFile,
		Warnings:     warnings,
	}, nil
}

// Execute generates a resource report without progress reporting.
func (uc *ResourceReportUseCase) Execute(ctx context.Context, input ResourceReportInput) (*ResourceReportOutput, error) {
	return uc.ExecuteWithProgress(ctx, input, nil)
}

// collectCallsResult holds the result of collecting calls from a workflow.
type collectCallsResult struct {
	calls    []workflowDomain.Call
	warnings []string
}

// collectCalls collects all calls with monitoring logs from a workflow,
// including calls from subworkflows (recursively).
func (uc *ResourceReportUseCase) collectCalls(ctx context.Context, wf *workflowDomain.Workflow) collectCallsResult {
	var calls []workflowDomain.Call
	var warnings []string

	for _, callList := range wf.Calls {
		for _, call := range callList {
			// If this is a subworkflow, fetch its metadata and process recursively
			if call.SubWorkflowID != "" {
				subWf, err := uc.metadataReader.GetMetadata(ctx, call.SubWorkflowID)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("failed to fetch subworkflow %s: %v", call.SubWorkflowID, err))
					continue
				}
				subResult := uc.collectCalls(ctx, subWf)
				calls = append(calls, subResult.calls...)
				warnings = append(warnings, subResult.warnings...)
			} else if call.MonitoringLog != "" && !call.CacheHit {
				// Regular task with monitoring log
				calls = append(calls, call)
			}
		}
	}

	return collectCallsResult{calls: calls, warnings: warnings}
}

func (uc *ResourceReportUseCase) sortTaskMetrics(tasks []workflowDomain.TaskMetrics) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].TaskName != tasks[j].TaskName {
			return tasks[i].TaskName < tasks[j].TaskName
		}
		return tasks[i].ShardIndex < tasks[j].ShardIndex
	})
}

func (uc *ResourceReportUseCase) writeReport(workflowID string, tasks []workflowDomain.TaskMetrics) (string, error) {
	outputFile := fmt.Sprintf("%s.tsv", workflowID)
	if err := uc.metricsWriter.WriteToFile(outputFile, tasks); err != nil {
		return "", err
	}
	return outputFile, nil
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
