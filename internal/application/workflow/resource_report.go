// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/ports"
	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// ResourceReportUseCase handles resource usage analysis for all tasks in a workflow.
type ResourceReportUseCase struct {
	metadataReader ports.WorkflowMetadataReader
	fileProvider   ports.FileProvider
}

// NewResourceReportUseCase creates a new resource report use case.
func NewResourceReportUseCase(reader ports.WorkflowMetadataReader, fp ports.FileProvider) *ResourceReportUseCase {
	return &ResourceReportUseCase{
		metadataReader: reader,
		fileProvider:   fp,
	}
}

// ResourceReportInput represents the input for resource report generation.
type ResourceReportInput struct {
	WorkflowID  string
	Concurrency int // Number of concurrent workers (default: 5)
}

// TaskResourceReport contains resource metrics for a single task.
type TaskResourceReport struct {
	TaskName      string
	ShardIndex    int    // -1 if not sharded
	CPURequest    string // Configured CPU (from runtime attributes)
	MemoryRequest string // Configured memory (from runtime attributes)
	DiskRequest   string // Configured disk (from runtime attributes)

	TotalInputBytes int64
	CPUMean         float64
	MemoryPeakMB    float64
	DiskPeakGB      float64
	Error           string // Non-empty if failed to get metrics
}

// ResourceReportOutput contains the result of resource report generation.
type ResourceReportOutput struct {
	WorkflowID   string
	WorkflowName string
	Tasks        []TaskResourceReport
	OutputFile   string
}

// ProgressCallback is called to report progress during execution.
type ProgressCallback func(completed, total int, currentTask string)

// fileSizeCache provides thread-safe caching of file sizes.
type fileSizeCache struct {
	mu    sync.RWMutex
	sizes map[string]int64
}

func newFileSizeCache() *fileSizeCache {
	return &fileSizeCache{
		sizes: make(map[string]int64),
	}
}

func (c *fileSizeCache) get(path string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	size, ok := c.sizes[path]
	return size, ok
}

func (c *fileSizeCache) set(path string, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sizes[path] = size
}

