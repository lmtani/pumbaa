package wdl

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// indexer_test.go is in internal/infrastructure/wdl/
	// so we need to go up 3 levels to get to project root
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..", "..")
}

func TestNewIndexer(t *testing.T) {
	// Get the test_data directory using absolute path
	projectRoot := getProjectRoot()
	testDir := filepath.Join(projectRoot, "test_data", "wdl")

	// Verify test directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatalf("Test directory does not exist: %s", testDir)
	}

	// Create a temp file for the index
	tmpFile, err := os.CreateTemp("", "wdl_index_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test indexer creation
	indexer, err := NewIndexer(testDir, tmpFile.Name(), true)
	if err != nil {
		t.Fatalf("Failed to create indexer: %v", err)
	}

	// Verify index was built
	idx, err := indexer.List()
	if err != nil {
		t.Fatalf("Failed to list index: %v", err)
	}

	// Should have at least one task and one workflow
	if len(idx.Tasks) == 0 {
		t.Error("Expected at least one task in the index")
	}
	if len(idx.Workflows) == 0 {
		t.Error("Expected at least one workflow in the index")
	}

	t.Logf("Indexed %d tasks, %d workflows", len(idx.Tasks), len(idx.Workflows))

	// Test search
	tasks, err := indexer.SearchTasks("hello")
	if err != nil {
		t.Fatalf("Failed to search tasks: %v", err)
	}
	t.Logf("Search 'hello' found %d tasks", len(tasks))

	// Test GetTask
	if len(idx.Tasks) > 0 {
		for name := range idx.Tasks {
			task, err := indexer.GetTask(name)
			if err != nil {
				t.Errorf("Failed to get task %s: %v", name, err)
			} else {
				t.Logf("Task %s has command with %d characters", task.Name, len(task.Command))
			}
			break
		}
	}

	// Test cache loading
	indexer2, err := NewIndexer(testDir, tmpFile.Name(), false)
	if err != nil {
		t.Fatalf("Failed to load from cache: %v", err)
	}

	idx2, _ := indexer2.List()
	if len(idx2.Tasks) != len(idx.Tasks) {
		t.Errorf("Cache mismatch: expected %d tasks, got %d", len(idx.Tasks), len(idx2.Tasks))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	projectRoot := getProjectRoot()
	testDir := filepath.Join(projectRoot, "test_data", "wdl")

	tmpFile, err := os.CreateTemp("", "wdl_index_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	indexer, err := NewIndexer(testDir, tmpFile.Name(), true)
	if err != nil {
		t.Fatalf("Failed to create indexer: %v", err)
	}

	// Search should be case-insensitive
	tasksLower, _ := indexer.SearchTasks("hello")
	tasksUpper, _ := indexer.SearchTasks("HELLO")
	tasksMixed, _ := indexer.SearchTasks("HeLLo")

	if len(tasksLower) != len(tasksUpper) || len(tasksUpper) != len(tasksMixed) {
		t.Errorf("Case-insensitive search failed: lower=%d, upper=%d, mixed=%d",
			len(tasksLower), len(tasksUpper), len(tasksMixed))
	}
}
