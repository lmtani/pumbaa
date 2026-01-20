package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTSVReader_ReadFromDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "tsv_reader_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test TSV file
	tsvContent := `task_name	shard_index	cpu_request	memory_request_bytes	disk_size_request_bytes	disk_type	total_bytes_input	duration_seconds	cpu_mean	memory_peak_mb	disk_peak_bytes	error
MyTask	0	2	1073741824	10737418240	HDD	512000	3600	50.5	512	5368709120
MyTask	1	2	1073741824	10737418240	HDD	1024000	7200	60.0	640	6442450944
OtherTask	0	4	2147483648	21474836480	SSD	2048000	1800	75.0	1024	10737418240	`

	tsvFile := filepath.Join(tmpDir, "workflow123.tsv")
	if err := os.WriteFile(tsvFile, []byte(tsvContent), 0644); err != nil {
		t.Fatalf("Failed to write TSV file: %v", err)
	}

	reader := NewTSVReader()
	collection, workflows, err := reader.ReadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ReadFromDirectory() error = %v", err)
	}

	// Check workflows
	if len(workflows) != 1 {
		t.Errorf("ReadFromDirectory() returned %d workflows, want 1", len(workflows))
	}
	if workflows[0] != "workflow123" {
		t.Errorf("Workflow ID = %s, want workflow123", workflows[0])
	}

	// Check collection
	if collection.Len() != 3 {
		t.Errorf("Collection has %d items, want 3", collection.Len())
	}

	// Check first metric
	metrics := collection.Metrics()
	found := false
	for _, m := range metrics {
		if m.TaskName == "MyTask" && m.ShardIndex == 0 {
			found = true
			if m.CPURequest != "2" {
				t.Errorf("CPURequest = %s, want 2", m.CPURequest)
			}
			if m.MemoryRequestBytes != 1073741824 {
				t.Errorf("MemoryRequestBytes = %d, want 1073741824", m.MemoryRequestBytes)
			}
			if m.CPUMean != 50.5 {
				t.Errorf("CPUMean = %f, want 50.5", m.CPUMean)
			}
			if m.WorkflowID != "workflow123" {
				t.Errorf("WorkflowID = %s, want workflow123", m.WorkflowID)
			}
			break
		}
	}
	if !found {
		t.Error("Could not find MyTask shard 0 in collection")
	}
}

func TestTSVReader_ReadFromDirectory_Empty(t *testing.T) {
	// Create an empty temporary directory
	tmpDir, err := os.MkdirTemp("", "tsv_reader_test_empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	reader := NewTSVReader()
	collection, workflows, err := reader.ReadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ReadFromDirectory() error = %v", err)
	}

	if workflows != nil && len(workflows) != 0 {
		t.Errorf("Expected no workflows, got %d", len(workflows))
	}

	if collection != nil && collection.Len() != 0 {
		t.Errorf("Expected empty collection, got %d items", collection.Len())
	}
}

func TestTSVReader_ReadFromDirectory_WithInputsJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tsv_reader_test_json")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tsvContent := `task_name	shard_index	inputs_json	cpu_request
MyTask	0	{"file1":1024,"file2":2048}	2`

	tsvFile := filepath.Join(tmpDir, "workflow.tsv")
	if err := os.WriteFile(tsvFile, []byte(tsvContent), 0644); err != nil {
		t.Fatalf("Failed to write TSV file: %v", err)
	}

	reader := NewTSVReader()
	collection, _, err := reader.ReadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ReadFromDirectory() error = %v", err)
	}

	metrics := collection.Metrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.Inputs == nil {
		t.Fatal("Inputs should not be nil")
	}
	if m.Inputs["file1"] != 1024 {
		t.Errorf("Inputs[file1] = %d, want 1024", m.Inputs["file1"])
	}
	if m.Inputs["file2"] != 2048 {
		t.Errorf("Inputs[file2] = %d, want 2048", m.Inputs["file2"])
	}
}

func TestTSVReader_ReadFromDirectory_LegacyDiskPeakGB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tsv_reader_test_legacy")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Using legacy disk_peak_gb format (in GB, not bytes)
	tsvContent := `task_name	disk_peak_gb
MyTask	5.0`

	tsvFile := filepath.Join(tmpDir, "workflow.tsv")
	if err := os.WriteFile(tsvFile, []byte(tsvContent), 0644); err != nil {
		t.Fatalf("Failed to write TSV file: %v", err)
	}

	reader := NewTSVReader()
	collection, _, err := reader.ReadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ReadFromDirectory() error = %v", err)
	}

	metrics := collection.Metrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	// 5 GB = 5 * 1024^3 bytes
	expected := int64(5 * 1024 * 1024 * 1024)
	if metrics[0].DiskPeakBytes != expected {
		t.Errorf("DiskPeakBytes = %d, want %d", metrics[0].DiskPeakBytes, expected)
	}
}
