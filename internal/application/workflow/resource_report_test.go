package workflow

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// validMonitoringTSV is a valid monitoring log content for testing.
const validMonitoringTSV = `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb	disk_total_gb
2023-01-01 00:00:00	10.0	512.0	8192.0	5.0	100.0
2023-01-01 00:01:00	20.0	1024.0	8192.0	10.0	100.0
2023-01-01 00:02:00	30.0	2048.0	8192.0	15.0	100.0`

func TestResourceReportUseCase_Execute_Validation(t *testing.T) {
	uc := NewResourceReportUseCase(&mockWorkflowRepository{}, &mockFileProvider{})

	_, err := uc.Execute(context.Background(), ResourceReportInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	var inputErr *application.InputValidationError
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected InputValidationError, got %T", err)
	}
	if inputErr.Field != "workflowID" {
		t.Errorf("expected field workflowID, got %s", inputErr.Field)
	}
}

func TestResourceReportUseCase_Execute_MetadataError(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return nil, errors.New("metadata fetch failed")
		},
	}
	uc := NewResourceReportUseCase(repo, &mockFileProvider{})

	_, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, application.ErrOperationFailed) {
		t.Errorf("expected ErrOperationFailed, got %v", err)
	}
	var ucErr *application.UseCaseError
	if !errors.As(err, &ucErr) {
		t.Fatalf("expected UseCaseError, got %T", err)
	}
	if ucErr.Operation != "resource_report" {
		t.Errorf("expected operation resource_report, got %s", ucErr.Operation)
	}
}

func TestResourceReportUseCase_Execute_NoTasksWithMonitoringLogs(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{Name: "TestWorkflow.task1", ShardIndex: -1, MonitoringLog: ""}, // No monitoring log
					},
				},
			}, nil
		},
	}
	uc := NewResourceReportUseCase(repo, &mockFileProvider{})

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(output.Tasks))
	}
	if output.OutputFile != "test-workflow.tsv" {
		t.Errorf("expected output file test-workflow.tsv, got %s", output.OutputFile)
	}
}

