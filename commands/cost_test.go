package commands

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"
)

func TestCost(t *testing.T) {
	file, _ := ioutil.ReadFile("./metadata.json")
	meta := MetadataResponse{}
	_ = json.Unmarshal(file, &meta)

	resp, _ := GetComputeCost(meta.Calls)
	if resp != 0.1 {
		t.Errorf("Expected %v, got %v", 0.1, resp)
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
