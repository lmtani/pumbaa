package monitoring

import (
	"testing"
)

func TestParseFromTSV_DynamicHeaders(t *testing.T) {
	content := `timestamp	other_junk	cpu_percent	mem_total_mb	mem_used_mb	disk_used_gb	disk_total_gb
2025-12-21 08:00:00	xxx	25.0	4000	1000	10	50
2025-12-21 08:00:10	yyy	50.0	4000	2000	15	50`

	metrics, err := ParseFromTSV(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.DataPoints() != 2 {
		t.Errorf("expected 2 data points, got %d", metrics.DataPoints())
	}

	// Verify values are correctly mapped regardless of column order
	if metrics.CPU[0] != 25.0 {
		t.Errorf("expected first CPU 25.0, got %f", metrics.CPU[0])
	}
	if metrics.MemTotal != 4000 {
		t.Errorf("expected MemTotal 4000, got %f", metrics.MemTotal)
	}
	if metrics.DiskUsed[1] != 15 {
		t.Errorf("expected second DiskUsed 15, got %f", metrics.DiskUsed[1])
	}
}

func TestParseFromTSV_MissingRequiredColumn(t *testing.T) {
	// Missing disk_total_gb
	content := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb
2025-12-21 08:00:00	25.0	1000	4000	10`

	_, err := ParseFromTSV(content)
	if err == nil {
		t.Error("expected error due to missing required column")
	}
}

func TestParseFromTSV_MalformedFloat(t *testing.T) {
	content := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_total_gb	disk_used_gb
2025-12-21 08:00:00	invalid	1000	4000	50	10`

	metrics, err := ParseFromTSV(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid float should default to 0
	if metrics.CPU[0] != 0 {
		t.Errorf("expected CPU 0 for invalid float, got %f", metrics.CPU[0])
	}
}

func TestParseFromTSV_ExtraSpacesInHeader(t *testing.T) {
	content := `  timestamp  	 cpu_percent 	mem_used_mb	mem_total_mb	disk_total_gb	disk_used_gb  
2025-12-21 08:00:00	25.0	1000	4000	50	10`

	metrics, err := ParseFromTSV(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.DataPoints() != 1 {
		t.Errorf("expected 1 data point, got %d", metrics.DataPoints())
	}
}