func TestResourceReportUseCase_Execute_CacheHitsIgnored(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    -1,
							MonitoringLog: "gs://bucket/monitoring.log",
							CacheHit:      true, // Should be ignored
						},
					},
				},
			}, nil
		},
	}
	uc := NewResourceReportUseCase(repo, &mockFileProvider{})

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Tasks) != 0 {
		t.Errorf("expected 0 tasks (cache hits should be ignored), got %d", len(output.Tasks))
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_Execute_Success(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    -1,
							MonitoringLog: "gs://bucket/task1/monitoring.log",
							CPU:           "4",
							Memory:        "8 GB",
							Disk:          "100 GB",
							Inputs: map[string]interface{}{
								"input_file": "gs://bucket/input.txt",
							},
						},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
		getSizeFunc: func(ctx context.Context, path string) (int64, error) {
			// Return a mock file size
			return 1024, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(output.Tasks))
	}

	task := output.Tasks[0]
	if task.TaskName != "task1" {
		t.Errorf("expected task name 'task1', got '%s'", task.TaskName)
	}
	if task.ShardIndex != -1 {
		t.Errorf("expected shard index -1, got %d", task.ShardIndex)
	}
	if task.CPURequest != "4" {
		t.Errorf("expected CPU request '4', got '%s'", task.CPURequest)
	}
	if task.MemoryRequest != "8 GB" {
		t.Errorf("expected memory request '8 GB', got '%s'", task.MemoryRequest)
	}
	if task.DiskRequest != "100 GB" {
		t.Errorf("expected disk request '100 GB', got '%s'", task.DiskRequest)
	}
	if task.Error != "" {
		t.Errorf("expected no error, got '%s'", task.Error)
	}
	// CPU mean should be (10+20+30)/3 = 20
	if task.CPUMean != 20.0 {
		t.Errorf("expected CPU mean 20.0, got %f", task.CPUMean)
	}
	// Memory peak should be 2048
	if task.MemoryPeakMB != 2048.0 {
		t.Errorf("expected memory peak 2048.0, got %f", task.MemoryPeakMB)
	}
	// Disk peak should be 15
	if task.DiskPeakGB != 15.0 {
		t.Errorf("expected disk peak 15.0, got %f", task.DiskPeakGB)
	}
	// Total input bytes should be 1024 (from mock)
	if task.TotalInputBytes != 1024 {
		t.Errorf("expected total input bytes 1024, got %d", task.TotalInputBytes)
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_Execute_ReadError(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    -1,
							MonitoringLog: "gs://bucket/task1/monitoring.log",
							CPU:           "4",
							Memory:        "8 GB",
							Disk:          "100 GB",
						},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return "", errors.New("file not found")
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return the task, but with an error message
	if len(output.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(output.Tasks))
	}

	task := output.Tasks[0]
	if task.Error == "" {
		t.Error("expected error message in task, got empty")
	}
	if !strings.Contains(task.Error, "failed to read monitoring log") {
		t.Errorf("expected error to contain 'failed to read monitoring log', got '%s'", task.Error)
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_Execute_ParseError(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    -1,
							MonitoringLog: "gs://bucket/task1/monitoring.log",
						},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			// Invalid TSV - missing required columns
			return "timestamp\tcpu_percent\n", nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(output.Tasks))
	}

	task := output.Tasks[0]
	if task.Error == "" {
		t.Error("expected error message in task, got empty")
	}
	if !strings.Contains(task.Error, "failed to parse monitoring log") {
		t.Errorf("expected error to contain 'failed to parse monitoring log', got '%s'", task.Error)
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_Execute_SortOrder(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task_b": {
						{Name: "TestWorkflow.task_b", ShardIndex: 1, MonitoringLog: "gs://bucket/b1.log"},
						{Name: "TestWorkflow.task_b", ShardIndex: 0, MonitoringLog: "gs://bucket/b0.log"},
					},
					"TestWorkflow.task_a": {
						{Name: "TestWorkflow.task_a", ShardIndex: -1, MonitoringLog: "gs://bucket/a.log"},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(output.Tasks))
	}

	// Should be sorted by task name, then shard index
	expectedOrder := []struct {
		name  string
		shard int
	}{
		{"task_a", -1},
		{"task_b", 0},
		{"task_b", 1},
	}

	for i, expected := range expectedOrder {
		if output.Tasks[i].TaskName != expected.name {
			t.Errorf("task %d: expected name '%s', got '%s'", i, expected.name, output.Tasks[i].TaskName)
		}
		if output.Tasks[i].ShardIndex != expected.shard {
			t.Errorf("task %d: expected shard %d, got %d", i, expected.shard, output.Tasks[i].ShardIndex)
		}
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_ExecuteWithProgress(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{Name: "TestWorkflow.task1", ShardIndex: 0, MonitoringLog: "gs://bucket/1.log"},
						{Name: "TestWorkflow.task1", ShardIndex: 1, MonitoringLog: "gs://bucket/2.log"},
						{Name: "TestWorkflow.task1", ShardIndex: 2, MonitoringLog: "gs://bucket/3.log"},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	var progressCalls int64
	progress := func(completed, total int, currentTask string) {
		atomic.AddInt64(&progressCalls, 1)
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
	}

	output, err := uc.ExecuteWithProgress(context.Background(), ResourceReportInput{
		WorkflowID:  "test-workflow",
		Concurrency: 2,
	}, progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(output.Tasks))
	}

	// Progress should be called 3 times (once per task)
	if progressCalls != 3 {
		t.Errorf("expected 3 progress calls, got %d", progressCalls)
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_Execute_DefaultConcurrency(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{Name: "TestWorkflow.task1", ShardIndex: -1, MonitoringLog: "gs://bucket/1.log"},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	// Concurrency = 0 should default to 5
	output, err := uc.Execute(context.Background(), ResourceReportInput{
		WorkflowID:  "test-workflow",
		Concurrency: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(output.Tasks))
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected []string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: []string{},
		},
		{
			name:     "empty map",
			value:    map[string]interface{}{},
			expected: []string{},
		},
		{
			name: "single GCS path",
			value: map[string]interface{}{
				"file": "gs://bucket/file.txt",
			},
			expected: []string{"gs://bucket/file.txt"},
		},
		{
			name: "array of GCS paths",
			value: map[string]interface{}{
				"files": []interface{}{
					"gs://bucket/file1.txt",
					"gs://bucket/file2.txt",
				},
			},
			expected: []string{"gs://bucket/file1.txt", "gs://bucket/file2.txt"},
		},
		{
			name: "nested map with GCS paths",
			value: map[string]interface{}{
				"reference_fasta": map[string]interface{}{
					"ref_fasta": "gs://bucket/ref.fasta",
					"ref_index": "gs://bucket/ref.fasta.fai",
				},
			},
			expected: []string{"gs://bucket/ref.fasta", "gs://bucket/ref.fasta.fai"},
		},
		{
			name: "mixed values (only paths extracted)",
			value: map[string]interface{}{
				"sample_name": "sample1",
				"count":       42,
				"enabled":     true,
				"input_file":  "gs://bucket/input.bam",
			},
			expected: []string{"gs://bucket/input.bam"},
		},
		{
			name: "local absolute path",
			value: map[string]interface{}{
				"local_file": "/path/to/file.txt",
			},
			expected: []string{"/path/to/file.txt"},
		},
		{
			name: "S3 path",
			value: map[string]interface{}{
				"s3_file": "s3://bucket/file.txt",
			},
			expected: []string{"s3://bucket/file.txt"},
		},
		{
			name: "non-path strings (should not be extracted)",
			value: map[string]interface{}{
				"name":    "sample1",
				"command": "echo hello",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilePaths(tt.value)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d paths, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			// Check that all expected paths are present (order may vary for maps)
			for _, exp := range tt.expected {
				found := false
				for _, res := range result {
					if res == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected path '%s' not found in result: %v", exp, result)
				}
			}
		})
	}
}

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"gs://bucket/file.txt", true},
		{"gs://bucket/path/to/file.bam", true},
		{"s3://bucket/file.txt", true},
		{"/path/to/file.txt", true},
		{"/absolute/path/file.bam", true},
		{"sample_name", false},
		{"echo hello", false},
		{"42", false},
		{"", false},
		{"relative/path.txt", false}, // Relative paths are not detected
		{"/no-extension", false},     // Paths without extension
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("isFilePath(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractTaskName(t *testing.T) {
	tests := []struct {
		fullName string
		expected string
	}{
		{"MyWorkflow.task1", "task1"},
		{"task1", "task1"},
		{"Nested.Workflow.task1", "task1"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			result := extractTaskName(tt.fullName)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestResourceReportUseCase_TSVOutput(t *testing.T) {
	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    0,
							MonitoringLog: "gs://bucket/task1/monitoring.log",
							CPU:           "4",
							Memory:        "8 GB",
							Disk:          "100 GB",
							Inputs: map[string]interface{}{
								"input": "gs://bucket/input.bam",
							},
						},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
		getSizeFunc: func(ctx context.Context, path string) (int64, error) {
			return 2048, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{WorkflowID: "test-workflow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the generated TSV file
	content, err := os.ReadFile(output.OutputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + data), got %d", len(lines))
	}

	// Check header
	expectedHeader := "task_name\tshard_index\tcpu_request\tmemory_request\tdisk_request\ttotal_bytes_input\tcpu_mean\tmemory_peak_mb\tdisk_peak_gb\terror"
	if lines[0] != expectedHeader {
		t.Errorf("expected header:\n%s\ngot:\n%s", expectedHeader, lines[0])
	}

	// Check data row has correct number of columns
	dataColumns := strings.Split(lines[1], "\t")
	if len(dataColumns) != 10 {
		t.Errorf("expected 10 columns in data row, got %d", len(dataColumns))
	}

	// Verify some column values
	if dataColumns[0] != "task1" {
		t.Errorf("expected task_name 'task1', got '%s'", dataColumns[0])
	}
	if dataColumns[1] != "0" {
		t.Errorf("expected shard_index '0', got '%s'", dataColumns[1])
	}
	if dataColumns[2] != "4" {
		t.Errorf("expected cpu_request '4', got '%s'", dataColumns[2])
	}
	// Check total_bytes_input is 2048 (from mock GetSize)
	if dataColumns[5] != "2048" {
		t.Errorf("expected total_bytes_input '2048', got '%s'", dataColumns[5])
	}

	// Cleanup
	os.Remove(output.OutputFile)
}

func TestResourceReportUseCase_FileSizeCache(t *testing.T) {
	// Track GetSize calls to verify caching
	var getSizeCalls int64

	repo := &mockWorkflowRepository{
		getMetadataFunc: func(ctx context.Context, workflowID string) (*workflow.Workflow, error) {
			return &workflow.Workflow{
				ID:   workflowID,
				Name: "TestWorkflow",
				Calls: map[string][]workflow.Call{
					"TestWorkflow.task1": {
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    0,
							MonitoringLog: "gs://bucket/task1/monitoring.log",
							Inputs: map[string]interface{}{
								"shared_file": "gs://bucket/shared.bam", // Same file
							},
						},
						{
							Name:          "TestWorkflow.task1",
							ShardIndex:    1,
							MonitoringLog: "gs://bucket/task2/monitoring.log",
							Inputs: map[string]interface{}{
								"shared_file": "gs://bucket/shared.bam", // Same file - should use cache
							},
						},
					},
				},
			}, nil
		},
	}

	fp := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return validMonitoringTSV, nil
		},
		getSizeFunc: func(ctx context.Context, path string) (int64, error) {
			atomic.AddInt64(&getSizeCalls, 1)
			return 1024, nil
		},
	}
	uc := NewResourceReportUseCase(repo, fp)

	output, err := uc.Execute(context.Background(), ResourceReportInput{
		WorkflowID:  "test-workflow",
		Concurrency: 1, // Sequential to ensure cache is tested properly
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GetSize should be called only once for the shared file due to caching
	if getSizeCalls != 1 {
		t.Errorf("expected 1 GetSize call (cache should work), got %d", getSizeCalls)
	}

	// Both tasks should have the same input size
	for _, task := range output.Tasks {
		if task.TotalInputBytes != 1024 {
			t.Errorf("expected total input bytes 1024, got %d for task %s.%d", task.TotalInputBytes, task.TaskName, task.ShardIndex)
		}
	}

	// Cleanup
	os.Remove(output.OutputFile)
}
