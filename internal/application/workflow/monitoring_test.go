package workflow

import (
	"context"
	"testing"
)

type mockFileProvider struct {
	readFunc func(ctx context.Context, path string) (string, error)
}

func (m *mockFileProvider) Read(ctx context.Context, path string) (string, error) {
	return m.readFunc(ctx, path)
}

func TestMonitoringUseCase_Execute(t *testing.T) {
	tsvContent := "timestamp\tcpu_percent\tmem_used_mb\tmem_total_mb\tdisk_used_gb\tdisk_total_gb\n2023-01-01 00:00:00\t10.0\t20.0\t100.0\t5.0\t50.0\n2023-01-01 00:01:00\t15.0\t25.0\t100.0\t6.0\t50.0"

	fileProvider := &mockFileProvider{
		readFunc: func(ctx context.Context, path string) (string, error) {
			return tsvContent, nil
		},
	}
	uc := NewMonitoringUseCase(fileProvider)

	input := MonitoringInput{LogPath: "gs://bucket/logs/monitoring.tsv"}
	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Report == nil {
		t.Fatal("expected a report, got nil")
	}
}
