package cromwell

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/types"
)

func TestCost(t *testing.T) {
	content, err := os.ReadFile("../../adapters/testdata/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	meta := types.MetadataResponse{}
	err = json.Unmarshal(content, &meta)
	if err != nil {
		t.Fatal(err)
	}
	fakeCromwell := adapters.FakeCromwell{
		MetadataResponse: meta,
	}

	w := adapters.NewColoredWriter(os.Stdout)
	c := NewResourcesUsed(&fakeCromwell, w)

	resp, err := c.GetComputeUsageForPricing(meta.Calls)
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
	expectedDisk := 20.0
	if resp.PreemptSsd != expectedDisk {
		t.Errorf("Expected %v, got %v", expectedDisk, resp.PreemptSsd)
	}
	expectedDisk = 20.0
	if resp.PreemptHdd != expectedDisk {
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

func TestParseDisk(t *testing.T) {
	r1 := types.RuntimeAttributes{
		BootDiskSizeGb: "10",
		CPU:            "1",
		Disks:          "",
		Docker:         "ubuntu:20.04",
		Memory:         "2 GB",
		Preemptible:    "1",
	}
	r2 := types.RuntimeAttributes{
		BootDiskSizeGb: "10",
		CPU:            "1",
		Disks:          "local-disk 1a0 HDD",
		Docker:         "ubuntu:20.04",
		Memory:         "2 GB",
		Preemptible:    "1",
	}
	// TODO: Correct parse of multiple disks
	// r3 := RuntimeAttributes{
	// 	BootDiskSizeGb: "10",
	// 	CPU:            "1",
	// 	Disks:          "local-disk 10 HDD, work-disk 10 HDD",
	// 	Docker:         "ubuntu:20.04",
	// 	Memory:         "2 GB",
	// 	Preemptible:    "1",
	// }

	tt := []struct {
		runtime        types.RuntimeAttributes
		expectedAmount float64
		expectedType   string
		expectedErr    string
	}{
		{runtime: r1, expectedAmount: 0, expectedType: "", expectedErr: "no disks, found:"},
		{runtime: r2, expectedAmount: 0, expectedType: "", expectedErr: "strconv.ParseFloat: parsing"},
	}

	fakeCromwell := adapters.FakeCromwell{}
	w := adapters.NewColoredWriter(os.Stdout)
	c := NewResourcesUsed(&fakeCromwell, w)

	for i, test := range tt {
		amount, diskType, err := c.parseDisk(test.runtime)
		if !ErrorContains(err, test.expectedErr) {
			t.Errorf("[%d] Err is expected to be '%s', found '%s'", i, test.expectedErr, err)
		}
		if amount != test.expectedAmount {
			t.Errorf("[%d] Wrong amount of disk. Expected '%f', got '%f'", i, test.expectedAmount, amount)
		}
		if diskType != test.expectedType {
			t.Errorf("[%d] Wrong disk type. Expected '%s', got '%s'", i, test.expectedType, diskType)
		}
	}
}

func TestParseMemory(t *testing.T) {
	r1 := types.RuntimeAttributes{
		BootDiskSizeGb: "10",
		CPU:            "1",
		Disks:          "local-disk 10 HDD",
		Docker:         "ubuntu:20.04",
		Memory:         "2 GB",
		Preemptible:    "1",
	}
	r2 := types.RuntimeAttributes{
		BootDiskSizeGb: "10",
		CPU:            "1",
		Disks:          "local-disk 10 HDD",
		Docker:         "ubuntu:20.04",
		Memory:         "",
		Preemptible:    "1",
	}
	r3 := types.RuntimeAttributes{
		BootDiskSizeGb: "10",
		CPU:            "1",
		Disks:          "local-disk 10 HDD, work-disk 10 HDD",
		Docker:         "ubuntu:20.04",
		Memory:         "1a0 GB",
		Preemptible:    "1",
	}

	tt := []struct {
		runtime        types.RuntimeAttributes
		expectedAmount float64
		expectedType   string
		expectedErr    string
	}{
		{runtime: r1, expectedAmount: 2, expectedErr: ""},
		{runtime: r2, expectedAmount: 0, expectedErr: "no memory, found:"},
		{runtime: r3, expectedAmount: 0, expectedErr: "strconv.ParseFloat: parsing"},
	}

	fakeCromwell := adapters.FakeCromwell{}
	w := adapters.NewColoredWriter(os.Stdout)
	c := NewResourcesUsed(&fakeCromwell, w)
	for i, test := range tt {
		amount, err := c.parseMemory(test.runtime)
		if !ErrorContains(err, test.expectedErr) {
			t.Errorf("[%d] Err is expected to be '%s', found '%s'", i, test.expectedErr, err)
		}
		if amount != test.expectedAmount {
			t.Errorf("[%d] Wrong amount of disk. Expected '%f', got '%f'", i, test.expectedAmount, amount)
		}
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
