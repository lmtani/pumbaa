// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	TaskName             string
	ShardIndex           int    // -1 if not sharded
	CPURequest           string // Configured CPU (from runtime attributes)
	MemoryRequestBytes   int64  // Configured memory in bytes (parsed from runtime attributes)
	DiskSizeRequestBytes int64  // Configured disk size in bytes (parsed from runtime attributes)
	DiskType             string // Disk type (HDD, SSD, etc.)

	TotalInputBytes int64
	Inputs          map[string]int64 // Map of input name to total size in bytes
	DurationSeconds float64          // Task execution duration in seconds
	CPUMean         float64
	MemoryPeakMB    float64
	DiskPeakBytes   int64
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

// fileSizeCache provides thread-safe caching of file sizes with persistent storage.
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

// getCacheFilePath returns the path to the persistent cache file.
func getCacheFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pumbaa", "input_sizes.json")
}

// loadFromDisk loads the cache from the persistent file.
func (c *fileSizeCache) loadFromDisk() {
	path := getCacheFilePath()
	if path == "" {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist or can't be read - start fresh
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_ = json.Unmarshal(data, &c.sizes)
}

// saveToDisk persists the cache to the filesystem.
func (c *fileSizeCache) saveToDisk() {
	path := getCacheFilePath()
	if path == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	c.mu.RLock()
	data, err := json.Marshal(c.sizes)
	c.mu.RUnlock()
	if err != nil {
		return
	}

	_ = os.WriteFile(path, data, 0644)
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

	// Collect all calls recursively (including from subworkflows)
	callsToProcess := uc.collectCallsRecursively(ctx, wf)

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
	sizeCache.loadFromDisk() // Load previously cached sizes

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

	// Save cache to disk for future use
	sizeCache.saveToDisk()

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
func (uc *ResourceReportUseCase) processCall(ctx context.Context, call workflowDomain.Call, sizeCache *fileSizeCache) TaskResourceReport {
	taskName := extractTaskName(call.Name)

	// Calculate total input bytes and per-input sizes
	totalInputBytes, inputs := uc.calculateInputFileSizes(ctx, call.Inputs, sizeCache)

	// Parse memory and disk configurations using domain Value Objects
	memory := workflowDomain.Memory(call.Memory)
	disk := workflowDomain.NewDiskConfig(call.Disk)

	// Calculate duration from call timing
	var durationSeconds float64
	if !call.Start.IsZero() && !call.End.IsZero() {
		durationSeconds = call.End.Sub(call.Start).Seconds()
	}

	// Base report with task identification and configuration
	baseReport := TaskResourceReport{
		TaskName:             taskName,
		ShardIndex:           call.ShardIndex,
		CPURequest:           call.CPU,
		MemoryRequestBytes:   memory.ToBytes(),
		DiskSizeRequestBytes: disk.SizeBytes(),
		DiskType:             disk.Type(),
		TotalInputBytes:      totalInputBytes,
		Inputs:               inputs,
		DurationSeconds:      durationSeconds,
	}

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

	baseReport.CPUMean = report.CPU.Avg
	baseReport.MemoryPeakMB = report.Mem.Peak
	baseReport.DiskPeakBytes = int64(report.Disk.Peak * 1024 * 1024 * 1024)

	return baseReport
}

// calculateInputFileSizes calculates the total size of input files and per-input sizes.
// It extracts file paths from the inputs and queries their sizes.
func (uc *ResourceReportUseCase) calculateInputFileSizes(ctx context.Context, inputs map[string]interface{}, cache *fileSizeCache) (int64, map[string]int64) {
	if inputs == nil {
		return 0, nil
	}

	total := int64(0)
	inputSizes := make(map[string]int64)

	for key, value := range inputs {
		// Clean the input key (remove workflow name prefix if likely present)
		inputName := extractTaskName(key)

		// Use domain Value Object to extract file paths
		paths := workflowDomain.ExtractFilePaths(value)
		if len(paths) == 0 {
			continue
		}

		var keySize int64
		for _, path := range paths {
			pathStr := path.String()
			// Check cache first
			if size, ok := cache.get(pathStr); ok {
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
			cache.set(pathStr, size)
			keySize += size
		}
		inputSizes[inputName] = keySize
		total += keySize
	}

	return total, inputSizes
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
	_, err = fmt.Fprintln(file, "task_name\tshard_index\tcpu_request\tmemory_request_bytes\tdisk_size_request_bytes\tdisk_type\ttotal_bytes_input\tinputs_json\tduration_seconds\tcpu_mean\tmemory_peak_mb\tdisk_peak_bytes\terror")
	if err != nil {
		return err
	}

	// Write data rows
	for _, task := range tasks {
		inputsJSON, _ := json.Marshal(task.Inputs)
		// Clean up errors in inputsJSON marshalling by using empty object if needed, though map shouldn't fail
		if inputsJSON == nil {
			inputsJSON = []byte("{}")
		}

		// Sanitize error message to prevent newlines breaking TSV format
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
