package commands

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func TestCost(t *testing.T) {
	file, err := ioutil.ReadFile("../sample/meta.json")
	if err != nil {
		t.Fatal(err)
	}
	meta := cromwell.MetadataResponse{}
	err = json.Unmarshal(file, &meta)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := GetComputeUsageForPricing(meta.Calls)
	if err != nil {
		t.Error(err)
	}
	expectedCPU := 720.0
	if resp.PreemptCPU != expectedCPU {
		t.Errorf("Expected %v, got %v", expectedCPU, resp.PreemptCPU)
	}
	expectedMemory := 1440.0
	if resp.PreemptMemory != expectedMemory {
		t.Errorf("Expected %v, got %v", expectedMemory, resp.PreemptMemory)
	}
	expectedDisk := 20.0
	if resp.PreemptSsd != expectedDisk {
		t.Errorf("Expected %v, got %v", expectedDisk, resp.PreemptSsd)
	}
	expectedHours := 720.0
	if resp.TotalTime.Hours() != expectedHours {
		t.Errorf("Expected %v, got %v", expectedHours, resp.TotalTime.Hours())
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
