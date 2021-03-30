package commands

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"
)

func TestCost(t *testing.T) {
	file, err := ioutil.ReadFile("../sample/meta.json")
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
	expectedCPU := 11.53736111111111
	if resp.PreemptCPU != expectedCPU {
		t.Errorf("Expected %v, got %v", expectedCPU, resp.PreemptCPU)
	}
	expectedMemory := 0.23074722222222221
	if resp.PreemptMemory != expectedMemory {
		t.Errorf("Expected %v, got %v", expectedMemory, resp.PreemptMemory)
	}
}

func TestNormalizeUsePerHour(t *testing.T) {
	hoursPerCPU := normalizeUsePerHour(20, 3*time.Hour)
	if hoursPerCPU != 60.0 {
		t.Errorf("Expected %v, got %v", 60.0, hoursPerCPU)
	}
	hoursPerCPU = normalizeUsePerHour(20, time.Hour/2)
	if hoursPerCPU != 10.0 {
		t.Errorf("Expected %v, got %v", 10.0, hoursPerCPU)
	}
}
