// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow/monitoring"
)

// MonitoringUseCase handles resource usage analysis from monitoring logs.
type MonitoringUseCase struct {
	fileProvider ports.FileProvider
}

// NewMonitoringUseCase creates a new monitoring use case.
func NewMonitoringUseCase(fp ports.FileProvider) *MonitoringUseCase {
	return &MonitoringUseCase{fileProvider: fp}
}

// MonitoringInput represents the input for resource analysis.
type MonitoringInput struct {
	LogPath string
}

// MonitoringOutput contains the result of resource usage analysis.
type MonitoringOutput struct {
	Report *monitoring.EfficiencyReport
}

// Execute analyzes resource usage from a monitoring log file.
func (uc *MonitoringUseCase) Execute(ctx context.Context, input MonitoringInput) (*MonitoringOutput, error) {
	// Read the file content using the injected file provider
	content, err := uc.fileProvider.Read(ctx, input.LogPath)
	if err != nil {
		return nil, err
	}

	// Parse the TSV content into metrics
	metrics, err := monitoring.ParseFromTSV(content)
	if err != nil {
		return nil, err
	}

	// Analyze the metrics and generate efficiency report
	report := metrics.Analyze()

	return &MonitoringOutput{
		Report: report,
	}, nil
}
