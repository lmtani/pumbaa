package debuginfo

import (
	"os"
	"testing"
)

func TestGetDebugInfo(t *testing.T) {
	data, err := os.ReadFile("../../../../test_data/metadata.json")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	uc := NewUsecase()
	di, err := uc.GetDebugInfo(data)
	if err != nil {
		t.Fatalf("GetDebugInfo failed: %v", err)
	}

	if di.Metadata == nil {
		t.Fatal("Expected Metadata to be set")
	}
	if di.Metadata.ID != "de8b03fd-ac06-45e8-b3c4-ef921ba0dd80" {
		t.Errorf("Expected workflow ID 'de8b03fd-ac06-45e8-b3c4-ef921ba0dd80', got '%s'", di.Metadata.ID)
	}
	if di.Root == nil {
		t.Fatal("Expected Root to be set")
	}
	if di.Root.Name != "SingleSampleGenotyping" {
		t.Errorf("Expected root name 'SingleSampleGenotyping', got '%s'", di.Root.Name)
	}
	if len(di.Visible) == 0 {
		t.Error("Expected visible nodes to be non-empty")
	}
	if di.Preemption == nil {
		t.Error("Expected preemption summary to be set")
	}
}
