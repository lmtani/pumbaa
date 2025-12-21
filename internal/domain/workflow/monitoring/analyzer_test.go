package monitoring

import (
	"strings"
	"testing"
	"time"
)

func TestParseFromTSV_ValidData(t *testing.T) {
	content := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	mem_percent	disk_total_gb	disk_used_gb	disk_percent
2025-12-20 10:00:00	25.5	1024	4096	25.0	100	20	20.0
2025-12-20 10:00:10	50.0	2048	4096	50.0	100	25	25.0
2025-12-20 10:00:20	75.0	3072	4096	75.0	100	30	30.0`

	metrics, err := ParseFromTSV(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.DataPoints() != 3 {
		t.Errorf("expected 3 data points, got %d", metrics.DataPoints())
	}

	if metrics.MemTotal != 4096 {
		t.Errorf("expected MemTotal 4096, got %f", metrics.MemTotal)
	}

	if metrics.DiskTotal != 100 {
		t.Errorf("expected DiskTotal 100, got %f", metrics.DiskTotal)
	}

	// Check CPU values
	expectedCPU := []float64{25.5, 50.0, 75.0}
	for i, expected := range expectedCPU {
		if metrics.CPU[i] != expected {
			t.Errorf("CPU[%d]: expected %f, got %f", i, expected, metrics.CPU[i])
		}
	}

	// Check duration (20 seconds)
	expectedDuration := 20 * time.Second
	if metrics.Duration() != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, metrics.Duration())
	}
}

func TestParseFromTSV_EmptyContent(t *testing.T) {
	_, err := ParseFromTSV("")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestParseFromTSV_HeaderOnly(t *testing.T) {
	content := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	mem_percent	disk_total_gb	disk_used_gb	disk_percent`
	_, err := ParseFromTSV(content)
	if err == nil {
		t.Error("expected error for header-only content")
	}
}

func TestParseFromTSV_MalformedLines(t *testing.T) {
	content := `timestamp	cpu_percent	mem_used_mb	mem_total_mb	mem_percent	disk_total_gb	disk_used_gb	disk_percent
2025-12-20 10:00:00	25.5	1024	4096	25.0	100	20	20.0
malformed line
2025-12-20 10:00:20	75.0	3072	4096	75.0	100	30	30.0`

	metrics, err := ParseFromTSV(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 valid data points (malformed line skipped)
	if metrics.DataPoints() != 2 {
		t.Errorf("expected 2 data points, got %d", metrics.DataPoints())
	}
}

func TestAnalyze_CPUStats(t *testing.T) {
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{
			time.Now(),
			time.Now().Add(10 * time.Second),
			time.Now().Add(20 * time.Second),
		},
		CPU:       []float64{20, 40, 60},
		MemUsed:   []float64{1000, 1000, 1000},
		MemTotal:  4000,
		DiskUsed:  []float64{10, 10, 10},
		DiskTotal: 50,
	}

	report := Analyze(metrics)

	if report.CPUPeak != 60 {
		t.Errorf("expected CPUPeak 60, got %f", report.CPUPeak)
	}

	if report.CPUAvg != 40 {
		t.Errorf("expected CPUAvg 40, got %f", report.CPUAvg)
	}

	if report.CPUEfficiency != 0.4 {
		t.Errorf("expected CPUEfficiency 0.4, got %f", report.CPUEfficiency)
	}
}

func TestAnalyze_MemoryStats(t *testing.T) {
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{time.Now()},
		CPU:        []float64{50},
		MemUsed:    []float64{1000, 2000, 3000},
		MemTotal:   4000,
		DiskUsed:   []float64{10},
		DiskTotal:  50,
	}

	report := Analyze(metrics)

	if report.MemPeak != 3000 {
		t.Errorf("expected MemPeak 3000, got %f", report.MemPeak)
	}

	if report.MemAvg != 2000 {
		t.Errorf("expected MemAvg 2000, got %f", report.MemAvg)
	}

	expectedEfficiency := 3000.0 / 4000.0 // 0.75
	if report.MemEfficiency != expectedEfficiency {
		t.Errorf("expected MemEfficiency %f, got %f", expectedEfficiency, report.MemEfficiency)
	}
}

func TestAnalyze_DiskStats(t *testing.T) {
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{time.Now()},
		CPU:        []float64{50},
		MemUsed:    []float64{1000},
		MemTotal:   4000,
		DiskUsed:   []float64{10, 20, 30},
		DiskTotal:  100,
	}

	report := Analyze(metrics)

	if report.DiskPeak != 30 {
		t.Errorf("expected DiskPeak 30, got %f", report.DiskPeak)
	}

	if report.DiskAvg != 20 {
		t.Errorf("expected DiskAvg 20, got %f", report.DiskAvg)
	}

	expectedEfficiency := 30.0 / 100.0 // 0.3
	if report.DiskEfficiency != expectedEfficiency {
		t.Errorf("expected DiskEfficiency %f, got %f", expectedEfficiency, report.DiskEfficiency)
	}
}

func TestAnalyze_MemoryRecommendation(t *testing.T) {
	// Low memory efficiency with high total memory
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{time.Now()},
		CPU:        []float64{50},
		MemUsed:    []float64{500}, // Only using 500MB
		MemTotal:   8000,           // 8GB allocated
		DiskUsed:   []float64{10},
		DiskTotal:  20,
	}

	report := Analyze(metrics)

	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "Memory:") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected memory recommendation, got: %v", report.Recommendations)
	}
}

func TestAnalyze_DiskRecommendation(t *testing.T) {
	// Low disk efficiency
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{time.Now()},
		CPU:        []float64{50},
		MemUsed:    []float64{4000},
		MemTotal:   4000,
		DiskUsed:   []float64{2}, // Only using 2GB
		DiskTotal:  50,           // 50GB allocated
	}

	report := Analyze(metrics)

	found := false
	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "Disk:") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected disk recommendation, got: %v", report.Recommendations)
	}
}

func TestAnalyze_OptimizedResources(t *testing.T) {
	// Well-optimized resources
	metrics := &MonitoringMetrics{
		Timestamps: []time.Time{time.Now()},
		CPU:        []float64{80},
		MemUsed:    []float64{3500}, // 87.5% usage
		MemTotal:   4000,
		DiskUsed:   []float64{45}, // 90% usage
		DiskTotal:  50,
	}

	report := Analyze(metrics)

	if len(report.Recommendations) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(report.Recommendations))
	}

	if !strings.Contains(report.Recommendations[0], "well-optimized") {
		t.Errorf("expected optimization message, got: %v", report.Recommendations)
	}
}
