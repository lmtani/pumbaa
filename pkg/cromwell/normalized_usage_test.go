package cromwell

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestCost(t *testing.T) {
	file, err := ioutil.ReadFile("mocks/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	meta := MetadataResponse{}
	err = json.Unmarshal(file, &meta)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := GetComputeUsageForPricing(meta.Calls)
	if err != nil {
		t.Error(err)
	}
	// Preemptible setup
	expectedCPU := 720.0 * 2
	if resp.PreemptCPU != expectedCPU {
		t.Errorf("Expected %v, got %v", expectedCPU, resp.PreemptCPU)
	}
	expectedMemory := 1440.0 * 2
	if resp.PreemptMemory != expectedMemory {
		t.Errorf("Expected %v, got %v", expectedMemory, resp.PreemptMemory)
	}
	expectedDisk := 20.0 * 2
	if resp.PreemptSsd != expectedDisk {
		t.Errorf("Expected %v, got %v", expectedDisk, resp.PreemptSsd)
	}
	// Normal setup
	expectedCPU = 720.0
	if resp.CPU != expectedCPU {
		t.Errorf("Expected %v, got %v", expectedCPU, resp.PreemptCPU)
	}
	expectedMemory = 1440.0
	if resp.Memory != expectedMemory {
		t.Errorf("Expected %v, got %v", expectedMemory, resp.PreemptMemory)
	}
	expectedDisk = 20.0
	if resp.Ssd != expectedDisk {
		t.Errorf("Expected %v, got %v", expectedDisk, resp.PreemptSsd)
	}
	expectedHours := 1440.0 + 720.0
	if resp.TotalTime.Hours() != expectedHours {
		t.Errorf("Expected %v, got %v", expectedHours, resp.TotalTime.Hours())
	}
}
