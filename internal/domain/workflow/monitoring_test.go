package workflow

import (
	"strings"
	"testing"
	"time"
)

const validMonitoringTSV = `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb	disk_total_gb
2024-01-01 10:00:00	25.5	2048	8192	10.5	100
2024-01-01 10:01:00	50.0	3072	8192	11.0	100
2024-01-01 10:02:00	75.0	4096	8192	12.0	100
2024-01-01 10:03:00	30.0	2560	8192	12.5	100`

func TestParseMonitoringTSV_ValidInput(t *testing.T) {
	metrics, err := ParseMonitoringTSV(validMonitoringTSV)
	if err != nil {
		t.Fatalf("ParseMonitoringTSV() error = %v", err)
	}

	if len(metrics.Timestamps) != 4 {
		t.Errorf("len(Timestamps) = %d, want 4", len(metrics.Timestamps))
	}

	if len(metrics.CPU) != 4 {
		t.Errorf("len(CPU) = %d, want 4", len(metrics.CPU))
	}

	// Check first CPU value
	if metrics.CPU[0] != 25.5 {
		t.Errorf("CPU[0] = %f, want 25.5", metrics.CPU[0])
	}

	// Check totals
	if metrics.MemTotal != 8192 {
		t.Errorf("MemTotal = %f, want 8192", metrics.MemTotal)
	}

	if metrics.DiskTotal != 100 {
		t.Errorf("DiskTotal = %f, want 100", metrics.DiskTotal)
	}
}

func TestParseMonitoringTSV_EmptyInput(t *testing.T) {
	_, err := ParseMonitoringTSV("")
	if err == nil {
		t.Error("ParseMonitoringTSV() expected error for empty input")
	}
}

func TestParseMonitoringTSV_HeaderOnly(t *testing.T) {
	headerOnly := "timestamp\tcpu_percent\tmem_used_mb\tmem_total_mb\tdisk_used_gb\tdisk_total_gb"
	_, err := ParseMonitoringTSV(headerOnly)
	if err == nil {
		t.Error("ParseMonitoringTSV() expected error for header-only input")
	}
}

func TestParseMonitoringTSV_MissingColumn(t *testing.T) {
	// Missing disk_total_gb column
	missingColumn := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb
2024-01-01 10:00:00	25.5	2048	8192	10.5`

	_, err := ParseMonitoringTSV(missingColumn)
	if err == nil {
		t.Error("ParseMonitoringTSV() expected error for missing column")
	}
	if !strings.Contains(err.Error(), "disk_total_gb") {
		t.Errorf("error should mention missing column, got: %v", err)
	}
}

func TestParseMonitoringTSV_InvalidTimestamp(t *testing.T) {
	invalidTimestamp := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb	disk_total_gb
not-a-timestamp	25.5	2048	8192	10.5	100`

	_, err := ParseMonitoringTSV(invalidTimestamp)
	if err == nil {
		t.Error("ParseMonitoringTSV() expected error for invalid timestamp")
	}
}

func TestMonitoringMetrics_DataPoints(t *testing.T) {
	metrics, _ := ParseMonitoringTSV(validMonitoringTSV)

	if metrics.DataPoints() != 4 {
		t.Errorf("DataPoints() = %d, want 4", metrics.DataPoints())
	}
}

func TestMonitoringMetrics_Duration(t *testing.T) {
	metrics, _ := ParseMonitoringTSV(validMonitoringTSV)

	expectedDuration := 3 * time.Minute // 10:00 to 10:03
	if metrics.Duration() != expectedDuration {
		t.Errorf("Duration() = %v, want %v", metrics.Duration(), expectedDuration)
	}
}