// ExecuteWithProgress generates a resource report for all tasks in a workflow with progress reporting.
func (uc *ResourceReportUseCase) ExecuteWithProgress(ctx context.Context, input ResourceReportInput, progress ProgressCallback) (*ResourceReportOutput, error) {
	if input.WorkflowID == "" {
		return nil, application.NewInputValidationError("workflowID", "is required")
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

	// Collect all calls that have monitoring logs
	var callsToProcess []workflowDomain.Call
	for _, calls := range wf.Calls {
		for _, call := range calls {
			// Only process calls that have monitoring logs and are not cache hits
			if call.MonitoringLog != "" && !call.CacheHit {
				callsToProcess = append(callsToProcess, call)
			}
		}
	}

	if len(callsToProcess) == 0 {
		return &ResourceReportOutput{
			WorkflowID:   wf.ID,
			WorkflowName: wf.Name,
			Tasks:        []TaskResourceReport{},
			OutputFile:   fmt.Sprintf("%s.tsv", input.WorkflowID),
		}, nil
	}

	// Create a cache for file sizes to avoid redundant GCS queries
	sizeCache := newFileSizeCache()

	// Process calls concurrently
	results := make([]TaskResourceReport, len(callsToProcess))
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

			report := uc.processCall(ctx, c, sizeCache)
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

	// Sort results by task name, then by shard index for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].TaskName != results[j].TaskName {
			return results[i].TaskName < results[j].TaskName
		}
		return results[i].ShardIndex < results[j].ShardIndex
	})

	// Generate output file
	outputFile := fmt.Sprintf("%s.tsv", input.WorkflowID)
	if err := uc.writeTSV(outputFile, results); err != nil {
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

// processCall processes a single call and returns its resource report.
func (uc *ResourceReportUseCase) processCall(ctx context.Context, call workflowDomain.Call, sizeCache *fileSizeCache) TaskResourceReport {
	taskName := extractTaskName(call.Name)

	// Calculate total input bytes from actual file sizes
	totalInputBytes := uc.calculateInputFileSizes(ctx, call.Inputs, sizeCache)

	// Base report with task identification and configuration
	baseReport := TaskResourceReport{
		TaskName:        taskName,
		ShardIndex:      call.ShardIndex,
		CPURequest:      call.CPU,
		MemoryRequest:   call.Memory,
		DiskRequest:     call.Disk,
		TotalInputBytes: totalInputBytes,
	}

	// Read and parse monitoring log
	content, err := uc.fileProvider.Read(ctx, call.MonitoringLog)
	if err != nil {
		baseReport.Error = fmt.Sprintf("failed to read monitoring log: %v", err)
		return baseReport
	}

	metrics, err := workflowDomain.ParseMonitoringTSV(content)
	if err != nil {
		baseReport.Error = fmt.Sprintf("failed to parse monitoring log: %v", err)
		return baseReport
	}

	report := metrics.Analyze()

	baseReport.CPUMean = report.CPU.Avg
	baseReport.MemoryPeakMB = report.Mem.Peak
	baseReport.DiskPeakGB = report.Disk.Peak

	return baseReport
}

// calculateInputFileSizes calculates the total size of input files.
// It extracts GCS paths from the inputs and queries their sizes.
func (uc *ResourceReportUseCase) calculateInputFileSizes(ctx context.Context, inputs map[string]interface{}, cache *fileSizeCache) int64 {
	if inputs == nil {
		return 0
	}

	// Extract all file paths from inputs
	paths := extractFilePaths(inputs)
	if len(paths) == 0 {
		return 0
	}

	var total int64
	for _, path := range paths {
		// Check cache first
		if size, ok := cache.get(path); ok {
			total += size
			continue
		}

		// Query file size
		size, err := uc.fileProvider.GetSize(ctx, path)
		if err != nil {
			// Skip files that can't be accessed (might be deleted or inaccessible)
			continue
		}

		// Cache the result
		cache.set(path, size)
		total += size
	}

	return total
}

// extractFilePaths recursively extracts all file paths (gs:// or local) from a value.
func extractFilePaths(value interface{}) []string {
	var paths []string
	extractFilePathsRecursive(value, &paths)
	return paths
}

// extractFilePathsRecursive is the recursive helper for extractFilePaths.
func extractFilePathsRecursive(value interface{}, paths *[]string) {
	switch v := value.(type) {
	case string:
		// Check if it's a GCS path or a local file path
		if isFilePath(v) {
			*paths = append(*paths, v)
		}
	case []interface{}:
		for _, item := range v {
			extractFilePathsRecursive(item, paths)
		}
	case map[string]interface{}:
		for _, val := range v {
			extractFilePathsRecursive(val, paths)
		}
	}
}

// isFilePath checks if a string looks like a file path.
// Returns true for GCS paths (gs://) and paths that look like file paths.
func isFilePath(s string) bool {
	// GCS paths
	if strings.HasPrefix(s, "gs://") {
		return true
	}
	// Local absolute paths (Unix)
	if strings.HasPrefix(s, "/") && strings.Contains(s, ".") {
		return true
	}
	// S3 paths (for future support)
	if strings.HasPrefix(s, "s3://") {
		return true
	}
	return false
}

// extractTaskName removes the workflow prefix from task name.
func extractTaskName(fullName string) string {
	// Remove workflow prefix if present (e.g., "MyWorkflow.task_name" -> "task_name")
	if idx := strings.LastIndex(fullName, "."); idx != -1 {
		return fullName[idx+1:]
	}
	return fullName
}

// writeTSV writes the resource report to a TSV file.
func (uc *ResourceReportUseCase) writeTSV(filename string, tasks []TaskResourceReport) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	_, err = fmt.Fprintln(file, "task_name\tshard_index\tcpu_request\tmemory_request\tdisk_request\ttotal_bytes_input\tcpu_mean\tmemory_peak_mb\tdisk_peak_gb\terror")
	if err != nil {
		return err
	}

	// Write data rows
	for _, task := range tasks {
		_, err = fmt.Fprintf(file, "%s\t%d\t%s\t%s\t%s\t%d\t%.2f\t%.2f\t%.2f\t%s\n",
			task.TaskName,
			task.ShardIndex,
			task.CPURequest,
			task.MemoryRequest,
			task.DiskRequest,
			task.TotalInputBytes,
			task.CPUMean,
			task.MemoryPeakMB,
			task.DiskPeakGB,
			task.Error,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
