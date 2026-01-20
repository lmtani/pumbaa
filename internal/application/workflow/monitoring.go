// Package workflow contains use cases for workflow management operations.
package workflow

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	workflowDomain "github.com/lmtani/pumbaa/internal/domain/workflow"
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
	Report *workflowDomain.EfficiencyReport
}

// Execute analyzes resource usage from a monitoring log file.
func (uc *MonitoringUseCase) Execute(ctx context.Context, input MonitoringInput) (*MonitoringOutput, error) {
	if input.LogPath == "" {
		return nil, application.NewInputValidationError("logPath", "is required")
	}

	// Read the file content using the injected file provider
	content, err := uc.fileProvider.Read(ctx, input.LogPath)
	if err != nil {
		return nil, application.NewUseCaseError("monitoring", "failed to read monitoring log", err)
	}

	// Parse the TSV content into metrics
	metrics, err := workflowDomain.ParseMonitoringTSV(content)
	if err != nil {
		return nil, application.NewUseCaseError("monitoring", "failed to parse monitoring log", err)
	}

	// Analyze the metrics and generate efficiency report
	report := metrics.Analyze()

	return &MonitoringOutput{
		Report: report,
	}, nil
}