func TestMonitoringMetrics_Analyze(t *testing.T) {
	metrics, _ := ParseMonitoringTSV(validMonitoringTSV)

	report := metrics.Analyze()

	// CPU: values are 25.5, 50.0, 75.0, 30.0
	// Peak = 75.0
	// Avg = (25.5 + 50.0 + 75.0 + 30.0) / 4 = 45.125
	if report.CPU.Peak != 75.0 {
		t.Errorf("CPU.Peak = %f, want 75.0", report.CPU.Peak)
	}

	expectedCPUAvg := 45.125
	tolerance := 0.001
	if diff := report.CPU.Avg - expectedCPUAvg; diff < -tolerance || diff > tolerance {
		t.Errorf("CPU.Avg = %f, want %f", report.CPU.Avg, expectedCPUAvg)
	}

	// Mem: values are 2048, 3072, 4096, 2560 out of 8192
	// Peak = 4096
	if report.Mem.Peak != 4096 {
		t.Errorf("Mem.Peak = %f, want 4096", report.Mem.Peak)
	}

	// Disk: values are 10.5, 11.0, 12.0, 12.5 out of 100
	// Peak = 12.5
	if report.Disk.Peak != 12.5 {
		t.Errorf("Disk.Peak = %f, want 12.5", report.Disk.Peak)
	}

	// Check metadata
	if report.DataPoints != 4 {
		t.Errorf("DataPoints = %d, want 4", report.DataPoints)
	}

	if report.Duration != 3*time.Minute {
		t.Errorf("Duration = %v, want 3m", report.Duration)
	}
}

func TestNewResourceUsageStats(t *testing.T) {
	tests := []struct {
		name             string
		values           []float64
		total            float64
		useAvgForEff     bool
		wantPeak         float64
		wantAvg          float64
		wantEfficiency   float64
		efficiencyMargin float64
	}{
		{
			name:             "empty values",
			values:           []float64{},
			total:            100,
			useAvgForEff:     false,
			wantPeak:         0,
			wantAvg:          0,
			wantEfficiency:   0,
			efficiencyMargin: 0.001,
		},
		{
			name:             "CPU usage (use avg for efficiency)",
			values:           []float64{25, 50, 75, 50},
			total:            100, // 100%
			useAvgForEff:     true,
			wantPeak:         75,
			wantAvg:          50,  // (25+50+75+50)/4 = 50
			wantEfficiency:   0.5, // 50/100 = 0.5
			efficiencyMargin: 0.001,
		},
		{
			name:             "Memory usage (use peak for efficiency)",
			values:           []float64{2000, 3000, 4000},
			total:            8000,
			useAvgForEff:     false,
			wantPeak:         4000,
			wantAvg:          3000,
			wantEfficiency:   0.5, // 4000/8000 = 0.5
			efficiencyMargin: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewResourceUsageStats(tt.values, tt.total, tt.useAvgForEff)

			if stats.Peak != tt.wantPeak {
				t.Errorf("Peak = %f, want %f", stats.Peak, tt.wantPeak)
			}
			if diff := stats.Avg - tt.wantAvg; diff < -0.001 || diff > 0.001 {
				t.Errorf("Avg = %f, want %f", stats.Avg, tt.wantAvg)
			}
			if diff := stats.Efficiency - tt.wantEfficiency; diff < -tt.efficiencyMargin || diff > tt.efficiencyMargin {
				t.Errorf("Efficiency = %f, want %f", stats.Efficiency, tt.wantEfficiency)
			}
		})
	}
}

func TestAnalyze_GeneratesRecommendations(t *testing.T) {
	// Create metrics with very low memory efficiency (should trigger recommendation)
	lowEfficiencyTSV := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb	disk_total_gb
2024-01-01 10:00:00	50.0	500	16000	5	100
2024-01-01 10:01:00	50.0	600	16000	5	100`

	metrics, err := ParseMonitoringTSV(lowEfficiencyTSV)
	if err != nil {
		t.Fatalf("ParseMonitoringTSV() error = %v", err)
	}

	report := metrics.Analyze()

	// Should have at least one recommendation about memory
	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "Memory") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected memory recommendation for low efficiency, got: " + strings.Join(report.Recommendations, "; "))
	}
}

func TestAnalyze_WellOptimizedNoRecommendations(t *testing.T) {
	// Create metrics with good efficiency
	goodEfficiencyTSV := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	disk_used_gb	disk_total_gb
2024-01-01 10:00:00	80.0	7000	8000	90	100
2024-01-01 10:01:00	85.0	7500	8000	92	100`

	metrics, err := ParseMonitoringTSV(goodEfficiencyTSV)
	if err != nil {
		t.Fatalf("ParseMonitoringTSV() error = %v", err)
	}

	report := metrics.Analyze()

	// Should have the "well-optimized" message
	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "well-optimized") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'well-optimized' recommendation, got: " + strings.Join(report.Recommendations, "; "))
	}
}
